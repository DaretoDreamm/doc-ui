package crawler

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/frkn/doc-ui/internal/database"
	"github.com/frkn/doc-ui/internal/models"
	"github.com/gocolly/colly/v2"
)

var userAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
}

type Crawler struct {
	db          *database.DB
	maxDepth    int
	delayMs     int
	sem         chan struct{}
	mu          sync.Mutex
	progressMap map[int64]*models.CrawlProgress
	cancelMap   map[int64]*atomic.Bool
}

func New(db *database.DB, maxDepth, delayMs, maxConcurrent int) *Crawler {
	return &Crawler{
		db:          db,
		maxDepth:    maxDepth,
		delayMs:     delayMs,
		sem:         make(chan struct{}, maxConcurrent),
		progressMap: make(map[int64]*models.CrawlProgress),
		cancelMap:   make(map[int64]*atomic.Bool),
	}
}

func (c *Crawler) GetProgress(projectID int64) *models.CrawlProgress {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.progressMap[projectID]; ok {
		cp := *p
		return &cp
	}
	return nil
}

func (c *Crawler) setProgress(projectID int64, p *models.CrawlProgress) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.progressMap[projectID] = p
}

func (c *Crawler) StopCrawl(projectID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cancel, ok := c.cancelMap[projectID]; ok {
		cancel.Store(true)
	}
}

func (c *Crawler) StartCrawl(project *models.Project) {
	cancel := &atomic.Bool{}
	c.mu.Lock()
	c.cancelMap[project.ID] = cancel
	c.mu.Unlock()

	c.sem <- struct{}{}
	go func() {
		defer func() {
			<-c.sem
			c.mu.Lock()
			delete(c.cancelMap, project.ID)
			c.mu.Unlock()
		}()
		c.crawl(project, cancel)
	}()
}

func (c *Crawler) crawl(project *models.Project, cancel *atomic.Bool) {
	c.setProgress(project.ID, &models.CrawlProgress{
		ProjectID: project.ID,
		Status:    "crawling",
	})
	c.db.UpdateProjectStatus(project.ID, "crawling", 0, "")

	parsed, err := url.Parse(project.BaseURL)
	if err != nil {
		c.finishWithError(project.ID, fmt.Sprintf("Invalid URL: %v", err))
		return
	}

	// Build path prefix to restrict crawling scope.
	// Only restrict for deep paths (3+ segments like /develop/ui/compose/).
	// Shallow paths (like /docs/) don't restrict - rely on MaxDepth instead.
	pathPrefix := parsed.Path
	if !strings.HasSuffix(pathPrefix, "/") {
		lastSlash := strings.LastIndex(pathPrefix, "/")
		if lastSlash > 0 {
			pathPrefix = pathPrefix[:lastSlash+1]
		} else {
			pathPrefix = "/"
		}
	}
	segments := strings.Count(strings.Trim(pathPrefix, "/"), "/") + 1
	if strings.Trim(pathPrefix, "/") == "" {
		segments = 0
	}
	usePathFilter := segments >= 3

	for i, ua := range userAgents {
		if cancel.Load() {
			c.finishStopped(project.ID)
			return
		}
		log.Printf("crawl attempt %d for %s (prefix: %s, filter: %v)", i+1, project.BaseURL, pathPrefix, usePathFilter)
		count := c.doCrawl(project, parsed, pathPrefix, usePathFilter, ua, cancel)
		if cancel.Load() {
			c.finishStopped(project.ID)
			return
		}
		if count > 0 {
			c.db.UpdateProjectStatus(project.ID, "done", count, "")
			c.setProgress(project.ID, &models.CrawlProgress{
				ProjectID: project.ID,
				Status:    "done",
				PageCount: count,
			})
			return
		}
	}

	c.finishWithError(project.ID, "No pages could be crawled. The site may use JavaScript rendering (SPA), block automated requests, or have no extractable content.")
}

func (c *Crawler) doCrawl(project *models.Project, parsed *url.URL, pathPrefix string, usePathFilter bool, userAgent string, cancel *atomic.Bool) int {
	var pageCount atomic.Int32
	var navOrder atomic.Int32
	var lastPageTime atomic.Int64
	var done atomic.Bool
	lastPageTime.Store(time.Now().UnixMilli())

	opts := []colly.CollectorOption{
		colly.AllowedDomains(parsed.Host),
		colly.MaxDepth(c.maxDepth),
		colly.Async(true),
		colly.UserAgent(userAgent),
	}
	if usePathFilter {
		escapedPrefix := regexp.QuoteMeta(pathPrefix)
		pathFilter := regexp.MustCompile(fmt.Sprintf(`^https?://%s%s`, regexp.QuoteMeta(parsed.Host), escapedPrefix))
		opts = append(opts, colly.URLFilters(pathFilter))
	}
	collector := colly.NewCollector(opts...)

	collector.SetRequestTimeout(30 * time.Second)

	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       time.Duration(c.delayMs) * time.Millisecond,
	})

	collector.DisallowedURLFilters = []*regexp.Regexp{
		regexp.MustCompile(`\.(css|js|png|jpg|jpeg|gif|svg|ico|woff|woff2|ttf|eot|pdf|zip|tar|gz|xml|json|rss|atom)(\?.*)?$`),
		regexp.MustCompile(`[?&](utm_|ref=|source=)`),
		regexp.MustCompile(`oauth|signin|login|accounts\.google\.com`),
	}

	collector.OnRequest(func(r *colly.Request) {
		if done.Load() || cancel.Load() {
			r.Abort()
			return
		}
		lastTime := time.UnixMilli(lastPageTime.Load())
		if time.Since(lastTime) > 60*time.Second {
			done.Store(true)
			r.Abort()
			log.Printf("idle timeout, aborting (%d pages)", pageCount.Load())
		}
	})

	collector.OnHTML("html", func(e *colly.HTMLElement) {
		if done.Load() || cancel.Load() {
			return
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(e.Response.Body)))
		if err != nil {
			return
		}

		// Rewrite relative image/source URLs to absolute
		pageURL := e.Request.URL
		doc.Find("img[src], source[src], video[src], audio[src]").Each(func(i int, s *goquery.Selection) {
			if src, exists := s.Attr("src"); exists {
				if src != "" && !strings.HasPrefix(src, "http") && !strings.HasPrefix(src, "//") && !strings.HasPrefix(src, "data:") {
					if resolved, err := pageURL.Parse(src); err == nil {
						s.SetAttr("src", resolved.String())
					}
				}
			}
		})
		doc.Find("img[srcset], source[srcset]").Each(func(i int, s *goquery.Selection) {
			if srcset, exists := s.Attr("srcset"); exists {
				var newParts []string
				for _, part := range strings.Split(srcset, ",") {
					part = strings.TrimSpace(part)
					fields := strings.Fields(part)
					if len(fields) >= 1 && !strings.HasPrefix(fields[0], "http") && !strings.HasPrefix(fields[0], "//") && !strings.HasPrefix(fields[0], "data:") {
						if resolved, err := pageURL.Parse(fields[0]); err == nil {
							fields[0] = resolved.String()
						}
					}
					newParts = append(newParts, strings.Join(fields, " "))
				}
				s.SetAttr("srcset", strings.Join(newParts, ", "))
			}
		})

		title := extractTitle(doc)
		content := extractContent(doc)

		if strings.TrimSpace(content) == "" {
			return
		}

		textOnly := doc.Find("body").Text()
		if len(strings.Fields(textOnly)) < 10 {
			return
		}

		reqURL := e.Request.URL.String()
		pagePath := e.Request.URL.Path
		if pagePath == "" {
			pagePath = "/"
		}

		relPath := strings.TrimPrefix(pagePath, strings.TrimSuffix(pathPrefix, "/"))
		if relPath == "" || relPath == "/" {
			relPath = "index"
		}
		relPath = strings.TrimPrefix(relPath, "/")
		relPath = strings.TrimSuffix(relPath, "/")

		depth := e.Request.Depth
		order := int(navOrder.Add(1))

		if err := c.db.InsertPage(project.ID, reqURL, relPath, title, content, order, depth); err != nil {
			return
		}

		lastPageTime.Store(time.Now().UnixMilli())
		count := int(pageCount.Add(1))
		c.setProgress(project.ID, &models.CrawlProgress{
			ProjectID:  project.ID,
			Status:     "crawling",
			PageCount:  count,
			CurrentURL: reqURL,
		})
		c.db.UpdateProjectStatus(project.ID, "crawling", count, "")
	})

	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if done.Load() || cancel.Load() {
			return
		}
		link := e.Attr("href")
		e.Request.Visit(e.Request.AbsoluteURL(link))
	})

	collector.OnError(func(r *colly.Response, err error) {
		if !done.Load() && !cancel.Load() {
			log.Printf("crawl error %s (status %d): %v", r.Request.URL, r.StatusCode, err)
		}
	})

	collector.Visit(project.BaseURL)
	collector.Wait()

	return int(pageCount.Load())
}

func (c *Crawler) finishWithError(projectID int64, msg string) {
	c.db.UpdateProjectStatus(projectID, "error", 0, msg)
	c.setProgress(projectID, &models.CrawlProgress{
		ProjectID: projectID,
		Status:    "error",
		ErrorMsg:  msg,
	})
}

func (c *Crawler) finishStopped(projectID int64) {
	p, err := c.db.GetProject(projectID)
	if err != nil {
		return
	}
	if p.PageCount > 0 {
		c.db.UpdateProjectStatus(projectID, "done", p.PageCount, "")
		c.setProgress(projectID, &models.CrawlProgress{
			ProjectID: projectID,
			Status:    "done",
			PageCount: p.PageCount,
		})
	} else {
		c.db.UpdateProjectStatus(projectID, "error", 0, "Crawl stopped")
		c.setProgress(projectID, &models.CrawlProgress{
			ProjectID: projectID,
			Status:    "error",
			ErrorMsg:  "Crawl stopped",
		})
	}
}
