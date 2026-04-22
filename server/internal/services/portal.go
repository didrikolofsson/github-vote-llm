package services

import (
	"context"
	"errors"
	"sort"

	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPortalNotFound = errors.New("portal not found or not public")

type PortalService struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func NewPortalService(db *pgxpool.Pool, q *store.Queries) *PortalService {
	return &PortalService{db: db, q: q}
}

// resolvePublicRepo looks up a repository by org slug + repo name and ensures it is public.
func (s *PortalService) resolvePublicRepo(ctx context.Context, orgSlug, repoName string) (store.Repository, error) {
	repo, err := s.q.GetPublicRepositoryByOrgAndName(ctx, store.GetPublicRepositoryByOrgAndNameParams{
		Slug: orgSlug,
		Name: repoName,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Repository{}, ErrPortalNotFound
	}
	return repo, err
}

func (s *PortalService) GetPortalPage(ctx context.Context, orgSlug, repoName, voterToken string) (*dtos.PortalPageDTO, error) {
	repo, err := s.resolvePublicRepo(ctx, orgSlug, repoName)
	if err != nil {
		return nil, err
	}

	rows, err := s.q.ListFeaturesForPortal(ctx, repo.ID)
	if err != nil {
		return nil, err
	}

	const recentlyShippedLimit = 10

	requests := []dtos.PortalFeatureDTO{}
	pending := []dtos.PortalFeatureDTO{}
	inProgress := []dtos.PortalFeatureDTO{}
	done := []dtos.PortalFeatureDTO{}

	for _, row := range rows {
		hasVoted := false
		if voterToken != "" {
			_, voteErr := s.q.GetFeatureVote(ctx, store.GetFeatureVoteParams{
				FeatureID:  row.ID,
				VoterToken: voterToken,
			})
			hasVoted = voteErr == nil
		}

		dto := portalFeatureFromRow(row, hasVoted)

		if row.ReviewStatus == store.ReviewStatusTypePending {
			requests = append(requests, dto)
			continue
		}

		switch row.BuildStatus.BuildStatusType {
		case store.BuildStatusTypePending:
			pending = append(pending, dto)
		case store.BuildStatusTypeInProgress:
			inProgress = append(inProgress, dto)
		case store.BuildStatusTypeDone:
			done = append(done, dto)
		}
	}

	// Recently shipped: top N done features sorted by updated_at desc.
	recentlyShipped := make([]dtos.PortalFeatureDTO, len(done))
	copy(recentlyShipped, done)
	sort.Slice(recentlyShipped, func(i, j int) bool {
		return recentlyShipped[i].UpdatedAt > recentlyShipped[j].UpdatedAt
	})
	if len(recentlyShipped) > recentlyShippedLimit {
		recentlyShipped = recentlyShipped[:recentlyShippedLimit]
	}

	return &dtos.PortalPageDTO{
		OrgSlug:    orgSlug,
		RepoOwner:  repo.Owner,
		RepoName:   repo.Name,
		RepoID:     repo.ID,
		Requests:   requests,
		Pending:    pending,
		InProgress: inProgress,
		Done:       done,
	}, nil
}

func (s *PortalService) ToggleVote(ctx context.Context, orgSlug, repoName string, featureID int64, voterToken, reason string, urgency store.NullVoteUrgencyType) (int64, error) {
	repo, err := s.resolvePublicRepo(ctx, orgSlug, repoName)
	if err != nil {
		return 0, err
	}

	// Ensure feature belongs to this repo.
	feature, err := s.q.GetFeature(ctx, featureID)
	if errors.Is(err, pgx.ErrNoRows) || feature.RepositoryID != repo.ID {
		return 0, ErrPortalNotFound
	}
	if err != nil {
		return 0, err
	}

	// Try to add; on unique constraint violation, remove instead.
	_, err = s.q.AddFeatureVote(ctx, store.AddFeatureVoteParams{
		FeatureID:  featureID,
		VoterToken: voterToken,
		Reason:     reason,
		Urgency:    urgency,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Already voted — remove the vote.
			if removeErr := s.q.RemoveFeatureVote(ctx, store.RemoveFeatureVoteParams{
				FeatureID:  featureID,
				VoterToken: voterToken,
			}); removeErr != nil {
				return 0, removeErr
			}
		} else {
			return 0, err
		}
	}

	return s.q.CountFeatureVotes(ctx, featureID)
}

func (s *PortalService) ListComments(ctx context.Context, orgSlug, repoName string, featureID int64) ([]dtos.PortalCommentDTO, error) {
	repo, err := s.resolvePublicRepo(ctx, orgSlug, repoName)
	if err != nil {
		return nil, err
	}

	feature, err := s.q.GetFeature(ctx, featureID)
	if errors.Is(err, pgx.ErrNoRows) || feature.RepositoryID != repo.ID {
		return nil, ErrPortalNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.q.ListFeatureComments(ctx, featureID)
	if err != nil {
		return nil, err
	}

	out := make([]dtos.PortalCommentDTO, len(rows))
	for i, c := range rows {
		out[i] = dtos.PortalCommentDTO{
			ID:         c.ID,
			FeatureID:  c.FeatureID,
			Body:       c.Body,
			AuthorName: c.AuthorName,
			CreatedAt:  c.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return out, nil
}

func (s *PortalService) CreateComment(ctx context.Context, orgSlug, repoName string, featureID int64, body, authorName string) (*dtos.PortalCommentDTO, error) {
	repo, err := s.resolvePublicRepo(ctx, orgSlug, repoName)
	if err != nil {
		return nil, err
	}

	feature, err := s.q.GetFeature(ctx, featureID)
	if errors.Is(err, pgx.ErrNoRows) || feature.RepositoryID != repo.ID {
		return nil, ErrPortalNotFound
	}
	if err != nil {
		return nil, err
	}

	if authorName == "" {
		authorName = "Anonymous"
	}

	c, err := s.q.CreateFeatureComment(ctx, store.CreateFeatureCommentParams{
		FeatureID:  featureID,
		Body:       body,
		AuthorName: authorName,
	})
	if err != nil {
		return nil, err
	}

	dto := dtos.PortalCommentDTO{
		ID:         c.ID,
		FeatureID:  c.FeatureID,
		Body:       c.Body,
		AuthorName: c.AuthorName,
		CreatedAt:  c.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
	return &dto, nil
}

func portalFeatureFromRow(row store.ListFeaturesForPortalRow, hasVoted bool) dtos.PortalFeatureDTO {
	var buildStatus *store.BuildStatusType
	if row.BuildStatus.Valid {
		buildStatus = &row.BuildStatus.BuildStatusType
	}
	return dtos.PortalFeatureDTO{
		ID:           row.ID,
		Title:        row.Title,
		Description:  row.Description,
		ReviewStatus: row.ReviewStatus,
		BuildStatus:  buildStatus,
		Area:         row.Area,
		VoteCount:    row.VoteCount,
		HasVoted:     hasVoted,
		CreatedAt:    row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}
