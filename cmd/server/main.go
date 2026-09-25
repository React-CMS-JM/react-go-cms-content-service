package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"react-go-cms-content-service/internal/content"
	"react-go-cms-content-service/internal/platform"
)

func main() {
	cfg := platform.LoadConfig("8082")
	db, err := platform.OpenDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("database: %v", err)
	}
	store := content.NewStore(db, cfg.AuthServiceURL)
	h := content.NewHandler(store, cfg.JWTSecret, cfg.JWTIssuer)
	mux := chi.NewRouter()
	h.Register(mux)
	addr := ":" + cfg.Port
	log.Printf("react-go-cms-content-service listening on %s", addr)
	server := &http.Server{
		Addr:              addr,
		Handler:           platform.CORS(cfg.CORSOrigins, mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
