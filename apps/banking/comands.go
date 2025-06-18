package banking

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
