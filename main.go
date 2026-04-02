package main

import (
	"log"
	"os"
	"strconv"

	"github.com/frkn/doc-ui/internal/crawler"
	"github.com/frkn/doc-ui/internal/database"
	"github.com/frkn/doc-ui/internal/server"
)

func main() {
	dbPath := envOrDefault("DB_PATH", "./data/docui.db")
	port, _ := strconv.Atoi(envOrDefault("PORT", "4000"))
	maxDepth, _ := strconv.Atoi(envOrDefault("MAX_CRAWL_DEPTH", "3"))
	crawlDelay, _ := strconv.Atoi(envOrDefault("CRAWL_DELAY_MS", "500"))

	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer db.Close()

	c := crawler.New(db, maxDepth, crawlDelay, 3)
	srv := server.New(db, c, port)

	log.Fatal(srv.Start())
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
