package test

import (
	"encoding/json"

	"github.com/blinkinglight/bee/gen"
)

type AccountCreatedEvent struct {
	AccountId string `json:"account_id"`
	Currency  string `json:"currency"`
	Balance   int64  `json:"balance"`
	Ref       string `json:"ref"`
	Timestamp int64  `json:"timestamp"`
}

type AccountDebitedEvent struct {
	AccountId  string `json:"account_id"`
	Amount     int64  `json:"amount"`
	Ref        string `json:"ref"`
	NewBalance int64  `json:"new_balance"`
	Timestamp  int64  `json:"timestamp"`
}

type AccountCreditedEvent struct {
	AccountId  string `json:"account_id"`
	Amount     int64  `json:"amount"`
	Ref        string `json:"ref"`
	NewBalance int64  `json:"new_balance"`
	Timestamp  int64  `json:"timestamp"`
}

func AccountCreated(AccountId string, Currency string, Balance int64, Ref string, Timestamp int64) *gen.EventEnvelope {
	payload := &AccountCreatedEvent{
		AccountId: AccountId,
		Currency:  Currency,
		Balance:   Balance,
		Ref:       Ref,
		Timestamp: Timestamp,
	}
	b, _ := json.Marshal(payload)

	event := &gen.EventEnvelope{
		AggregateType: "accounts",
		EventType:     "created",
		Payload:       b,
	}
	return event
}

func AccountDebited(AccountId string, Amount int64, Ref string, NewBalance int64, Timestamp int64) *gen.EventEnvelope {
	payload := &AccountDebitedEvent{
		AccountId:  AccountId,
		Amount:     Amount,
		Ref:        Ref,
		NewBalance: NewBalance,
		Timestamp:  Timestamp,
	}
	b, _ := json.Marshal(payload)

	event := &gen.EventEnvelope{
		AggregateType: "accounts",
		EventType:     "debited",
		Payload:       b,
	}
	return event
}

func AccountCredited(AccountId string, Amount int64, Ref string, NewBalance int64, Timestamp int64) *gen.EventEnvelope {
	payload := &AccountCreditedEvent{
		AccountId:  AccountId,
		Amount:     Amount,
		Ref:        Ref,
		NewBalance: NewBalance,
		Timestamp:  Timestamp,
	}
	b, _ := json.Marshal(payload)

	event := &gen.EventEnvelope{
		AggregateType: "accounts",
		EventType:     "credited",
		Payload:       b,
	}
	return event
}
