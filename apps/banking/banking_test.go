package banking_test

import (
	"context"
	"testing"
	"time"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/co"
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

	service := &banking.PaymentService{Ctx: ctx}
	go bee.Command(ctx, service, co.WithAggreate(banking.Aggregate))

	createCmd1 := banking.CreateAccount("54321", "USD", 0, "create-54321")
	bee.PublishCommand(ctx, createCmd1, nil)

	createCmd2 := banking.CreateAccount("12345", "USD", 0, "create-12345")
	bee.PublishCommand(ctx, createCmd2, nil)

	time.Sleep(100 * time.Millisecond) // Wait for events to be processed

	creditCmd := banking.CreditAccount("CASH", "12345", 1000, "payment-001")
	bee.PublishCommand(ctx, creditCmd, nil)

	time.Sleep(100 * time.Millisecond) // Wait for events to be processed

	agg := &banking.AccountAggregate{ID: "12345"}
	bee.Replay(ctx, agg, ro.WithAggreate(banking.Aggregate), ro.WithAggregateID("12345"))

	if agg.Balance != 1000 {
		t.Errorf("Expected balance to be 1000, got %v", agg.Balance)
	}

	debitCmd := banking.DebitAccount("12345", "54321", 1000, "payment-001")
	bee.PublishCommand(ctx, debitCmd, nil)

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
