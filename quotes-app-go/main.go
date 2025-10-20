package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

//go:embed quotes.json
var quotesFile []byte

func (q QuotesRedis) loadQuotes(ctx context.Context) error {
	var quotes []string
	if err := json.Unmarshal(quotesFile, &quotes); err != nil {
		return err
	}

	for _, v := range quotes {
		err := q.writeQuoteToRedis(ctx, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (q QuotesRedis) getAllQuotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	quotes, err := q.rdb.SMembers(ctx, q.key).Result()
	if err != nil {
		http.Error(w, `{"error": "failed to fetch quotes"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quotes)
}

func (q QuotesRedis) getQuoteByIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	quotes, err := q.rdb.SMembers(ctx, q.key).Result()
	if err != nil {
		http.Error(w, `{"error": "failed to fetch quote"}`, http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	indexStr, ok := vars["index"]
	if !ok {
		http.Error(w, `{"error":"Index not provided"}`, http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 || index >= len(quotes) {
		http.Error(w, `{"error":"Index out of bounds"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, quotes[index])
}

type QuotesRedis struct {
	rdb *redis.Client
	key string
}

func NewQuotes(host string) *QuotesRedis {
	rdb := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: "",
		DB:       0,
	})
	return &QuotesRedis{rdb: rdb, key: "quotes"}
}

func (q QuotesRedis) writeQuoteToRedis(ctx context.Context, value string) error {
	_, err := q.rdb.SAdd(ctx, q.key, value).Result()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	var redisHost string
	flag.StringVar(&redisHost, "r", "localhost:16379", "hostname for redis")
	flag.Parse()

	client := NewQuotes(redisHost)

	ctx := context.Background()

	err := client.loadQuotes(ctx)
	if err != nil {
		log.Fatalf("Failed to load quotes: %v\n", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/quotes", client.getAllQuotes).Methods("GET")
	r.HandleFunc("/quotes/{index}", client.getQuoteByIndex).Methods("GET")

	addr := ":3000"
	fmt.Printf("Server running on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
