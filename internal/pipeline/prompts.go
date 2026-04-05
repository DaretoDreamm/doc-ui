package pipeline

import (
	"fmt"
	"strings"
)

func SummarizePrompt(docContent, fileName string) string {
	return fmt.Sprintf(`You are a research assistant helping build a knowledge base wiki.

Summarize the following document concisely. Extract the key concepts, findings, and important details.
Your summary should be 2-4 paragraphs and capture the essential information.

Document name: %s

Document content:
---
%s
---

Provide a clear, factual summary.`, fileName, truncate(docContent, 12000))
}

func ExtractConceptsPrompt(summary string, existingConcepts []string) string {
	existing := "None yet"
	if len(existingConcepts) > 0 {
		existing = strings.Join(existingConcepts, ", ")
	}
	return fmt.Sprintf(`You are a research assistant analyzing a document summary to extract key concepts.

Existing concepts in the wiki: %s

Document summary:
---
%s
---

Extract 3-8 key concepts from this summary. For each concept, provide:
1. A short title (2-5 words, suitable as an article title)
2. A one-line description

Return your answer as a JSON array:
[{"title": "Concept Title", "description": "One-line description"}]

Only include genuinely distinct concepts. If a concept already exists in the wiki, skip it unless the new information significantly expands it.`, existing, summary)
}

func WriteArticlePrompt(concept, description string, sources []string, relatedArticles []string) string {
	sourcesText := "No source material provided."
	if len(sources) > 0 {
		sourcesText = strings.Join(sources, "\n\n---\n\n")
	}
	related := "None"
	if len(relatedArticles) > 0 {
		related = strings.Join(relatedArticles, ", ")
	}
	return fmt.Sprintf(`You are a technical writer building a knowledge base wiki.

Write a comprehensive article about: %s
Description: %s

Related existing articles (link to these using [[Article Title]] syntax): %s

Source material:
---
%s
---

Write the article in markdown format. Include:
1. YAML frontmatter with title, category, tags, and a brief summary
2. Clear sections with headers
3. Links to related articles using [[Article Title]] wiki-link syntax
4. Code examples if relevant
5. A "See Also" section at the end linking to related concepts

The article should be thorough but concise. Target 300-800 words.
Start with the YAML frontmatter block (---).`, concept, description, related, truncate(sourcesText, 10000))
}

func UpdateArticlePrompt(existingArticle, newInfo string) string {
	return fmt.Sprintf(`You are a technical writer updating a knowledge base article with new information.

Current article:
---
%s
---

New information to integrate:
---
%s
---

Update the article to incorporate the new information. Maintain the existing structure and frontmatter format.
Do not remove existing content unless it contradicts the new information.
Add [[wiki-links]] to any new concepts mentioned.
Return the complete updated article including frontmatter.`, existingArticle, truncate(newInfo, 8000))
}

func HealthCheckPrompt(articleSummaries string) string {
	return fmt.Sprintf(`You are a knowledge base quality analyst. Review these article summaries and identify issues.

Articles in the wiki:
---
%s
---

Check for:
1. Contradictions between articles
2. Topics that seem to be missing (gaps in coverage)
3. Articles that cover very similar topics (potential duplicates)
4. Concepts mentioned but not linked or explained

Return your findings as a JSON array:
[{"type": "inconsistency|missing-coverage|duplicate|dead-link", "severity": "error|warning|suggestion", "message": "Description of the issue", "article": "Affected article title or empty"}]

Only report genuine issues. If the wiki looks healthy, return an empty array [].`, truncate(articleSummaries, 15000))
}

func QASystemPrompt(wikiSummary string) string {
	return fmt.Sprintf(`You are a knowledgeable research assistant with access to a personal knowledge base wiki.

The wiki contains the following articles:
%s

Use the available tools to search and read articles to answer the user's questions.
Always base your answers on the wiki content. If the wiki doesn't contain relevant information, say so.
When referencing information from specific articles, mention them by name.
Format your responses in clear markdown with headers and bullet points where appropriate.`, truncate(wikiSummary, 8000))
}

func CompileIndexPrompt(articles []string) string {
	list := strings.Join(articles, "\n")
	return fmt.Sprintf(`You are maintaining a knowledge base wiki index.

Current articles:
%s

Generate an updated index.md file that organizes these articles by category.
Use this format:

---
title: Index
category: root
---

# Knowledge Base Index

## Category Name
- [[Article Title]] — brief description

Group related articles together. Include a brief description for each.
Return the complete markdown file including frontmatter.`, list)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... [truncated]"
}
