package identity

import (
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type UserStatus string

const (
	UserStatusActive      UserStatus = "active"
	UserStatusSuspended   UserStatus = "suspended"
	UserStatusDeactivated UserStatus = "deactivated"
)

// User represents a global authenticated person or service account.
type User struct {
	ID           shared.ID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Name         string     `json:"name"`
	Status       UserStatus `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type MembershipStatus string

const (
	MembershipStatusActive    MembershipStatus = "active"
	MembershipStatusInvited   MembershipStatus = "invited"
	MembershipStatusSuspended MembershipStatus = "suspended"
)

// TenantUser represents the membership link between a User and a Tenant.
type TenantUser struct {
	ID       shared.ID        `json:"id"`
	TenantID shared.ID        `json:"tenant_id"`
	UserID   shared.ID        `json:"user_id"`
	Status   MembershipStatus `json:"status"`
	JoinedAt time.Time        `json:"joined_at"`
}

// TenantMembership represents a tenant accessible by a user.
type TenantMembership struct {
	TenantID         shared.ID        `json:"tenant_id"`
	TenantCode       string           `json:"tenant_code"`
	TenantName       string           `json:"tenant_name"`
	TenantStatus     string           `json:"tenant_status"`
	MembershipStatus MembershipStatus `json:"membership_status"`
	JoinedAt         time.Time        `json:"joined_at"`
}

// TenantUserItem represents a user in a tenant context.
type TenantUserItem struct {
	UserID           shared.ID        `json:"user_id"`
	Email            string           `json:"email"`
	Name             string           `json:"name"`
	UserStatus       UserStatus       `json:"user_status"`
	MembershipStatus MembershipStatus `json:"membership_status"`
	JoinedAt         time.Time        `json:"joined_at"`
	Roles            []Role           `json:"roles,omitempty"`
}
