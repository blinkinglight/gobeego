package shopping_test

import (
	"context"
	"testing"
	"time"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/co"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/ro"
	"github.com/blinkinglight/gobeego/apps/shopping"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func client() (*nats.Conn, func(), error) {
	server, err := embeddednats.New(
		context.Background(),
		embeddednats.WithShouldClearData(true),
		embeddednats.WithNATSServerOptions(&server.Options{
			JetStream:    true,
			NoLog:        false,
			Debug:        true,
			Trace:        true,
			TraceVerbose: true,
			Port:         4333,
			StoreDir:     "./data/nats",
		}),
	)
	if err != nil {
		return nil, nil, err
	}
	server.WaitForServer()

	nc, err := server.Client()

	return nc, func() {
		nc.Close()
		server.Close()
	}, err
}

func TestCore(t *testing.T) {
	nc, cleanup, err := client()
	if err != nil {
		t.Fatalf("failed to create NATS client: %v", err)
	}
	defer cleanup()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to get JetStream context: %v", err)
	}
	js.DeleteStream("events")          // Clean up any existing stream
	time.Sleep(100 * time.Millisecond) // Ensure stream deletion is processed
	js.AddStream(&nats.StreamConfig{
		Name:     "events",
		Subjects: []string{"events.>"},
	})
	time.Sleep(100 * time.Millisecond) // Ensure stream deletion is processed

	ctx := bee.WithNats(t.Context(), nc)
	ctx = bee.WithJetStream(ctx, js)

	service := &shopping.CartService{Ctx: ctx}
	go bee.Command(ctx, service, co.WithAggreate("cart"))

	userService := &shopping.UserService{Ctx: ctx}
	go bee.Command(ctx, userService, co.WithAggreate("user"))

	createCmd := gen.CommandEnvelope{
		Aggregate:   "cart",
		AggregateId: "cart-1",
		CommandType: "create",
		Payload:     []byte(`{"items":[],"total":0,"discount":0}`),
	}
	bee.PublishCommand(ctx, &createCmd, nil)

	addItemPayload := shopping.CartItemAdd{
		Product: shopping.Product{
			ID:    "item1",
			Name:  "Test Item",
			Price: 10.0,
		},
	}

	addItemCmd := gen.CommandEnvelope{
		Aggregate:   "cart",
		AggregateId: "cart-1",
		CommandType: "add_item",
		Metadata:    map[string]string{"source": "test"},
	}

	bee.PublishCommand(ctx, &addItemCmd, addItemPayload)

	addItemPayload1 := shopping.CartItemAdd{
		Product: shopping.Product{
			ID:    "item1",
			Name:  "Test Item",
			Price: 10.0,
		},
	}

	addItemCmd1 := gen.CommandEnvelope{
		Aggregate:   "cart",
		AggregateId: "cart-1",
		CommandType: "add_item",
		Metadata:    map[string]string{"source": "test"},
	}

	bee.PublishCommand(ctx, &addItemCmd1, addItemPayload1)

	time.Sleep(100 * time.Millisecond)

	cart := &shopping.ShoppingCartAggregate{ID: "cart-1"}
	bee.Replay(ctx, cart, ro.WithAggreate("cart"), ro.WithAggregateID("cart-1"))

	if len(cart.Items) != 2 {
		t.Errorf("Expected 2 items in cart, got %d", len(cart.Items))
	}
	if cart.Total != 20.0 {
		t.Errorf("Expected total to be 20.0, got %f", cart.Total)
	}

	removeItemPayload := shopping.CartItemRemove{
		ProductID: "item1",
	}

	removeItemCmd := gen.CommandEnvelope{
		Aggregate:   "cart",
		AggregateId: "cart-1",
		CommandType: "remove_item",
		Metadata:    map[string]string{"source": "test"},
	}

	bee.PublishCommand(ctx, &removeItemCmd, removeItemPayload)

	time.Sleep(100 * time.Millisecond)

	cart = &shopping.ShoppingCartAggregate{ID: "cart-1"}
	bee.Replay(ctx, cart, ro.WithAggreate("cart"), ro.WithAggregateID("cart-1"))

	if len(cart.Items) != 1 {
		t.Errorf("Expected 1 item in cart after removal, got %d", len(cart.Items))
	}
	if cart.Total != 10.0 {
		t.Errorf("Expected total to be 10.0 after removal, got %f", cart.Total)
	}

	createUserCmd := gen.CommandEnvelope{
		Aggregate:   "user",
		AggregateId: "user-1",
		CommandType: "create",
		Payload:     []byte(`{"name":"Test User","email":"x@x.x"}`),
		Metadata:    map[string]string{"cart_id": "cart-1"},
	}
	bee.PublishCommand(ctx, &createUserCmd, nil)

	time.Sleep(100 * time.Millisecond)

	user := &shopping.UserAggregate{ID: "user-1"}
	bee.Replay(ctx, user, ro.WithAggreate("user"), ro.WithAggregateID("user-1"))

	if user.ID != "user-1" {
		t.Errorf("Expected user ID to be '1', got '%s'", user.ID)
	}
	if user.Name != "Test User" {
		t.Errorf("Expected user name to be 'Test User', got '%s'", user.Name)
	}
	if user.Email != "x@x.x" {
		t.Errorf("Expected user email to be 'x@x.x', got '%s'", user.Email)
	}
	if len(user.Carts) != 1 {
		t.Errorf("Expected user to have 1 cart, got %d", len(user.Carts))
	}
}
