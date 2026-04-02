# DocUI

A self-hosted documentation generator that crawls any documentation website and renders it as a customizable, browsable reader. Built with Go, HTMX, and Templ. Runs in Docker.

Give it a URL, pick a layout, choose a color palette and typography -- DocUI crawls the site and builds a clean documentation viewer you can browse locally.

## Features

- **Crawl any documentation site** -- paste a URL and DocUI extracts all pages under that path
- **3 layout templates** -- Minimal (single-column reader), Sidebar (left nav + content), Modern (three-column with TOC and dark mode toggle)
- **10 color palettes** -- Noir, Snow, Ocean, Forest, Sunset, Grape, Rose, Amber, Teal, Slate. Each has light and dark mode variants.
- **5 typography options** -- Modern (Syne + Plus Jakarta Sans), Classic (Playfair Display + Source Serif 4), Geometric (Outfit + DM Sans), Mono (Space Mono), Humanist (Fraunces + Nunito Sans)
- **Spotlight search** -- press Cmd/Ctrl+K to search across all pages in a project by title or content
- **Smart crawling** -- automatically retries with different user agents for sites that block crawlers (e.g. developer.android.com). Path-scoped filtering for deep URL hierarchies. Idle timeout prevents stuck crawls.
- **URL reuse** -- if a URL was already crawled, new projects with the same URL instantly copy the pages instead of re-crawling
- **Stop and delete** -- crawls can be stopped mid-way or deleted at any time
- **Image handling** -- relative image URLs are rewritten to absolute so images load correctly from the original source
- **SQLite storage** -- all data persists in a single file, survives container restarts

## Quick Start

```
docker compose up --build
```

Open http://localhost:4000

## Configuration

Environment variables in `docker-compose.yml`:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `4000` | Server port |
| `DB_PATH` | `/app/data/docui.db` | SQLite database path |
| `MAX_CRAWL_DEPTH` | `3` | Maximum link depth to follow from the entry URL |
| `CRAWL_DELAY_MS` | `500` | Delay between requests to avoid rate limiting |

## How It Works

1. You provide a documentation URL (e.g. `https://htmx.org/docs/`)
2. DocUI crawls the URL and all linked pages on the same domain
3. For each page, it extracts the main content using priority selectors (`<main>`, `<article>`, `[role="main"]`, etc.), strips navigation/headers/footers, and sanitizes the HTML
4. Pages are stored in SQLite with their extracted title, content, and URL path
5. The documentation viewer renders pages using your chosen layout, palette, and typography

The crawler handles common obstacles:
- Sites that redirect to OAuth (e.g. Google properties) -- retries with a crawler-compatible user agent
- Deep URL hierarchies (3+ path segments) -- restricts crawling to the same path prefix
- Shallow URL structures -- allows crawling across the full domain within the depth limit
- Rate limiting -- configurable delay between requests

## Tech Stack

- **Go** -- backend server, crawler, content extraction
- **Templ** -- type-safe HTML templates
- **HTMX** -- dynamic interactions without JavaScript frameworks
- **Colly** -- web crawling
- **Goquery** -- HTML parsing and content extraction
- **Bluemonday** -- HTML sanitization
- **SQLite** -- storage (pure-Go driver, no CGO)
- **Tailwind CSS** -- styling via CDN
- **Docker** -- multi-stage build, single container deployment

## Project Structure

```
doc-ui/
  main.go                          # Entry point
  Dockerfile                       # Multi-stage build
  docker-compose.yml               # Container config
  internal/
    server/                        # HTTP server, routes, handlers
    crawler/                       # Colly crawler + content extractor
    database/                      # SQLite queries and migrations
    models/                        # Data types, palettes, typography
  templates/
    layouts/                       # App shell (homepage)
    pages/                         # Page templates
    components/                    # HTMX components (cards, forms, search)
    doctemplates/                  # Doc viewer templates (minimal, sidebar, modern)
  data/                            # SQLite database (Docker volume)
```

## License

MIT
