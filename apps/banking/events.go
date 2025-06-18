package banking

type AccountDebited struct {
	AccountID  string `json:"account_id"`
	Amount     int64  `json:"amount"`
	Ref        string `json:"ref"` // e.g. PaymentID
	NewBalance int64  `json:"new_balance"`
	Timestamp  int64  `json:"timestamp"`
}

type AccountCredited struct {
	AccountID  string `json:"account_id"`
	Amount     int64  `json:"amount"`
	Ref        string `json:"ref"` // e.g. PaymentID
	NewBalance int64  `json:"new_balance"`
	Timestamp  int64  `json:"timestamp"`
}

type AccountCreated struct {
	AccountID string `json:"account_id"`
	Currency  string `json:"currency"`
	Balance   int64  `json:"balance"`
	Ref       string `json:"ref"` // e.g. PaymentID
	Timestamp int64  `json:"timestamp"`
}
