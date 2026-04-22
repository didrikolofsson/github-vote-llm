package services

import (
	"context"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrFeatureNotFound    = errors.New("feature not found")
	ErrVoteAlreadyExists  = errors.New("already voted for this feature")
	ErrDependencyNotFound = errors.New("dependency not found")
	ErrDependencyExists   = errors.New("dependency already exists")
)

type FeatureDTO struct {
	ID            int64    `json:"id"`
	RepositoryID  int64    `json:"repository_id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	ReviewStatus  string   `json:"review_status"`
	BuildStatus   *string  `json:"build_status"`
	Area          *string  `json:"area"`
	RoadmapX      *float64 `json:"roadmap_x"`
	RoadmapY      *float64 `json:"roadmap_y"`
	RoadmapLocked bool     `json:"roadmap_locked"`
	VoteCount     int64    `json:"vote_count"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type FeatureCommentDTO struct {
	ID         int64  `json:"id"`
	FeatureID  int64  `json:"feature_id"`
	Body       string `json:"body"`
	AuthorName string `json:"author_name"`
	CreatedAt  string `json:"created_at"`
}

type VoteSignalDTO struct {
	ID         int64   `json:"id"`
	FeatureID  int64   `json:"feature_id"`
	VoterToken string  `json:"voter_token"`
	Reason     string  `json:"reason"`
	Urgency    *string `json:"urgency"`
	CreatedAt  string  `json:"created_at"`
}

type RoadmapDTO struct {
	Features     []FeatureDTO        `json:"features"`
	Dependencies []FeatureDependency `json:"dependencies"`
}

type FeatureDependency struct {
	FeatureID int64 `json:"feature_id"`
	DependsOn int64 `json:"depends_on"`
}

type PatchFeatureParams struct {
	Title        *string
	Description  *string
	ReviewStatus *store.ReviewStatusType
	BuildStatus  *store.BuildStatusType
	Area         *string
}

type FeaturesService struct {
	db *pgxpool.Pool
	q  *store.Queries
	h  hub.Hub
}

func NewFeaturesService(db *pgxpool.Pool, q *store.Queries, h hub.Hub) *FeaturesService {
	return &FeaturesService{db: db, q: q, h: h}
}

func (s *FeaturesService) ListFeatures(ctx context.Context, repoID int64) ([]FeatureDTO, error) {
	rows, err := s.q.ListFeatures(ctx, repoID)
	if err != nil {
		return nil, err
	}
	out := make([]FeatureDTO, len(rows))
	for i, f := range rows {
		dto, err := s.toDTO(ctx, f)
		if err != nil {
			return nil, err
		}
		out[i] = dto
	}
	return out, nil
}

func (s *FeaturesService) GetFeature(ctx context.Context, featureID int64) (*FeatureDTO, error) {
	f, err := s.q.GetFeature(ctx, featureID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFeatureNotFound
	}
	if err != nil {
		return nil, err
	}
	dto, err := s.toDTO(ctx, f)
	if err != nil {
		return nil, err
	}
	return &dto, nil
}

func (s *FeaturesService) CreateFeature(ctx context.Context, repoID int64, title, description string) (*FeatureDTO, error) {
	f, err := s.q.CreateFeature(ctx, store.CreateFeatureParams{
		RepositoryID: repoID,
		Title:        title,
		Description:  description,
		ReviewStatus: store.ReviewStatusTypeApproved,
		BuildStatus: store.NullBuildStatusType{
			BuildStatusType: store.BuildStatusTypePending,
			Valid:           true,
		},
	})
	if err != nil {
		return nil, err
	}
	dto, err := s.toDTO(ctx, f)
	if err != nil {
		return nil, err
	}
	s.h.Publish(f.RepositoryID, hub.EventFeatureCreated)
	return &dto, nil
}

func (s *FeaturesService) DeleteFeature(ctx context.Context, featureID int64) error {
	f, err := s.q.GetFeature(ctx, featureID)
	if err != nil {
		return err
	}
	if err := s.q.DeleteFeature(ctx, featureID); err != nil {
		return err
	}
	s.h.Publish(f.RepositoryID, hub.EventFeatureUpdated)
	return nil
}

func (s *FeaturesService) PatchFeature(ctx context.Context, featureID int64, p PatchFeatureParams) (*FeatureDTO, error) {
	nullReviewStatus := store.NullReviewStatusType{}
	if p.ReviewStatus != nil {
		nullReviewStatus = store.NullReviewStatusType{ReviewStatusType: *p.ReviewStatus, Valid: true}
	}
	nullBuildStatus := store.NullBuildStatusType{}
	if p.BuildStatus != nil {
		nullBuildStatus = store.NullBuildStatusType{BuildStatusType: *p.BuildStatus, Valid: true}
	}
	f, err := s.q.PatchFeature(ctx, store.PatchFeatureParams{
		ID:           featureID,
		Title:        p.Title,
		Description:  p.Description,
		ReviewStatus: nullReviewStatus,
		BuildStatus:  nullBuildStatus,
		Area:         p.Area,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFeatureNotFound
	}
	if err != nil {
		return nil, err
	}
	dto, err := s.toDTO(ctx, f)
	if err != nil {
		return nil, err
	}
	s.h.Publish(f.RepositoryID, hub.EventFeatureUpdated)
	return &dto, nil
}

func (s *FeaturesService) UpdatePosition(ctx context.Context, featureID int64, x, y *float64, locked bool) (*FeatureDTO, error) {
	f, err := s.q.UpdateFeaturePosition(ctx, store.UpdateFeaturePositionParams{
		ID:            featureID,
		RoadmapX:      x,
		RoadmapY:      y,
		RoadmapLocked: locked,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFeatureNotFound
	}
	if err != nil {
		return nil, err
	}
	dto, err := s.toDTO(ctx, f)
	if err != nil {
		return nil, err
	}
	return &dto, nil
}

func (s *FeaturesService) GetRoadmap(ctx context.Context, repoID int64) (*RoadmapDTO, error) {
	features, err := s.ListFeatures(ctx, repoID)
	if err != nil {
		return nil, err
	}
	depRows, err := s.q.ListFeatureDependenciesForRepository(ctx, repoID)
	if err != nil {
		return nil, err
	}
	deps := make([]FeatureDependency, len(depRows))
	for i, d := range depRows {
		deps[i] = FeatureDependency{FeatureID: d.FeatureID, DependsOn: d.DependsOn}
	}
	return &RoadmapDTO{Features: features, Dependencies: deps}, nil
}

func (s *FeaturesService) AddDependency(ctx context.Context, featureID, dependsOn int64) error {
	if err := s.q.AddFeatureDependency(ctx, store.AddFeatureDependencyParams{
		FeatureID: featureID,
		DependsOn: dependsOn,
	}); err != nil {
		return err
	}
	return nil
}

func (s *FeaturesService) RemoveDependency(ctx context.Context, featureID, dependsOn int64) error {
	if err := s.q.RemoveFeatureDependency(ctx, store.RemoveFeatureDependencyParams{
		FeatureID: featureID,
		DependsOn: dependsOn,
	}); err != nil {
		return err
	}
	return nil
}

func (s *FeaturesService) ToggleVote(ctx context.Context, featureID int64, voterToken, reason string, urgency store.NullVoteUrgencyType) (int64, error) {
	_, err := s.q.GetFeatureVote(ctx, store.GetFeatureVoteParams{
		FeatureID:  featureID,
		VoterToken: voterToken,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// No vote yet — add it
		if _, err := s.q.AddFeatureVote(ctx, store.AddFeatureVoteParams{
			FeatureID:  featureID,
			VoterToken: voterToken,
			Reason:     reason,
			Urgency:    urgency,
		}); err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	} else {
		// Already voted — remove it
		if err := s.q.RemoveFeatureVote(ctx, store.RemoveFeatureVoteParams{
			FeatureID:  featureID,
			VoterToken: voterToken,
		}); err != nil {
			return 0, err
		}
	}
	count, err := s.q.CountFeatureVotes(ctx, featureID)
	if err != nil {
		return 0, err
	}
	s.h.Publish(featureID, hub.EventFeatureUpdated)
	return count, nil
}

func (s *FeaturesService) ListVoteSignals(ctx context.Context, featureID int64) ([]VoteSignalDTO, error) {
	rows, err := s.q.ListFeatureVotesWithSignals(ctx, featureID)
	if err != nil {
		return nil, err
	}
	out := make([]VoteSignalDTO, len(rows))
	for i, v := range rows {
		var urgency *string
		if v.Urgency.Valid {
			s := string(v.Urgency.VoteUrgencyType)
			urgency = &s
		}
		out[i] = VoteSignalDTO{
			ID:         v.ID,
			FeatureID:  v.FeatureID,
			VoterToken: v.VoterToken,
			Reason:     v.Reason,
			Urgency:    urgency,
			CreatedAt:  v.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return out, nil
}

func (s *FeaturesService) ListComments(ctx context.Context, featureID int64) ([]FeatureCommentDTO, error) {
	rows, err := s.q.ListFeatureComments(ctx, featureID)
	if err != nil {
		return nil, err
	}
	out := make([]FeatureCommentDTO, len(rows))
	for i, c := range rows {
		out[i] = storeCommentToDTO(c)
	}
	return out, nil
}

func (s *FeaturesService) CreateComment(ctx context.Context, featureID int64, body, authorName string) (*FeatureCommentDTO, error) {
	c, err := s.q.CreateFeatureComment(ctx, store.CreateFeatureCommentParams{
		FeatureID:  featureID,
		Body:       body,
		AuthorName: authorName,
	})
	if err != nil {
		return nil, err
	}
	dto := storeCommentToDTO(c)
	s.h.Publish(featureID, hub.EventFeatureUpdated)
	return &dto, nil
}

func (s *FeaturesService) toDTO(ctx context.Context, f store.Feature) (FeatureDTO, error) {
	count, err := s.q.CountFeatureVotes(ctx, f.ID)
	if err != nil {
		return FeatureDTO{}, err
	}
	var buildStatus *string
	if f.BuildStatus.Valid {
		s := string(f.BuildStatus.BuildStatusType)
		buildStatus = &s
	}
	return FeatureDTO{
		ID:            f.ID,
		RepositoryID:  f.RepositoryID,
		Title:         f.Title,
		Description:   f.Description,
		ReviewStatus:  string(f.ReviewStatus),
		BuildStatus:   buildStatus,
		Area:          f.Area,
		RoadmapX:      f.RoadmapX,
		RoadmapY:      f.RoadmapY,
		RoadmapLocked: f.RoadmapLocked,
		VoteCount:     count,
		CreatedAt:     f.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     f.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func storeCommentToDTO(c store.FeatureComment) FeatureCommentDTO {
	return FeatureCommentDTO{
		ID:         c.ID,
		FeatureID:  c.FeatureID,
		Body:       c.Body,
		AuthorName: c.AuthorName,
		CreatedAt:  c.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}
