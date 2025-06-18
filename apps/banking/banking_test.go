package banking_test

import (
	"context"
	"testing"
	"time"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/co"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/ro"
	"github.com/blinkinglight/gobeego/apps/banking"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func init() {
	bee.RegisterEvent[banking.AccountDebited](banking.Aggregate, banking.DebitedEvent)
	bee.RegisterEvent[banking.AccountCredited](banking.Aggregate, banking.CreditedEvent)
	bee.RegisterEvent[banking.AccountCreated](banking.Aggregate, banking.CreatedEvent)

	bee.RegisterCommand[banking.DebitAccountCommand](banking.Aggregate, banking.DebitCommand)
	bee.RegisterCommand[banking.CreditAccountCommand](banking.Aggregate, banking.CreditCommand)
	bee.RegisterCommand[banking.CreateAccountCommand](banking.Aggregate, banking.CreateCommand)
}

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

func TestMain(t *testing.T) {
	nc, cleanup, err := client()
	if err != nil {
		t.Fatalf("failed to create NATS client: %v", err)
	}
	defer cleanup()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to get JetStream context: %v", err)
	}
	js.DeleteStream("events") // Clean up any existing stream
	js.AddStream(&nats.StreamConfig{
		Name:     "events",
		Subjects: []string{"events.>"},
	})

	ctx := bee.WithNats(t.Context(), nc)
	ctx = bee.WithJetStream(ctx, js)

	service := &banking.PaymentService{}
	go bee.Command(ctx, service, co.WithAggreate(banking.Aggregate))

	createAccount1 := &banking.CreateAccountCommand{
		AccountID: "54321",
		Currency:  "USD",
		Balance:   0,
		Ref:       "create-54321",
	}
	createCmd := &gen.CommandEnvelope{
		AggregateId: "54321",
		Aggregate:   banking.Aggregate,
		CommandType: "create",
	}
	bee.PublishCommand(ctx, createCmd, createAccount1)

	createAccount2 := &banking.CreateAccountCommand{
		AccountID: "12345",
		Currency:  "USD",
		Balance:   0,
		Ref:       "create-12345",
	}
	createCmd2 := &gen.CommandEnvelope{
		AggregateId: "12345",
		Aggregate:   banking.Aggregate,
		CommandType: "create",
	}
	bee.PublishCommand(ctx, createCmd2, createAccount2)

	time.Sleep(100 * time.Millisecond) // Wait for events to be processed

	ev := &banking.CreditAccountCommand{
		ToAccountID:   "12345",
		FromAccountID: "CASH",
		Amount:        1000,
		Ref:           "payment-001",
	}
	event := &gen.CommandEnvelope{
		AggregateId: "12345",
		Aggregate:   banking.Aggregate,
		CommandType: banking.CreditCommand,
	}

	bee.PublishCommand(ctx, event, ev)

	time.Sleep(100 * time.Millisecond) // Wait for events to be processed

	agg := &banking.AccountAggregate{ID: "12345"}
	bee.Replay(ctx, agg, ro.WithAggreate(banking.Aggregate), ro.WithAggregateID("12345"))

	if agg.Balance != 1000 {
		t.Errorf("Expected balance to be 1000, got %v", agg.Balance)
	}

	debitCmd := &banking.DebitAccountCommand{
		FromAccountID: "12345",
		ToAccountID:   "54321",
		Amount:        1000,
		Ref:           "payment-001",
	}
	cmdd := &gen.CommandEnvelope{
		AggregateId: "12345",
		Aggregate:   banking.Aggregate,
		CommandType: banking.DebitCommand,
	}
	bee.PublishCommand(ctx, cmdd, debitCmd)

	time.Sleep(100 * time.Millisecond) // Wait for events to be processed

	agg = &banking.AccountAggregate{ID: "12345"}
	bee.Replay(ctx, agg, ro.WithAggreate(banking.Aggregate), ro.WithAggregateID("12345"))

	if agg.Balance != 0 {
		t.Errorf("Expected balance to be 0 after debit, got %v", agg.Balance)
	}

	agg = &banking.AccountAggregate{ID: "54321"}
	bee.Replay(ctx, agg, ro.WithAggreate(banking.Aggregate), ro.WithAggregateID("54321"))

	if agg.Balance != 1000 {
		t.Errorf("Expected balance of account 54321 to be 1000 after credit, got %v", agg.Balance)
	}
	time.Sleep(100 * time.Millisecond) // Wait for events to be processed
}
