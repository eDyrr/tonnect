package main

import (
	"context"
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

func main() {
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

	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
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
	})

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
	watcher := tonnect.NewWatcher(store, 5*time.Second, func(address string) ([]tonnect.ChainTx, error) {
		return tonnect.CheckChain(client, address)
	})
	stop := make(chan struct{})
	go watcher.Run(stop)
	defer close(stop)

	log.Println("tonnect listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
