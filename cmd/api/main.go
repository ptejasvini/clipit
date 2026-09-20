package main

import (
	"clipit/internal/upload"
	"fmt"
	"net/http"
)

func main() {
	//new multipler for handlers
	mux := http.NewServeMux()

	uploadStore := upload.NewStore()
	uploadService := upload.NewService(uploadStore)
	uploadHandler := upload.NewHandler(uploadService)

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /uploads", uploadHandler.CreateUpload)
	fmt.Println("ClipIt is starting ")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}

}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
