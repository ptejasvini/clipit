package main

import (
	"clipit/internal/upload"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"
)

//go:embed web/*
var assets embed.FS

func main() {
	dataDirectory := os.Getenv("DATA_DIR")
	if dataDirectory == "" {
		dataDirectory = "data"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	uploadStore, err := upload.NewStore(dataDirectory)
	if err != nil {
		log.Fatalf("initialize storage: %v", err)
	}
	uploadHandler := upload.NewHandler(upload.NewService(uploadStore))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	uploadHandler.RegisterRoutes(mux)

	webFiles, err := fs.Sub(assets, "web")
	if err != nil {
		log.Fatalf("load web assets: %v", err)
	}
	indexHTML, err := fs.ReadFile(webFiles, "index.html")
	if err != nil {
		log.Fatalf("load index page: %v", err)
	}
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(webFiles))))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Clipit listening on :%s (data directory: %s)", port, dataDirectory)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server stopped: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
