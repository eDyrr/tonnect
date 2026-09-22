package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	tonnect "github.com/edyrr/tonnect/orders"
	"github.com/tonkeeper/tonapi-go"
)

type createOrderRequest struct {
	Amount    int64  `json:"amount"`
	Recipient string `json:"recipient"`
}

type authSource struct {
	token string
}

func (a authSource) BearerAuth(ctx context.Context, operationName tonapi.OperationName, client *tonapi.Client) (tonapi.BearerAuth, error) {
	return tonapi.BearerAuth{Token: a.token}, nil
}

func requireKey(key string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		want := "Bearer " + key
		if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func main() {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY is required")
	}
	var store tonnect.Store
	var err error
	storeType := os.Getenv("STORE")
	if storeType == "" {
		storeType = "memory"
	}
	switch storeType {
	case "postgres":
		store, err = tonnect.NewPgStore(os.Getenv("DATABASE_URL"))
		if err != nil {
			log.Fatal(err)
		}
	case "memory":
		store = tonnect.NewMemStore()
	default:
		log.Fatalf("unknown STORE %q (use memory or postgres)", storeType)
	}
	log.Printf("using store %s", storeType)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /orders", requireKey(apiKey, func(w http.ResponseWriter, r *http.Request) {
		var req createOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Amount <= 0 || req.Recipient == "" {
			http.Error(w, "amount and recipient are required", http.StatusBadRequest)
			return
		}

		o := tonnect.New(req.Amount, req.Recipient)
		if err := store.Save(o); err != nil {
			http.Error(w, "failed to save order", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(o)
	}))

	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		o, err := store.Get(id)
		if err != nil {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(o)
	})

	apiURL := os.Getenv("TONAPI_URL")
	if apiURL == "" {
		apiURL = "https://testnet.tonapi.io"
	}

	client, err := tonapi.NewClient(apiURL, authSource{token: os.Getenv("TONAPI_KEY")})
	if err != nil {
		log.Fatal(err)
	}

	var hook *tonnect.Webhook
	if u := os.Getenv("WEBHOOK_URL"); u != "" {
		secret := os.Getenv("WEBHOOK_SECRET")
		if secret == "" {
			log.Fatal("WEBHOOK_SECRET is required when WEBHOOK_URL is set")
		}
		hook = tonnect.NewWebhook(u, secret)
	}

	watcher := tonnect.NewWatcher(store, 5*time.Second, func(address string) ([]tonnect.ChainTx, error) {
		return tonnect.CheckChain(client, address)
	}, hook)
	stop := make(chan struct{})
	go watcher.Run(stop)
	defer close(stop)

	log.Println("tonnect listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
