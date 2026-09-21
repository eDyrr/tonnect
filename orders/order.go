package tonnect

import "github.com/google/uuid"

type status string

const (
	Created status = "created"
	Paid    status = "paid"
)

type order struct {
	ID         string
	Status     status
	Reference  string
	Recipient  string
	Amount     int64
	PaidTxHash string
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
