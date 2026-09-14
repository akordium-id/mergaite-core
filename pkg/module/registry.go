package module

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/pkg/response"
)

var (
	// ErrModuleNotFound is returned when a requested module or dependency does not exist.
	ErrModuleNotFound = errors.New("module not found")

	// ErrDuplicateModule is returned when registering a module whose name is already taken.
	ErrDuplicateModule = errors.New("module with same name already registered")

	// ErrInvalidManifest is returned when a module manifest violates validation constraints.
	ErrInvalidManifest = errors.New("module manifest is invalid")

	// ErrCircularDependency is returned when module dependencies form a directed cycle.
	ErrCircularDependency = errors.New("circular dependency detected among modules")
)

// Registry manages the lifecycle, dependency graph, route mounting, and SPI of all registered modules.
type Registry struct {
	mu      sync.RWMutex
	host    Host
	logger  *slog.Logger
	modules map[string]Module
	order   []Module
}

// NewRegistry constructs a new module Registry instance.
func NewRegistry(host Host, logger ...*slog.Logger) *Registry {
	l := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &Registry{
		host:    host,
		logger:  l,
		modules: make(map[string]Module),
		order:   make([]Module, 0),
	}
}

// Register adds a module to the registry after validating its manifest.
func (r *Registry) Register(m Module) error {
	if m == nil {
		return fmt.Errorf("%w: module cannot be nil", ErrInvalidManifest)
	}

	manifest := m.Manifest()
	if manifest.Name == "" {
		return fmt.Errorf("%w: module name cannot be empty", ErrInvalidManifest)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.modules[manifest.Name]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicateModule, manifest.Name)
	}

	r.modules[manifest.Name] = m
	return nil
}

// Get returns the module with the specified name, or false if not found.
func (r *Registry) Get(name string) (Module, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.modules[name]
	return m, ok
}

// Count returns the number of registered modules.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.modules)
}

// List returns the manifests of all registered modules in resolved dependency order.
func (r *Registry) List() []Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.order) > 0 {
		list := make([]Manifest, len(r.order))
		for i, m := range r.order {
			list[i] = m.Manifest()
		}
		return list
	}

	// Fallback to sorted keys if order has not been resolved yet
	keys := make([]string, 0, len(r.modules))
	for k := range r.modules {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	list := make([]Manifest, len(keys))
	for i, k := range keys {
		list[i] = r.modules[k].Manifest()
	}
	return list
}

// ResolveOrder analyzes module dependencies and computes a valid initialization order
// using Kahn's algorithm for topological sorting. It detects missing dependencies and cycles.
func (r *Registry) ResolveOrder() ([]Module, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, err := r.resolveOrderLocked()
	if err != nil {
		return nil, err
	}
	r.order = order
	return order, nil
}

func (r *Registry) resolveOrderLocked() ([]Module, error) {
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // Prerequisite -> list of dependent modules

	for name := range r.modules {
		inDegree[name] = 0
	}

	// Build graph and calculate in-degrees
	for name, m := range r.modules {
		manifest := m.Manifest()
		for _, dep := range manifest.DependsOn {
			if _, exists := r.modules[dep]; !exists {
				return nil, fmt.Errorf("%w: module %q depends on missing module %q", ErrModuleNotFound, name, dep)
			}
			// dep must be initialized before name
			dependents[dep] = append(dependents[dep], name)
			inDegree[name]++
		}
	}

	// Enqueue all nodes with inDegree 0 (no prerequisites)
	queue := make([]string, 0)
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	// Sort for deterministic order across restarts
	sort.Strings(queue)

	resolved := make([]Module, 0, len(r.modules))
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		resolved = append(resolved, r.modules[curr])

		for _, depName := range dependents[curr] {
			inDegree[depName]--
			if inDegree[depName] == 0 {
				queue = append(queue, depName)
				sort.Strings(queue)
			}
		}
	}

	if len(resolved) != len(r.modules) {
		return nil, fmt.Errorf("%w: resolved %d of %d modules", ErrCircularDependency, len(resolved), len(r.modules))
	}

	return resolved, nil
}

// InitAll resolves dependencies and initializes all modules sequentially in topological order.
func (r *Registry) InitAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, err := r.resolveOrderLocked()
	if err != nil {
		return err
	}
	r.order = order

	for _, m := range r.order {
		manifest := m.Manifest()
		r.logger.Info("initializing module",
			slog.String("module", manifest.Name),
			slog.String("version", manifest.Version),
		)
		if err := m.Init(ctx, r.host); err != nil {
			return fmt.Errorf("failed to initialize module %q: %w", manifest.Name, err)
		}
	}

	return nil
}

// RegisterPermissions registers custom permissions defined by all loaded modules into the Core IAM catalog.
func (r *Registry) RegisterPermissions(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	db := r.host.DB()
	if db == nil {
		return nil
	}

	for _, m := range r.order {
		perms := m.Permissions()
		for _, p := range perms {
			permID, err := shared.NewID()
			if err != nil {
				return err
			}
			query := `
				INSERT INTO permissions (id, code, name, category, description)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (code) DO UPDATE SET
					name = EXCLUDED.name,
					category = EXCLUDED.category,
					description = EXCLUDED.description
			`
			if _, err := db.Exec(ctx, query, permID, p.Code, p.Name, p.Category, p.Description); err != nil {
				return fmt.Errorf("failed to register permission %q for module %q: %w", p.Code, m.Manifest().Name, err)
			}
		}
	}

	return nil
}

// BindSubscriptions registers all event listeners declared by loaded modules to the Core Event Bus.
func (r *Registry) BindSubscriptions() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bus := r.host.EventBus()
	if bus == nil {
		return
	}

	for _, m := range r.order {
		subs := m.Subscriptions()
		for _, sub := range subs {
			r.logger.Info("binding module event subscription",
				slog.String("module", m.Manifest().Name),
				slog.String("pattern", sub.Pattern),
			)
			bus.Subscribe(sub.Pattern, sub.Handler)
		}
	}
}

// MountRoutes mounts module discovery and each module's sub-router under /modules.
func (r *Registry) MountRoutes(router chi.Router) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	router.Route("/modules", func(mr chi.Router) {
		// Discovery endpoint: GET /api/v1/modules
		mr.Get("/", func(w http.ResponseWriter, req *http.Request) {
			manifests := make([]Manifest, 0, len(r.order))
			for _, m := range r.order {
				manifests = append(manifests, m.Manifest())
			}
			response.JSON(w, http.StatusOK, manifests)
		})

		// Mount each module sub-routes: /api/v1/modules/{module_name}/*
		for _, m := range r.order {
			manifest := m.Manifest()
			mr.Route("/"+manifest.Name, func(sub chi.Router) {
				m.RegisterRoutes(sub)
			})
		}
	})
}

// ShutdownAll gracefully terminates all modules in reverse topological order.
func (r *Registry) ShutdownAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for i := len(r.order) - 1; i >= 0; i-- {
		m := r.order[i]
		manifest := m.Manifest()
		r.logger.Info("shutting down module", slog.String("module", manifest.Name))
		if err := m.Shutdown(ctx); err != nil {
			r.logger.Error("error shutting down module",
				slog.String("module", manifest.Name),
				slog.Any("error", err),
			)
			errs = append(errs, fmt.Errorf("module %s: %w", manifest.Name, err))
		}
	}

	return errors.Join(errs...)
}
