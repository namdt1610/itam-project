package main

import (
	"fmt"
	"io"
	"net/http"
)

func handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		body, _ := io.ReadAll(r.Body)
		fmt.Printf("New Data Received: %s\n", string(body))
		w.WriteHeader(http.StatusOK)
	}
}

func main() {
	http.HandleFunc("/api/report", handleReport)
	fmt.Println("Server ITAM starting on :8080...")
	http.ListenAndServe(":8080", nil)
}