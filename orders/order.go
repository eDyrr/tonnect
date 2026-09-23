package tonnect

import "github.com/google/uuid"

type status string

const (
	Created status = "created"
	Paid    status = "paid"
)

type order struct {
	ID         string `json:"id"`
	Status     status `json:"status"`
	Reference  string `json:"reference"`
	Recipient  string `json:"recipient"`
	Amount     int64  `json:"amount"`
	PaidTxHash string `json:"paid_tx_hash,omitempty"`
}

func New(amount int64, recipient string) *order {
	var o order
	o.ID = uuid.NewString()
	o.Reference = uuid.NewString()
	o.Status = Created
	o.Amount = amount
	o.Recipient = recipient
	return &o
}
