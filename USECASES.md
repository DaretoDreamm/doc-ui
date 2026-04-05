# DocUI Use Cases & Guide

## What is DocUI?

DocUI is a self-hosted tool that combines **documentation crawling** with an **LLM-powered knowledge base**. You can crawl any documentation site into a beautiful reader, then import that content (or your own files) into a personal wiki that an LLM can compile, cross-reference, and answer questions about.

---

## Quick Start

```bash
# Basic (no LLM — crawling + manual KB works)
docker compose up -d

# With LLM enabled (compilation + Q&A)
LLM_PROVIDER=anthropic LLM_API_KEY=sk-ant-... LLM_MODEL=claude-sonnet-4-20250514 docker compose up -d

# Or with OpenAI
LLM_PROVIDER=openai LLM_API_KEY=sk-... LLM_MODEL=gpt-4o docker compose up -d

# Or with local Ollama (no API key needed)
LLM_PROVIDER=ollama LLM_MODEL=llama3 LLM_BASE_URL=http://host.docker.internal:11434 docker compose up -d
```

Open `http://localhost:4000`.

---

## Use Case 1: Mirror Documentation for Offline Reading

**Scenario:** You want a local, beautifully styled copy of a documentation site.

1. Go to `http://localhost:4000`
2. Enter a project name (e.g., "HTMX Docs") and the documentation URL
3. Pick a layout (Minimal, Sidebar, or Modern), color palette, and typography
4. Click **Start Crawling**
5. Watch the real-time progress as pages are discovered
6. Once done, click **Open** to browse the documentation

**Features available:**
- Full-text search across all pages (Cmd+K)
- Sidebar navigation with page filtering
- Previous/Next page navigation
- 10 color themes and 5 typography presets
- Works completely offline after crawling

**Tips:**
- If a site blocks the crawler, DocUI automatically retries with a different user agent
- Re-crawl anytime to pick up documentation updates
- If you crawl the same URL twice, pages are reused instantly (no duplicate requests)

---

## Use Case 2: Build a Research Wiki from Crawled Documentation

**Scenario:** You've crawled several documentation sites and want to build a unified knowledge base across them.

1. Crawl your documentation sources as projects (Use Case 1)
2. Go to **Knowledge Bases** in the nav bar
3. Create a new KB (e.g., "Frontend Stack") with your preferred theme
4. On the KB dashboard, find the **Import from Crawled Docs** dropdown
5. Select a crawled project and click **Import**
6. Repeat for each documentation source you want to include
7. Click **Browse Wiki** to see all imported pages organized by source project

**What happens:**
- Each crawled page becomes a wiki article with frontmatter metadata
- Articles are organized under `imported/{project-name}/` in the wiki
- Content is immediately searchable and browsable
- No LLM required for this workflow

---

## Use Case 3: Build a Knowledge Base from Your Own Documents

**Scenario:** You have research papers, notes, articles, or other documents you want to organize into a wiki.

1. Create a KB at `/kb`
2. Upload your files via the **Upload Source Documents** section
   - Supported formats: `.md`, `.txt`, `.html`, `.json`, `.csv`, `.pdf`
   - Max 50MB per file
3. Click **Sync from Disk** to index any files you placed directly in the filesystem
4. Browse uploaded documents in the raw documents list

**Obsidian integration:** The KB directory lives at `data/kb/{slug}/`. You can open this folder in Obsidian, edit `.md` files directly, then click **Sync from Disk** in DocUI to pick up changes. The `[[wiki-link]]` syntax works in both Obsidian and DocUI.

---

## Use Case 4: LLM-Compiled Wiki (The Full Pipeline)

**Scenario:** You have raw source documents and want an LLM to organize them into a structured, cross-referenced wiki.

**Prerequisites:** Set `LLM_PROVIDER`, `LLM_API_KEY`, and `LLM_MODEL` environment variables.

1. Create a KB and upload/import your source documents (Use Cases 2 or 3)
2. On the KB dashboard, click **Compile Wiki**
3. Watch the pipeline progress through its stages:
   - **Summarizing** — LLM reads each raw document and generates a summary
   - **Extracting** — LLM identifies key concepts across all summaries
   - **Generating** — LLM writes a wiki article for each concept, with `[[wiki-links]]` to related articles
   - **Syncing** — Articles are indexed into the database
   - **Indexing** — LLM generates an updated `index.md` table of contents
4. Browse the compiled wiki with full backlink tracking and search

**What the LLM produces:**
- Concept articles in `wiki/concepts/` with structured content
- Cross-references using `[[Article Title]]` wiki-link syntax
- YAML frontmatter with title, category, tags, and summary
- An auto-maintained index page organizing articles by category

**Tips:**
- You can stop compilation at any time and resume later
- Run compilation again after adding new raw documents — it only processes new/unprocessed files
- The LLM enhances but doesn't replace: manually-written articles in `wiki/` are preserved

---

## Use Case 5: Ask Questions About Your Knowledge Base

**Scenario:** Your wiki has grown large and you want to query it conversationally.

**Prerequisites:** LLM configured.

1. Open your KB and click **Q&A Chat**
2. Type a question about any topic covered in your wiki
3. The LLM streams its response in real-time, drawing on your wiki's content
4. The sidebar shows all wiki articles — click any to open in a new tab for reference
5. Conversation history is preserved between sessions

**How it works:**
- The LLM receives a summary of all your articles as context
- Your last 20 messages are included for conversational continuity
- Responses reference specific articles by name
- Clear chat history anytime with the delete endpoint

**Example questions:**
- "What are the key differences between X and Y based on my notes?"
- "Summarize everything I have on topic Z"
- "What connections exist between concept A and concept B?"
- "What topics am I missing coverage on?"

---

## Use Case 6: Visualize Concept Relationships

**Scenario:** You want to see how your wiki articles connect to each other.

1. Open your KB and click **Concept Graph**
2. An interactive force-directed graph shows all articles as nodes
3. Edges represent `[[wiki-links]]` between articles
4. Node size reflects article word count
5. Node color groups articles by category

**Interactions:**
- Drag nodes to rearrange the layout
- Click any node to navigate to that article
- Hover to see article details

**When this is useful:**
- Identifying isolated articles (orphans with no connections)
- Discovering unexpected relationships between topics
- Getting a high-level overview of your wiki's structure
- Finding clusters of related concepts

---

## Use Case 7: Audit Wiki Quality with Health Checks

**Scenario:** Your wiki has grown and you want to ensure quality and completeness.

1. Open your KB and click **Health Check**
2. Click **Run Health Check**
3. Review issues grouped by severity:

| Check | Severity | What it finds |
|-------|----------|---------------|
| **Orphan articles** | Warning | Articles with no incoming links from other articles |
| **Dead links** | Error | `[[wiki-links]]` pointing to articles that don't exist |
| **Stale articles** | Suggestion | Articles not updated since new raw documents were added |
| **Inconsistencies** | Varies | LLM-detected contradictions between articles (requires LLM) |
| **Missing coverage** | Suggestion | Topics mentioned but not explained (requires LLM) |
| **Duplicates** | Warning | Articles covering very similar topics (requires LLM) |

4. Dismiss resolved issues or use them as a checklist for improving your wiki

**Tips:**
- Orphan and dead-link checks work without an LLM (pure database queries)
- Run health checks periodically as your wiki grows
- Dead links are the most actionable — they indicate missing articles you should create

---

## Use Case 8: Combine Crawled Docs + Personal Notes

**Scenario:** You're learning a new technology. You want to merge official documentation with your own study notes and have an LLM tie them together.

1. Crawl the official documentation (e.g., "React Docs" from reactjs.org)
2. Create a KB (e.g., "React Learning")
3. Import the crawled docs into your KB
4. Upload your personal study notes, bookmarked articles, etc.
5. Run **Compile Wiki** — the LLM will:
   - Summarize your notes alongside the official docs
   - Identify concepts that appear in both
   - Generate unified articles that cross-reference official docs with your notes
   - Create a structured index
6. Use **Q&A Chat** to quiz yourself or explore connections

---

## Use Case 9: Obsidian as a Frontend

**Scenario:** You prefer Obsidian's editing experience but want DocUI's LLM compilation.

The wiki lives as plain `.md` files on disk at `data/kb/{slug}/wiki/`. This directory is fully Obsidian-compatible:

1. Create a KB in DocUI
2. Open `data/kb/{slug}/wiki/` as an Obsidian vault
3. Write and edit articles in Obsidian using `[[wiki-links]]`
4. In DocUI, click **Sync from Disk** to index your changes
5. Use DocUI for compilation, Q&A, health checks, and the concept graph
6. Any articles the LLM generates appear in Obsidian automatically

**Directory structure:**
```
data/kb/my-research/
  raw/                  # Upload source documents here
  wiki/                 # Your Obsidian vault
    index.md            # Auto-maintained table of contents
    concepts/           # LLM-generated concept articles
    imported/           # Articles imported from crawled projects
    summaries/          # Document summaries
    _meta/              # Metadata (ignored by indexer)
  output/               # LLM-generated outputs
```

---

## Environment Variables Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `4000` | HTTP server port |
| `DB_PATH` | `./data/docui.db` | SQLite database location |
| `MAX_CRAWL_DEPTH` | `3` | Maximum link depth when crawling |
| `CRAWL_DELAY_MS` | `500` | Delay between crawl requests (rate limiting) |
| `LLM_PROVIDER` | _(none)_ | `anthropic`, `openai`, or `ollama` |
| `LLM_API_KEY` | _(none)_ | API key (not needed for Ollama) |
| `LLM_MODEL` | _(none)_ | Model ID (e.g., `claude-sonnet-4-20250514`, `gpt-4o`, `llama3`) |
| `LLM_BASE_URL` | _(auto)_ | Custom API endpoint (for self-hosted or proxied APIs) |

---

## What Works Without an LLM

| Feature | LLM Required? |
|---------|---------------|
| Crawl documentation sites | No |
| Browse crawled docs with themes | No |
| Create knowledge bases | No |
| Upload raw documents | No |
| Import crawled docs as wiki articles | No |
| Browse wiki articles | No |
| Search articles | No |
| View concept graph | No |
| Orphan & dead-link health checks | No |
| Sync from disk (Obsidian workflow) | No |
| Wiki-link resolution & backlinks | No |
| **Compile wiki from raw docs** | **Yes** |
| **Q&A chat** | **Yes** |
| **LLM-powered health checks** | **Yes** |

---

## Theming Options

### Color Palettes
Noir, Snow, Ocean, Forest, Sunset, Grape, Rose, Amber, Teal, Slate — each with light and dark mode variants.

### Typography Presets
| Preset | Display Font | Body Font | Code Font |
|--------|-------------|-----------|-----------|
| Modern | Syne | Plus Jakarta Sans | Fira Code |
| Classic | Playfair Display | Source Serif 4 | IBM Plex Mono |
| Geometric | Outfit | DM Sans | JetBrains Mono |
| Mono | Space Mono | Space Mono | Space Mono |
| Humanist | Fraunces | Nunito Sans | Fira Code |

### Doc Viewer Layouts
- **Minimal** — Clean single-column reader
- **Sidebar** — Left navigation panel with content area
- **Modern** — Three-column: sidebar + content + table of contents
