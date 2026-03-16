package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"yadro.com/course/config"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

func getPong(w http.ResponseWriter, r *http.Request) {
	if _, err := fmt.Fprintln(w, "pong"); err != nil {
		logger.Error("cant write respnsse", "err", err)
	}
}

func getHello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "empty name", http.StatusBadRequest)
		return
	}
	if _, err := fmt.Fprintf(w, "Hello, %s!\n", name); err != nil {
		logger.Error("cant write response", "err", err)
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		logger.Error("cleanenv error", "err", err)
		return
	}
	port := ":" + cfg.Port

	srv := http.NewServeMux()
	srv.HandleFunc("GET /ping", getPong)

	srv.HandleFunc("GET /hello", getHello)

	if err := http.ListenAndServe(port, srv); err != http.ErrServerClosed {
		logger.Error("error during ListenAndServe", "err", err)
		return
	}
}
