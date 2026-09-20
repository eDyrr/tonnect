package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	tonnect "github.com/edyrr/tonnect/orders"
	"github.com/joho/godotenv"
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
	// store := *tonnect.NewStore() old
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}
	connString := os.Getenv("DATABASE_URL")

	store, err := tonnect.NewPgStore(connString)
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	// client.GetBlockchaininAccountTransactions()

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

	client, err := tonapi.NewClient("https://testnet.tonapi.io", authSource{token: "AFZ4OBY7RPHW63QAAAACMILEQLPY7Z6U6YGEXHTQU6SLX2CNTTAPGO4SPJNVYFKXLXHQJ5Q"})
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
