package tonnect

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Webhook struct {
	url    string
	secret string
	client *http.Client
}

func NewWebhook(url, secret string) *Webhook {
	return &Webhook{url: url, secret: secret, client: &http.Client{Timeout: 10 * time.Second}}
}

type paidEvent struct {
	Event     string `json:"event"`
	OrderID   string `json:"order_id"`
	Reference string `json:"reference"`
	Recipient string `json:"recipient"`
	Amount    int64  `json:"amount"`
	TxHash    string `json:"tx_hash"`
}

func sign(secret, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (w *Webhook) Paid(o *order, txHash string) {
	if w == nil {
		return
	}
	body, _ := json.Marshal(paidEvent{
		Event: "payment.confirmed", OrderID: o.ID, Reference: o.Reference,
		Recipient: o.Recipient, Amount: o.Amount, TxHash: o.PaidTxHash,
	})
	go w.deliver(o.ID, body)
}

func (w *Webhook) deliver(orderID string, body []byte) {
	if w.client == nil {
		w.client = &http.Client{Timeout: 10 * time.Second}
	}
	delays := []time.Duration{0, 5 * time.Second, 30 * time.Second, 2 * time.Minute, 10 * time.Minute}
	for i, d := range delays {
		time.Sleep(d)
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		req, err := http.NewRequest("POST", w.url, bytes.NewReader(body))
		if err != nil {
			log.Printf("webhooks: bad url: %v", err)
			return
		}
		req.Header.Set("Context-Type", "application/json")
		req.Header.Set("X-Tonnect-Timestamp", ts)
		req.Header.Set("X-Tonnect-Signature", sign(w.secret, ts, body))

		resp, err := w.client.Do(req)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				log.Printf("webhook: order %s delivered", orderID)
				return
			}
			err = fmt.Errorf("status %d", resp.StatusCode)
		}
		log.Printf("webhook: order %s attempt %d/%d faild: %v", orderID, i+1, len(delays), err)
	}
	log.Printf("webhook: order %s gave up", orderID)
}
