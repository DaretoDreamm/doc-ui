package llm

// WikiTools returns the tool definitions for wiki operations during LLM interactions.
func WikiTools() []Tool {
	return []Tool{
		{
			Name:        "search_wiki",
			Description: "Search the wiki for articles matching a query. Returns article titles, summaries, and file paths.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query string",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "read_article",
			Description: "Read the full content of a wiki article by its file path.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Relative file path of the article within the wiki, e.g. 'concepts/neural-networks.md'",
					},
				},
				"required": []string{"file_path"},
			},
		},
		{
			Name:        "write_article",
			Description: "Create or update a wiki article. The content should be valid markdown with YAML frontmatter.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Relative file path for the article, e.g. 'concepts/my-topic.md'",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Full markdown content including frontmatter",
					},
				},
				"required": []string{"file_path", "content"},
			},
		},
		{
			Name:        "list_articles",
			Description: "List all articles in the wiki, optionally filtered by category.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Optional category to filter by",
					},
				},
			},
		},
		{
			Name:        "read_raw_document",
			Description: "Read the content of a raw source document that was ingested into the knowledge base.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Relative file path within the raw/ directory",
					},
				},
				"required": []string{"file_path"},
			},
		},
		{
			Name:        "get_backlinks",
			Description: "Get articles that link to a given article.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"article_title": map[string]interface{}{
						"type":        "string",
						"description": "Title of the article to find backlinks for",
					},
				},
				"required": []string{"article_title"},
			},
		},
	}
}

// NewProvider creates a new LLM provider based on the provider name.
func NewProvider(provider, apiKey, model, baseURL string) Provider {
	switch provider {
	case "anthropic":
		return NewAnthropic(apiKey, model, baseURL)
	case "ollama":
		return NewOllama(model, baseURL)
	default:
		return NewOpenAI(apiKey, model, baseURL)
	}
}
