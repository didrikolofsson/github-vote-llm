package dtos

type Repository struct {
	ID           int64  `json:"id"`
	Owner        string `json:"owner"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	PortalPublic bool   `json:"portal_public"`
	CreatedAt    string `json:"created_at,omitempty"`
}

type RepoStatus string

const (
	RepoStatusActive RepoStatus = "active"
	RepoStatusIdle   RepoStatus = "idle"
)

type RepoMeta struct {
	ID              int64      `json:"id"`
	Description     string     `json:"description"`
	Features        int64      `json:"features"`
	Implementations int64      `json:"implementations"`
	Status          RepoStatus `json:"status"`
}
