package dtos

import "time"

type Organization struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrganizationMember struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

type OrganizationWithMembers struct {
	Organization
	Members []OrganizationMember `json:"members"`
}

type OrganizationMemberRole string

const (
	OrganizationMemberRoleOwner  OrganizationMemberRole = "owner"
	OrganizationMemberRoleMember OrganizationMemberRole = "member"
)
