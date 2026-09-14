package module

import (
	"context"

	"github.com/go-chi/chi/v5"

	"github.com/akordium-id/mergiate-core/pkg/eventbus"
)

// Manifest contains metadata describing an external or first-party business module.
type Manifest struct {
	Name        string   `json:"name"`                   // Unique slug e.g. "sales", "inventory", "pos"
	Version     string   `json:"version"`                // Semantic version e.g. "1.0.0"
	Title       string   `json:"title"`                  // Human-readable title
	Description string   `json:"description"`            // Brief summary of module functionality
	Author      string   `json:"author,omitempty"`       // Author or organization
	Website     string   `json:"website,omitempty"`      // Documentation or repository URL
	DependsOn   []string `json:"depends_on,omitempty"`   // Modules that must be initialized before this module
}

// PermissionDefinition defines a custom RBAC permission registered by the module.
type PermissionDefinition struct {
	Code        string `json:"code"`        // Unique code e.g. "sales:order:approve"
	Name        string `json:"name"`        // Human-readable display name
	Category    string `json:"category"`    // Permission category e.g. "sales"
	Description string `json:"description"` // Explanation of access granted
}

// Subscription defines an event bus subscription registered by the module.
type Subscription struct {
	Pattern string           // Event pattern (exact, prefix "order.*", or wildcard "*")
	Handler eventbus.Handler // Event listener callback function
}

// Module is the contract that any first-party or third-party extension must implement.
// Core never imports modules; modules import Core and implement this interface.
type Module interface {
	// Manifest returns the static metadata and dependency declarations of the module.
	Manifest() Manifest

	// Init initializes module usecases, services, and state using the Core Host environment.
	Init(ctx context.Context, host Host) error

	// RegisterRoutes mounts the module's HTTP endpoints into the router.
	// Endpoints will be accessible under /api/v1/modules/{manifest.Name}.
	RegisterRoutes(r chi.Router)

	// Permissions returns domain-specific RBAC permissions to register in Core IAM.
	Permissions() []PermissionDefinition

	// Subscriptions returns domain event listeners to bind to the Core Event Bus.
	Subscriptions() []Subscription

	// Shutdown gracefully terminates background workers, timers, or resources.
	Shutdown(ctx context.Context) error
}
