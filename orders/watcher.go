package tonnect

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/tonkeeper/tonapi-go"
)

type ChainTx struct {
	Hash       string
	Comment    string
	AmountNano int64
}

type Watcher struct {
	store      OrderStore
	interval   time.Duration
	checkChain func(address string) ([]ChainTx, error)
}

type textComment struct {
	Text string `json:"text"`
}

type OrderStore interface {
	Save(o *order) error
	Get(id string) (*order, error)
	Pending() ([]*order, error)
	MarkPaid(id string, txHash string) error
}

func CheckChain(client *tonapi.Client, address string) ([]ChainTx, error) {
	resp, err := client.GetBlockchainAccountTransactions(context.Background(), tonapi.GetBlockchainAccountTransactionsParams{
		AccountID: address,
	})
	if err != nil {
		return nil, err
	}
	// log.Printf("DEBUG: got %d transactions for %s", len(resp.Transactions), address)
	var out []ChainTx
	for _, tx := range resp.Transactions {
		inMsg, ok := tx.InMsg.Get()
		if !ok {
			continue
		}
		opName, _ := inMsg.DecodedOpName.Get()
		// log.Printf("DEBUG: opName=%q rawBody=%s", opName, inMsg.DecodedBody)
		if opName != "text_comment" {
			continue
		}
		var body textComment
		if err := json.Unmarshal(inMsg.DecodedBody, &body); err != nil {
			continue
		}
		out = append(out, ChainTx{
			Hash:       tx.Hash,
			Comment:    body.Text,
			AmountNano: inMsg.Value,
		})
	}
	return out, nil
}

func NewWatcher(store OrderStore, interval time.Duration, checkChain func(address string) ([]ChainTx, error)) *Watcher {
	return &Watcher{store: store, interval: interval, checkChain: checkChain}
}

func (w *Watcher) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			w.tick()
		}
	}
}

func (w *Watcher) tick() {
	pending, err := w.store.Pending()
	if err != nil {
		log.Printf("watcher: %v", err)
		return
	}
	for _, o := range pending {
		txs, err := w.checkChain(o.Recipient)
		if err != nil {
			log.Printf("watcher: chain check failed: %v", err)
			continue
		}
		for _, tx := range txs {
			if tx.Comment == o.Reference && tx.AmountNano >= o.Amount {
				w.store.MarkPaid(o.ID, tx.Hash)
				log.Printf("watcher: order %s paid (tx %s)", o.ID, tx.Hash)
			}
		}
	}
}
