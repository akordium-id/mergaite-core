package communication

import (
	"regexp"
	"strings"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

const (
	CommentTypeGeneral      = "comment"
	CommentTypeInternalNote = "internal_note"

	NotificationTypeMention      = "mention"
	NotificationTypeStatusChange = "status_change"
	NotificationTypeAssignment   = "assignment"
	NotificationTypeSystem       = "system"
)

// Comment represents an internal note or team comment attached to any domain entity.
type Comment struct {
	ID         shared.ID      `json:"id"`
	TenantID   shared.ID      `json:"tenant_id"`
	EntityType string         `json:"entity_type"`
	EntityID   shared.ID      `json:"entity_id"`
	AuthorID   shared.ID      `json:"author_id"`
	Type       string         `json:"type"`
	Content    string         `json:"content"`
	Mentions   []shared.ID    `json:"mentions"`
	ParentID   *shared.ID     `json:"parent_id,omitempty"`
	IsPinned   bool           `json:"is_pinned"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// CommentWithAuthor provides joined author metadata for display.
type CommentWithAuthor struct {
	Comment
	AuthorName  string `json:"author_name"`
	AuthorEmail string `json:"author_email"`
}

// Notification represents a targeted in-app notification delivered to a user.
type Notification struct {
	ID         shared.ID      `json:"id"`
	TenantID   shared.ID      `json:"tenant_id"`
	UserID     shared.ID      `json:"user_id"`
	ActorID    *shared.ID     `json:"actor_id,omitempty"`
	Type       string         `json:"type"`
	Title      string         `json:"title"`
	Message    string         `json:"message"`
	EntityType string         `json:"entity_type,omitempty"`
	EntityID   *shared.ID     `json:"entity_id,omitempty"`
	IsRead     bool           `json:"is_read"`
	ReadAt     *time.Time     `json:"read_at,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// ActivityItem is a normalized chronological event entry for the unified activity feed.
type ActivityItem struct {
	ID           shared.ID      `json:"id"`
	Timestamp    time.Time      `json:"timestamp"`
	ActivityType string         `json:"activity_type"` // "audit" | "comment" | "attachment"
	ActorID      *shared.ID     `json:"actor_id,omitempty"`
	ActorName    string         `json:"actor_name,omitempty"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Payload      map[string]any `json:"payload,omitempty"`
}

var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_\.\-]+)`)

// ExtractMentionStrings scans comment content and returns all unique mention identifiers.
func ExtractMentionStrings(content string) []string {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var list []string
	for _, m := range matches {
		if len(m) > 1 {
			val := strings.TrimSpace(m[1])
			if val != "" && !seen[val] {
				seen[val] = true
				list = append(list, val)
			}
		}
	}
	return list
}
