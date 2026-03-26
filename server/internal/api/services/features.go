package services

import (
	"context"
	"errors"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrFeatureNotFound      = errors.New("feature not found")
	ErrVoteAlreadyExists    = errors.New("already voted for this feature")
	ErrDependencyNotFound   = errors.New("dependency not found")
	ErrDependencyExists     = errors.New("dependency already exists")
)

type FeatureDTO struct {
	ID            int64    `json:"id"`
	RepositoryID  int64    `json:"repository_id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Status        string   `json:"status"`
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

type RoadmapDTO struct {
	Features     []FeatureDTO         `json:"features"`
	Dependencies []FeatureDependency  `json:"dependencies"`
}

type FeatureDependency struct {
	FeatureID int64 `json:"feature_id"`
	DependsOn int64 `json:"depends_on"`
}

type FeaturesService interface {
	ListFeatures(ctx context.Context, repoID int64) ([]FeatureDTO, error)
	GetFeature(ctx context.Context, featureID int64) (*FeatureDTO, error)
	CreateFeature(ctx context.Context, repoID int64, title, description string) (*FeatureDTO, error)
	UpdateStatus(ctx context.Context, featureID int64, status store.FeatureStatus) (*FeatureDTO, error)
	UpdateArea(ctx context.Context, featureID int64, area *string) (*FeatureDTO, error)
	UpdatePosition(ctx context.Context, featureID int64, x, y *float64, locked bool) (*FeatureDTO, error)
	GetRoadmap(ctx context.Context, repoID int64) (*RoadmapDTO, error)
	AddDependency(ctx context.Context, featureID, dependsOn int64) error
	RemoveDependency(ctx context.Context, featureID, dependsOn int64) error
	ToggleVote(ctx context.Context, featureID int64, voterToken string) (int64, error)
	ListComments(ctx context.Context, featureID int64) ([]FeatureCommentDTO, error)
	CreateComment(ctx context.Context, featureID int64, body, authorName string) (*FeatureCommentDTO, error)
}

type FeaturesServiceImpl struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func NewFeaturesService(db *pgxpool.Pool, q *store.Queries) FeaturesService {
	return &FeaturesServiceImpl{db: db, q: q}
}

func (s *FeaturesServiceImpl) ListFeatures(ctx context.Context, repoID int64) ([]FeatureDTO, error) {
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

func (s *FeaturesServiceImpl) GetFeature(ctx context.Context, featureID int64) (*FeatureDTO, error) {
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

func (s *FeaturesServiceImpl) CreateFeature(ctx context.Context, repoID int64, title, description string) (*FeatureDTO, error) {
	f, err := s.q.CreateFeature(ctx, store.CreateFeatureParams{
		RepositoryID: repoID,
		Title:        title,
		Description:  description,
	})
	if err != nil {
		return nil, err
	}
	dto, err := s.toDTO(ctx, f)
	if err != nil {
		return nil, err
	}
	return &dto, nil
}

func (s *FeaturesServiceImpl) UpdateStatus(ctx context.Context, featureID int64, status store.FeatureStatus) (*FeatureDTO, error) {
	f, err := s.q.UpdateFeatureStatus(ctx, store.UpdateFeatureStatusParams{ID: featureID, Status: status})
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

func (s *FeaturesServiceImpl) UpdateArea(ctx context.Context, featureID int64, area *string) (*FeatureDTO, error) {
	f, err := s.q.UpdateFeatureArea(ctx, store.UpdateFeatureAreaParams{ID: featureID, Area: area})
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

func (s *FeaturesServiceImpl) UpdatePosition(ctx context.Context, featureID int64, x, y *float64, locked bool) (*FeatureDTO, error) {
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

func (s *FeaturesServiceImpl) GetRoadmap(ctx context.Context, repoID int64) (*RoadmapDTO, error) {
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

func (s *FeaturesServiceImpl) AddDependency(ctx context.Context, featureID, dependsOn int64) error {
	return s.q.AddFeatureDependency(ctx, store.AddFeatureDependencyParams{
		FeatureID: featureID,
		DependsOn: dependsOn,
	})
}

func (s *FeaturesServiceImpl) RemoveDependency(ctx context.Context, featureID, dependsOn int64) error {
	return s.q.RemoveFeatureDependency(ctx, store.RemoveFeatureDependencyParams{
		FeatureID: featureID,
		DependsOn: dependsOn,
	})
}

func (s *FeaturesServiceImpl) ToggleVote(ctx context.Context, featureID int64, voterToken string) (int64, error) {
	_, err := s.q.GetFeatureVote(ctx, store.GetFeatureVoteParams{
		FeatureID:  featureID,
		VoterToken: voterToken,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// No vote yet — add it
		if _, err := s.q.AddFeatureVote(ctx, store.AddFeatureVoteParams{
			FeatureID:  featureID,
			VoterToken: voterToken,
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
	return count, nil
}

func (s *FeaturesServiceImpl) ListComments(ctx context.Context, featureID int64) ([]FeatureCommentDTO, error) {
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

func (s *FeaturesServiceImpl) CreateComment(ctx context.Context, featureID int64, body, authorName string) (*FeatureCommentDTO, error) {
	c, err := s.q.CreateFeatureComment(ctx, store.CreateFeatureCommentParams{
		FeatureID:  featureID,
		Body:       body,
		AuthorName: authorName,
	})
	if err != nil {
		return nil, err
	}
	dto := storeCommentToDTO(c)
	return &dto, nil
}

func (s *FeaturesServiceImpl) toDTO(ctx context.Context, f store.Feature) (FeatureDTO, error) {
	count, err := s.q.CountFeatureVotes(ctx, f.ID)
	if err != nil {
		return FeatureDTO{}, err
	}
	return FeatureDTO{
		ID:            f.ID,
		RepositoryID:  f.RepositoryID,
		Title:         f.Title,
		Description:   f.Description,
		Status:        string(f.Status),
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
