package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

//go:embed quotes.json
var quotesFile []byte

type QuoteStore interface {
	Load(ctx context.Context, quotes []string) error
	All(ctx context.Context) ([]string, error)
}

type RedisStore struct {
	rdb *redis.Client
	key string
}

func NewRedisStore(host string) *RedisStore {
	rdb := redis.NewClient(&redis.Options{
		Addr:     host,
		Password: "",
		DB:       0,
	})
	return &RedisStore{rdb: rdb, key: "quotes"}
}

func (s *RedisStore) Load(ctx context.Context, quotes []string) error {
	for _, v := range quotes {
		if _, err := s.rdb.SAdd(ctx, s.key, v).Result(); err != nil {
			return err
		}
	}
	return nil
}

func (s *RedisStore) All(ctx context.Context) ([]string, error) {
	return s.rdb.SMembers(ctx, s.key).Result()
}

type MemoryStore struct {
	quotes []string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) Load(ctx context.Context, quotes []string) error {
	s.quotes = append(s.quotes, quotes...)
	return nil
}

func (s *MemoryStore) All(ctx context.Context) ([]string, error) {
	return s.quotes, nil
}

type server struct {
	store QuoteStore
}

func (srv server) getAllQuotes(w http.ResponseWriter, r *http.Request) {
	quotes, err := srv.store.All(r.Context())
	if err != nil {
		http.Error(w, `{"error": "failed to fetch quotes"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quotes)
}

func (srv server) getQuoteByIndex(w http.ResponseWriter, r *http.Request) {
	quotes, err := srv.store.All(r.Context())
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

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, quotes[index])
}

func main() {
	var quotes []string
	if err := json.Unmarshal(quotesFile, &quotes); err != nil {
		log.Fatalf("Failed to parse quotes: %v\n", err)
	}

	var store QuoteStore
	if host := os.Getenv("REDIS_HOST"); host != "" {
		store = NewRedisStore(host)
		log.Printf("Using Redis storage at %s\n", host)
	} else {
		store = NewMemoryStore()
		log.Println("Using in-memory storage")
	}

	ctx := context.Background()
	if err := store.Load(ctx, quotes); err != nil {
		log.Fatalf("Failed to load quotes: %v\n", err)
	}

	srv := server{store: store}

	r := mux.NewRouter()
	r.HandleFunc("/quotes", srv.getAllQuotes).Methods("GET")
	r.HandleFunc("/quotes/{index}", srv.getQuoteByIndex).Methods("GET")

	addr := ":3000"
	fmt.Printf("Server running on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
