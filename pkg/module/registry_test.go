package module_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akordium-id/mergiate-core/pkg/eventbus"
	"github.com/akordium-id/mergiate-core/pkg/module"
	"github.com/akordium-id/mergiate-core/pkg/response"
)

// mockModule is a test implementation of module.Module.
type mockModule struct {
	manifest      module.Manifest
	initErr       error
	shutdownErr   error
	initCalled    bool
	shutdownOrder *[]string
	initOrder     *[]string
	mu            *sync.Mutex
	permissions   []module.PermissionDefinition
	subscriptions []module.Subscription
}

func (m *mockModule) Manifest() module.Manifest {
	return m.manifest
}

func (m *mockModule) Init(ctx context.Context, host module.Host) error {
	if m.initErr != nil {
		return m.initErr
	}
	m.initCalled = true
	if m.mu != nil && m.initOrder != nil {
		m.mu.Lock()
		*m.initOrder = append(*m.initOrder, m.manifest.Name)
		m.mu.Unlock()
	}
	return nil
}

func (m *mockModule) RegisterRoutes(r chi.Router) {
	r.Get("/ping", func(w http.ResponseWriter, req *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{
			"module": m.manifest.Name,
			"status": "pong",
		})
	})
}

func (m *mockModule) Permissions() []module.PermissionDefinition {
	return m.permissions
}

func (m *mockModule) Subscriptions() []module.Subscription {
	return m.subscriptions
}

func (m *mockModule) Shutdown(ctx context.Context) error {
	if m.shutdownErr != nil {
		return m.shutdownErr
	}
	if m.mu != nil && m.shutdownOrder != nil {
		m.mu.Lock()
		*m.shutdownOrder = append(*m.shutdownOrder, m.manifest.Name)
		m.mu.Unlock()
	}
	return nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	bus := eventbus.New()
	host := module.NewHost(nil, bus, nil, nil, nil)
	reg := module.NewRegistry(host)

	modA := &mockModule{
		manifest: module.Manifest{
			Name:    "module-a",
			Version: "1.0.0",
			Title:   "Module A",
		},
	}

	err := reg.Register(modA)
	require.NoError(t, err)
	assert.Equal(t, 1, reg.Count())

	got, ok := reg.Get("module-a")
	assert.True(t, ok)
	assert.Equal(t, "module-a", got.Manifest().Name)

	// Duplicate registration error
	err = reg.Register(modA)
	require.ErrorIs(t, err, module.ErrDuplicateModule)

	// Invalid module error (nil or empty name)
	err = reg.Register(nil)
	require.ErrorIs(t, err, module.ErrInvalidManifest)

	err = reg.Register(&mockModule{manifest: module.Manifest{Name: ""}})
	require.ErrorIs(t, err, module.ErrInvalidManifest)
}

func TestRegistry_TopologicalSort(t *testing.T) {
	bus := eventbus.New()
	host := module.NewHost(nil, bus, nil, nil, nil)
	reg := module.NewRegistry(host)

	// Dependencies: sales -> inventory -> warehouse
	modSales := &mockModule{
		manifest: module.Manifest{
			Name:      "sales",
			DependsOn: []string{"inventory"},
		},
	}
	modInventory := &mockModule{
		manifest: module.Manifest{
			Name:      "inventory",
			DependsOn: []string{"warehouse"},
		},
	}
	modWarehouse := &mockModule{
		manifest: module.Manifest{
			Name: "warehouse",
		},
	}

	// Register in arbitrary order
	require.NoError(t, reg.Register(modSales))
	require.NoError(t, reg.Register(modWarehouse))
	require.NoError(t, reg.Register(modInventory))

	order, err := reg.ResolveOrder()
	require.NoError(t, err)
	require.Len(t, order, 3)

	names := make([]string, len(order))
	for i, m := range order {
		names[i] = m.Manifest().Name
	}

	// Expected order: warehouse -> inventory -> sales
	assert.Equal(t, []string{"warehouse", "inventory", "sales"}, names)
}

func TestRegistry_MissingDependency(t *testing.T) {
	bus := eventbus.New()
	host := module.NewHost(nil, bus, nil, nil, nil)
	reg := module.NewRegistry(host)

	mod := &mockModule{
		manifest: module.Manifest{
			Name:      "sales",
			DependsOn: []string{"billing"}, // missing
		},
	}

	require.NoError(t, reg.Register(mod))

	_, err := reg.ResolveOrder()
	require.Error(t, err)
	assert.True(t, errors.Is(err, module.ErrModuleNotFound))
}

func TestRegistry_CircularDependency(t *testing.T) {
	bus := eventbus.New()
	host := module.NewHost(nil, bus, nil, nil, nil)
	reg := module.NewRegistry(host)

	modA := &mockModule{
		manifest: module.Manifest{
			Name:      "mod-a",
			DependsOn: []string{"mod-b"},
		},
	}
	modB := &mockModule{
		manifest: module.Manifest{
			Name:      "mod-b",
			DependsOn: []string{"mod-a"},
		},
	}

	require.NoError(t, reg.Register(modA))
	require.NoError(t, reg.Register(modB))

	_, err := reg.ResolveOrder()
	require.Error(t, err)
	assert.True(t, errors.Is(err, module.ErrCircularDependency))
}

func TestRegistry_LifecycleInitAndShutdown(t *testing.T) {
	var (
		mu            sync.Mutex
		initOrder     []string
		shutdownOrder []string
	)

	bus := eventbus.New()
	host := module.NewHost(nil, bus, nil, nil, nil)
	reg := module.NewRegistry(host)

	modBase := &mockModule{
		manifest:      module.Manifest{Name: "base"},
		initOrder:     &initOrder,
		shutdownOrder: &shutdownOrder,
		mu:            &mu,
	}
	modApp := &mockModule{
		manifest:      module.Manifest{Name: "app", DependsOn: []string{"base"}},
		initOrder:     &initOrder,
		shutdownOrder: &shutdownOrder,
		mu:            &mu,
	}

	require.NoError(t, reg.Register(modApp))
	require.NoError(t, reg.Register(modBase))

	ctx := context.Background()

	// Init All
	require.NoError(t, reg.InitAll(ctx))
	assert.Equal(t, []string{"base", "app"}, initOrder)

	// Shutdown All (reverse order)
	require.NoError(t, reg.ShutdownAll(ctx))
	assert.Equal(t, []string{"app", "base"}, shutdownOrder)
}

func TestRegistry_Subscriptions(t *testing.T) {
	bus := eventbus.New()
	host := module.NewHost(nil, bus, nil, nil, nil)
	reg := module.NewRegistry(host)

	received := make(chan string, 1)

	mod := &mockModule{
		manifest: module.Manifest{Name: "notify-mod"},
		subscriptions: []module.Subscription{
			{
				Pattern: "order.created",
				Handler: func(ctx context.Context, evt eventbus.Event) error {
					received <- evt.EventType()
					return nil
				},
			},
		},
	}

	require.NoError(t, reg.Register(mod))
	require.NoError(t, reg.InitAll(context.Background()))
	reg.BindSubscriptions()

	// Publish an event
	bus.Publish(context.Background(), eventbus.NewBaseEvent(
		[16]byte{}, "order.created", "Order", [16]byte{}, map[string]any{"total": 150000},
	))

	select {
	case topic := <-received:
		assert.Equal(t, "order.created", topic)
	default:
		t.Fatal("expected event to be received by module subscription")
	}
}

func TestRegistry_MountRoutes(t *testing.T) {
	bus := eventbus.New()
	host := module.NewHost(nil, bus, nil, nil, nil)
	reg := module.NewRegistry(host)

	mod := &mockModule{
		manifest: module.Manifest{
			Name:        "billing",
			Version:     "1.2.0",
			Title:       "Billing Module",
			Description: "Invoicing and payments",
		},
	}

	require.NoError(t, reg.Register(mod))
	require.NoError(t, reg.InitAll(context.Background()))

	router := chi.NewRouter()
	reg.MountRoutes(router)

	// 1. Discovery endpoint: GET /modules
	reqDiscovery := httptest.NewRequest(http.MethodGet, "/modules", nil)
	recDiscovery := httptest.NewRecorder()
	router.ServeHTTP(recDiscovery, reqDiscovery)

	assert.Equal(t, http.StatusOK, recDiscovery.Code)

	var discoveryResponse struct {
		Success bool              `json:"success"`
		Data    []module.Manifest `json:"data"`
	}
	err := json.Unmarshal(recDiscovery.Body.Bytes(), &discoveryResponse)
	require.NoError(t, err)
	assert.True(t, discoveryResponse.Success)
	require.Len(t, discoveryResponse.Data, 1)
	assert.Equal(t, "billing", discoveryResponse.Data[0].Name)
	assert.Equal(t, "1.2.0", discoveryResponse.Data[0].Version)

	// 2. Module endpoint: GET /modules/billing/ping
	reqPing := httptest.NewRequest(http.MethodGet, "/modules/billing/ping", nil)
	recPing := httptest.NewRecorder()
	router.ServeHTTP(recPing, reqPing)

	assert.Equal(t, http.StatusOK, recPing.Code)

	var pingResponse struct {
		Success bool              `json:"success"`
		Data    map[string]string `json:"data"`
	}
	err = json.Unmarshal(recPing.Body.Bytes(), &pingResponse)
	require.NoError(t, err)
	assert.Equal(t, "billing", pingResponse.Data["module"])
	assert.Equal(t, "pong", pingResponse.Data["status"])
}

func TestRegistry_RegisterPermissions_NilDB(t *testing.T) {
	bus := eventbus.New()
	host := module.NewHost(nil, bus, nil, nil, nil)
	reg := module.NewRegistry(host)

	mod := &mockModule{
		manifest: module.Manifest{Name: "sec-mod"},
		permissions: []module.PermissionDefinition{
			{
				Code:        "sec:item:read",
				Name:        "Read Security Items",
				Category:    "security",
				Description: "Allows viewing items",
			},
		},
	}

	require.NoError(t, reg.Register(mod))
	require.NoError(t, reg.InitAll(context.Background()))

	// Host has nil DB, should return nil without panic
	err := reg.RegisterPermissions(context.Background())
	assert.NoError(t, err)
}
