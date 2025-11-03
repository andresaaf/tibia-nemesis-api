package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"tibia-nemesis-api/internal/config"
	httpapi "tibia-nemesis-api/internal/http"
	"tibia-nemesis-api/internal/scraper"
	"tibia-nemesis-api/internal/service"
	"tibia-nemesis-api/internal/store"
)

func main() {
	cfg := config.Load()

	st, err := store.NewSQLite(cfg.DBPath)
	if err != nil {
		log.Fatalf("store init: %v", err)
	}
	defer st.Close()

	scr := scraper.New(cfg)
	svc := service.New(st, scr, cfg)
	go svc.StartScheduler()

	r := httpapi.NewRouter(svc)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("tibia-nemesis-api listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}
