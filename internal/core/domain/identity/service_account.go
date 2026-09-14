package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

var (
	ErrInvalidServiceAccount = errors.New("invalid service account")
	ErrInvalidAPIKey         = errors.New("invalid api key")
	ErrAPIKeyExpired         = errors.New("api key has expired")
	ErrAPIKeyRevoked         = errors.New("api key has been revoked")
	ErrIPNotAllowed          = errors.New("client IP address is not permitted by API key allowlist")
)

type ServiceAccountStatus string

const (
	ServiceAccountStatusActive    ServiceAccountStatus = "active"
	ServiceAccountStatusSuspended ServiceAccountStatus = "suspended"
	ServiceAccountStatusRevoked   ServiceAccountStatus = "revoked"
)

// ServiceAccount represents a machine actor or background integration identity.
type ServiceAccount struct {
	ID          shared.ID            `json:"id"`
	TenantID    shared.ID            `json:"tenant_id"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	RoleID      *shared.ID           `json:"role_id,omitempty"`
	RoleCode    string               `json:"role_code,omitempty"`
	RoleName    string               `json:"role_name,omitempty"`
	Status      ServiceAccountStatus `json:"status"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

func (s *ServiceAccount) IsActive() bool {
	return s.Status == ServiceAccountStatusActive
}

type APIKeyStatus string

const (
	APIKeyStatusActive  APIKeyStatus = "active"
	APIKeyStatusRevoked APIKeyStatus = "revoked"
	APIKeyStatusExpired APIKeyStatus = "expired"
)

const (
	APIKeyPrefix = "mrg_live_"
)

// APIKey represents a cryptographic secret assigned to a Service Account.
type APIKey struct {
	ID               shared.ID    `json:"id"`
	TenantID         shared.ID    `json:"tenant_id"`
	ServiceAccountID shared.ID    `json:"service_account_id"`
	Name             string       `json:"name"`
	KeyPrefix        string       `json:"key_prefix"` // e.g. "mrg_live_9f8a1234..."
	KeyHash          string       `json:"-"`          // SHA-256 hash (never serialized)
	Scopes           []string     `json:"scopes"`
	IPAllowlist      []string     `json:"ip_allowlist,omitempty"`
	Status           APIKeyStatus `json:"status"`
	ExpiresAt        *time.Time   `json:"expires_at,omitempty"`
	LastUsedAt       *time.Time   `json:"last_used_at,omitempty"`
	LastUsedIP       *string      `json:"last_used_ip,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	RevokedAt        *time.Time   `json:"revoked_at,omitempty"`
}

// APIKeyWithSecret includes the raw secret key for one-time display during creation.
type APIKeyWithSecret struct {
	APIKey
	SecretKey string `json:"secret_key"` // Plaintext secret (only displayed once)
}

func (k *APIKey) IsActive(now time.Time) bool {
	if k.Status != APIKeyStatusActive {
		return false
	}
	if k.ExpiresAt != nil && now.After(*k.ExpiresAt) {
		return false
	}
	return true
}

// GenerateAPIKey generates a cryptographically secure 256-bit API key.
// Returns the raw plaintext key, the safe display prefix, and the SHA-256 hash.
func GenerateAPIKey() (rawKey string, prefix string, hash string, err error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	hexToken := hex.EncodeToString(randomBytes)
	rawKey = APIKeyPrefix + hexToken
	prefix = rawKey[:len(APIKeyPrefix)+8] + "..."
	hash = HashAPIKey(rawKey)
	return rawKey, prefix, hash, nil
}

// HashAPIKey computes the hex-encoded SHA-256 hash of a raw API key.
func HashAPIKey(rawKey string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawKey)))
	return hex.EncodeToString(sum[:])
}

// ValidateIP checks whether clientIP matches any entry in the allowlist.
// If allowlist is empty or nil, all IPs are permitted.
func ValidateIP(clientIP string, allowlist []string) bool {
	if len(allowlist) == 0 {
		return true
	}

	// Extract host part if clientIP contains port (e.g. "127.0.0.1:54321")
	host := clientIP
	if h, _, err := net.SplitHostPort(clientIP); err == nil {
		host = h
	}

	parsedIP := net.ParseIP(strings.TrimSpace(host))
	if parsedIP == nil {
		return false
	}

	for _, entry := range allowlist {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// CIDR range (e.g. 192.168.1.0/24)
		if strings.Contains(entry, "/") {
			_, ipNet, err := net.ParseCIDR(entry)
			if err == nil && ipNet.Contains(parsedIP) {
				return true
			}
			continue
		}

		// Exact IP match
		if targetIP := net.ParseIP(entry); targetIP != nil && targetIP.Equal(parsedIP) {
			return true
		}
	}

	return false
}

// ServiceAccountRepository defines persistence contracts for Service Accounts and API Keys.
type ServiceAccountRepository interface {
	// Service Accounts
	CreateServiceAccount(ctx context.Context, sa *ServiceAccount) error
	GetServiceAccountByID(ctx context.Context, tenantID, id shared.ID) (*ServiceAccount, error)
	ListServiceAccounts(ctx context.Context, tenantID shared.ID) ([]ServiceAccount, error)
	UpdateServiceAccount(ctx context.Context, sa *ServiceAccount) error
	DeleteServiceAccount(ctx context.Context, tenantID, id shared.ID) error

	// API Keys
	CreateAPIKey(ctx context.Context, key *APIKey) error
	GetAPIKeyByID(ctx context.Context, tenantID, id shared.ID) (*APIKey, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, *ServiceAccount, []string, error)
	ListAPIKeysByServiceAccount(ctx context.Context, tenantID, serviceAccountID shared.ID) ([]APIKey, error)
	RevokeAPIKey(ctx context.Context, tenantID, id shared.ID) error
	UpdateAPIKeyUsage(ctx context.Context, id shared.ID, clientIP string, usedAt time.Time) error
}

// APIKeyValidator defines the verification contract used by authentication middleware.
type APIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, rawKey, clientIP string) (*shared.AuthClaims, error)
}
