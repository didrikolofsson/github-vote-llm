package services

import (
	"context"
	"errors"
	"sort"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPortalNotFound = errors.New("portal not found or not public")

// PortalFeatureDTO is the public-facing shape for a feature card.
type PortalFeatureDTO struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Area        *string `json:"area"`
	VoteCount   int64   `json:"vote_count"`
	HasVoted    bool    `json:"has_voted"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// PortalCommentDTO is the public-facing shape for a comment.
type PortalCommentDTO struct {
	ID         int64  `json:"id"`
	FeatureID  int64  `json:"feature_id"`
	Body       string `json:"body"`
	AuthorName string `json:"author_name"`
	CreatedAt  string `json:"created_at"`
}

// PortalPageDTO is the full data payload for a public portal page.
type PortalPageDTO struct {
	OrgSlug         string             `json:"org_slug"`
	RepoOwner       string             `json:"repo_owner"`
	RepoName        string             `json:"repo_name"`
	Proposals       []PortalFeatureDTO `json:"proposals"`        // status: open
	Planned         []PortalFeatureDTO `json:"planned"`          // status: planned
	InProgress      []PortalFeatureDTO `json:"in_progress"`      // status: in_progress
	Done            []PortalFeatureDTO `json:"done"`             // status: done
	RecentlyShipped []PortalFeatureDTO `json:"recently_shipped"` // top 10 done by updated_at
}

type PortalService interface {
	GetPortalPage(ctx context.Context, orgSlug, repoName, voterToken string) (*PortalPageDTO, error)
	ToggleVote(ctx context.Context, orgSlug, repoName string, featureID int64, voterToken string) (int64, error)
	ListComments(ctx context.Context, orgSlug, repoName string, featureID int64) ([]PortalCommentDTO, error)
	CreateComment(ctx context.Context, orgSlug, repoName string, featureID int64, body, authorName string) (*PortalCommentDTO, error)
}

type PortalServiceImpl struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func NewPortalService(db *pgxpool.Pool, q *store.Queries) PortalService {
	return &PortalServiceImpl{db: db, q: q}
}

// resolvePublicRepo looks up a repository by org slug + repo name and ensures it is public.
func (s *PortalServiceImpl) resolvePublicRepo(ctx context.Context, orgSlug, repoName string) (store.Repository, error) {
	repo, err := s.q.GetPublicRepositoryByOrgAndName(ctx, store.GetPublicRepositoryByOrgAndNameParams{
		Slug: orgSlug,
		Name: repoName,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Repository{}, ErrPortalNotFound
	}
	return repo, err
}

func (s *PortalServiceImpl) GetPortalPage(ctx context.Context, orgSlug, repoName, voterToken string) (*PortalPageDTO, error) {
	repo, err := s.resolvePublicRepo(ctx, orgSlug, repoName)
	if err != nil {
		return nil, err
	}

	rows, err := s.q.ListFeaturesForPortal(ctx, repo.ID)
	if err != nil {
		return nil, err
	}

	const recentlyShippedLimit = 10

	proposals := []PortalFeatureDTO{}
	planned := []PortalFeatureDTO{}
	inProgress := []PortalFeatureDTO{}
	done := []PortalFeatureDTO{}

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

		switch row.Status {
		case store.FeatureStatusOpen:
			proposals = append(proposals, dto)
		case store.FeatureStatusPlanned:
			planned = append(planned, dto)
		case store.FeatureStatusInProgress:
			inProgress = append(inProgress, dto)
		case store.FeatureStatusDone:
			done = append(done, dto)
		}
	}

	// Recently shipped: top N done features sorted by updated_at desc.
	recentlyShipped := make([]PortalFeatureDTO, len(done))
	copy(recentlyShipped, done)
	sort.Slice(recentlyShipped, func(i, j int) bool {
		return recentlyShipped[i].UpdatedAt > recentlyShipped[j].UpdatedAt
	})
	if len(recentlyShipped) > recentlyShippedLimit {
		recentlyShipped = recentlyShipped[:recentlyShippedLimit]
	}

	return &PortalPageDTO{
		OrgSlug:         orgSlug,
		RepoOwner:       repo.Owner,
		RepoName:        repo.Name,
		Proposals:       proposals,
		Planned:         planned,
		InProgress:      inProgress,
		Done:            done,
		RecentlyShipped: recentlyShipped,
	}, nil
}

func (s *PortalServiceImpl) ToggleVote(ctx context.Context, orgSlug, repoName string, featureID int64, voterToken string) (int64, error) {
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

func (s *PortalServiceImpl) ListComments(ctx context.Context, orgSlug, repoName string, featureID int64) ([]PortalCommentDTO, error) {
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

	out := make([]PortalCommentDTO, len(rows))
	for i, c := range rows {
		out[i] = PortalCommentDTO{
			ID:         c.ID,
			FeatureID:  c.FeatureID,
			Body:       c.Body,
			AuthorName: c.AuthorName,
			CreatedAt:  c.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return out, nil
}

func (s *PortalServiceImpl) CreateComment(ctx context.Context, orgSlug, repoName string, featureID int64, body, authorName string) (*PortalCommentDTO, error) {
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

	dto := PortalCommentDTO{
		ID:         c.ID,
		FeatureID:  c.FeatureID,
		Body:       c.Body,
		AuthorName: c.AuthorName,
		CreatedAt:  c.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
	return &dto, nil
}

func portalFeatureFromRow(row store.ListFeaturesForPortalRow, hasVoted bool) PortalFeatureDTO {
	return PortalFeatureDTO{
		ID:          row.ID,
		Title:       row.Title,
		Description: row.Description,
		Status:      string(row.Status),
		Area:        row.Area,
		VoteCount:   row.VoteCount,
		HasVoted:    hasVoted,
		CreatedAt:   row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}
