package main

import (
	"log"
	"os"
	"strconv"

	"github.com/frkn/doc-ui/internal/crawler"
	"github.com/frkn/doc-ui/internal/database"
	"github.com/frkn/doc-ui/internal/llm"
	"github.com/frkn/doc-ui/internal/server"
	"github.com/frkn/doc-ui/internal/wiki"
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

	// Wiki manager
	wikiMgr := wiki.NewManager(db)

	// LLM provider (optional — KB features work without it, just no compilation/Q&A)
	var llmProvider llm.Provider
	llmProviderName := envOrDefault("LLM_PROVIDER", "")
	llmAPIKey := envOrDefault("LLM_API_KEY", "")
	llmModel := envOrDefault("LLM_MODEL", "")
	llmBaseURL := envOrDefault("LLM_BASE_URL", "")

	if llmProviderName != "" && (llmAPIKey != "" || llmProviderName == "ollama") {
		llmProvider = llm.NewProvider(llmProviderName, llmAPIKey, llmModel, llmBaseURL)
		log.Printf("LLM provider: %s (model: %s)", llmProvider.Name(), llmModel)
	} else {
		log.Printf("No LLM provider configured — KB compilation and Q&A disabled")
	}

	srv := server.New(db, c, wikiMgr, llmProvider, port)

	log.Fatal(srv.Start())
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
