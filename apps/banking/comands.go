package banking

import (
	"encoding/json"

	"github.com/blinkinglight/bee/gen"
)

type CreateAccountCommand struct {
	AccountID string `json:"account_id"`
	Currency  string `json:"currency"`
	Balance   int64  `json:"balance"`
	Ref       string `json:"ref"`
}

type DebitAccountCommand struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int64  `json:"amount"`
	Ref           string `json:"ref"`
}

type CreditAccountCommand struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int64  `json:"amount"`
	Ref           string `json:"ref"`
}

func CreateAccount(accountID, currency string, balance int64, ref string) *gen.CommandEnvelope {
	payload := &CreateAccountCommand{
		AccountID: accountID,
		Currency:  currency,
		Balance:   balance,
		Ref:       ref,
	}
	b, _ := json.Marshal(payload)
	cmd := &gen.CommandEnvelope{
		AggregateId: accountID,
		Aggregate:   Aggregate,
		CommandType: CreateCommand,
		Payload:     b,
	}
	return cmd
}

func DebitAccount(fromAccountID, toAccountID string, amount int64, ref string) *gen.CommandEnvelope {
	payload := &DebitAccountCommand{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        amount,
		Ref:           ref,
	}
	b, _ := json.Marshal(payload)
	cmd := &gen.CommandEnvelope{
		AggregateId: fromAccountID,
		Aggregate:   Aggregate,
		CommandType: DebitCommand,
		Payload:     b,
	}
	return cmd
}

func CreditAccount(fromAccountID, toAccountID string, amount int64, ref string) *gen.CommandEnvelope {
	payload := &CreditAccountCommand{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        amount,
		Ref:           ref,
	}
	b, _ := json.Marshal(payload)
	cmd := &gen.CommandEnvelope{
		AggregateId: toAccountID,
		Aggregate:   Aggregate,
		CommandType: CreditCommand,
		Payload:     b,
	}
	return cmd
}
