package database

import (
	"database/sql"
	"strings"

	"github.com/frkn/doc-ui/internal/models"
)

// Knowledge Base CRUD

func (db *DB) CreateKnowledgeBase(name, slug, description, basePath, colorPalette, typography string) (*models.KnowledgeBase, error) {
	res, err := db.conn.Exec(
		`INSERT INTO knowledge_bases (name, slug, description, base_path, color_palette, typography, status)
		 VALUES (?, ?, ?, ?, ?, ?, 'idle')`,
		name, slug, description, basePath, colorPalette, typography,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return db.GetKnowledgeBase(id)
}

func (db *DB) GetKnowledgeBase(id int64) (*models.KnowledgeBase, error) {
	kb := &models.KnowledgeBase{}
	err := db.conn.QueryRow(
		`SELECT id, name, slug, COALESCE(description,''), base_path, article_count, raw_count,
		        status, COALESCE(llm_provider,''), COALESCE(llm_model,''),
		        COALESCE(color_palette,'ocean'), COALESCE(typography,'modern'),
		        created_at, updated_at
		 FROM knowledge_bases WHERE id = ?`, id,
	).Scan(&kb.ID, &kb.Name, &kb.Slug, &kb.Description, &kb.BasePath,
		&kb.ArticleCount, &kb.RawCount, &kb.Status, &kb.LLMProvider, &kb.LLMModel,
		&kb.ColorPalette, &kb.Typography, &kb.CreatedAt, &kb.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return kb, nil
}

func (db *DB) ListKnowledgeBases() ([]models.KnowledgeBase, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, slug, COALESCE(description,''), base_path, article_count, raw_count,
		        status, COALESCE(llm_provider,''), COALESCE(llm_model,''),
		        COALESCE(color_palette,'ocean'), COALESCE(typography,'modern'),
		        created_at, updated_at
		 FROM knowledge_bases ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var kbs []models.KnowledgeBase
	for rows.Next() {
		var kb models.KnowledgeBase
		if err := rows.Scan(&kb.ID, &kb.Name, &kb.Slug, &kb.Description, &kb.BasePath,
			&kb.ArticleCount, &kb.RawCount, &kb.Status, &kb.LLMProvider, &kb.LLMModel,
			&kb.ColorPalette, &kb.Typography, &kb.CreatedAt, &kb.UpdatedAt); err != nil {
			return nil, err
		}
		kbs = append(kbs, kb)
	}
	return kbs, rows.Err()
}

func (db *DB) DeleteKnowledgeBase(id int64) error {
	_, err := db.conn.Exec("DELETE FROM knowledge_bases WHERE id = ?", id)
	return err
}

func (db *DB) UpdateKBStatus(id int64, status string) error {
	_, err := db.conn.Exec(
		"UPDATE knowledge_bases SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		status, id,
	)
	return err
}

func (db *DB) UpdateKBCounts(id int64) error {
	_, err := db.conn.Exec(`
		UPDATE knowledge_bases SET
			article_count = (SELECT COUNT(*) FROM articles WHERE kb_id = ?),
			raw_count = (SELECT COUNT(*) FROM raw_documents WHERE kb_id = ?),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, id, id, id)
	return err
}

func (db *DB) UpdateKBLLMConfig(id int64, provider, model string) error {
	_, err := db.conn.Exec(
		"UPDATE knowledge_bases SET llm_provider = ?, llm_model = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		provider, model, id,
	)
	return err
}

// Article CRUD

func (db *DB) UpsertArticle(kbID int64, filePath, title, summary, content, contentHash, category, tags, source string, wordCount int) (*models.Article, error) {
	_, err := db.conn.Exec(`
		INSERT INTO articles (kb_id, file_path, title, summary, content, content_hash, category, tags, source, word_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(kb_id, file_path) DO UPDATE SET
			title = excluded.title,
			summary = excluded.summary,
			content = excluded.content,
			content_hash = excluded.content_hash,
			category = excluded.category,
			tags = excluded.tags,
			source = excluded.source,
			word_count = excluded.word_count,
			updated_at = CURRENT_TIMESTAMP`,
		kbID, filePath, title, summary, content, contentHash, category, tags, source, wordCount,
	)
	if err != nil {
		return nil, err
	}

	// Update FTS
	a := &models.Article{}
	err = db.conn.QueryRow(
		`SELECT id, kb_id, file_path, title, COALESCE(summary,''), content, COALESCE(content_hash,''),
		        COALESCE(category,''), COALESCE(tags,'[]'), backlink_count, word_count,
		        COALESCE(source,'manual'), created_at, updated_at
		 FROM articles WHERE kb_id = ? AND file_path = ?`, kbID, filePath,
	).Scan(&a.ID, &a.KBID, &a.FilePath, &a.Title, &a.Summary, &a.Content, &a.ContentHash,
		&a.Category, &a.Tags, &a.BacklinkCount, &a.WordCount, &a.Source, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}

	db.conn.Exec(`INSERT INTO articles_fts(rowid, title, summary, content, tags) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(rowid) DO UPDATE SET title=excluded.title, summary=excluded.summary, content=excluded.content, tags=excluded.tags`,
		a.ID, a.Title, a.Summary, a.Content, a.Tags)

	return a, nil
}

func (db *DB) GetArticle(id int64) (*models.Article, error) {
	a := &models.Article{}
	err := db.conn.QueryRow(
		`SELECT id, kb_id, file_path, title, COALESCE(summary,''), content, COALESCE(content_hash,''),
		        COALESCE(category,''), COALESCE(tags,'[]'), backlink_count, word_count,
		        COALESCE(source,'manual'), created_at, updated_at
		 FROM articles WHERE id = ?`, id,
	).Scan(&a.ID, &a.KBID, &a.FilePath, &a.Title, &a.Summary, &a.Content, &a.ContentHash,
		&a.Category, &a.Tags, &a.BacklinkCount, &a.WordCount, &a.Source, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (db *DB) GetArticleByPath(kbID int64, filePath string) (*models.Article, error) {
	a := &models.Article{}
	err := db.conn.QueryRow(
		`SELECT id, kb_id, file_path, title, COALESCE(summary,''), content, COALESCE(content_hash,''),
		        COALESCE(category,''), COALESCE(tags,'[]'), backlink_count, word_count,
		        COALESCE(source,'manual'), created_at, updated_at
		 FROM articles WHERE kb_id = ? AND file_path = ?`, kbID, filePath,
	).Scan(&a.ID, &a.KBID, &a.FilePath, &a.Title, &a.Summary, &a.Content, &a.ContentHash,
		&a.Category, &a.Tags, &a.BacklinkCount, &a.WordCount, &a.Source, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (db *DB) ListArticles(kbID int64) ([]models.Article, error) {
	rows, err := db.conn.Query(
		`SELECT id, kb_id, file_path, title, COALESCE(summary,''), '', COALESCE(content_hash,''),
		        COALESCE(category,''), COALESCE(tags,'[]'), backlink_count, word_count,
		        COALESCE(source,'manual'), created_at, updated_at
		 FROM articles WHERE kb_id = ? ORDER BY category, title`, kbID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var a models.Article
		if err := rows.Scan(&a.ID, &a.KBID, &a.FilePath, &a.Title, &a.Summary, &a.Content, &a.ContentHash,
			&a.Category, &a.Tags, &a.BacklinkCount, &a.WordCount, &a.Source, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, rows.Err()
}

func (db *DB) SearchArticles(kbID int64, query string, limit int) ([]models.Article, error) {
	// Try FTS5 first, fall back to LIKE
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}

	rows, err := db.conn.Query(
		`SELECT a.id, a.kb_id, a.file_path, a.title, COALESCE(a.summary,''), '', COALESCE(a.content_hash,''),
		        COALESCE(a.category,''), COALESCE(a.tags,'[]'), a.backlink_count, a.word_count,
		        COALESCE(a.source,'manual'), a.created_at, a.updated_at
		 FROM articles a
		 WHERE a.kb_id = ? AND (lower(a.title) LIKE lower(?) OR lower(a.content) LIKE lower(?) OR lower(a.summary) LIKE lower(?))
		 ORDER BY CASE WHEN lower(a.title) LIKE lower(?) THEN 0 ELSE 1 END, a.title
		 LIMIT ?`,
		kbID, "%"+q+"%", "%"+q+"%", "%"+q+"%", "%"+q+"%", limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var a models.Article
		if err := rows.Scan(&a.ID, &a.KBID, &a.FilePath, &a.Title, &a.Summary, &a.Content, &a.ContentHash,
			&a.Category, &a.Tags, &a.BacklinkCount, &a.WordCount, &a.Source, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, rows.Err()
}

func (db *DB) DeleteArticle(id int64) error {
	db.conn.Exec("DELETE FROM articles_fts WHERE rowid = ?", id)
	_, err := db.conn.Exec("DELETE FROM articles WHERE id = ?", id)
	return err
}

func (db *DB) DeleteArticlesByKB(kbID int64) error {
	_, err := db.conn.Exec("DELETE FROM articles WHERE kb_id = ?", kbID)
	return err
}

// Raw Documents

func (db *DB) InsertRawDocument(kbID int64, filePath, fileName, mimeType string, size int64) error {
	_, err := db.conn.Exec(
		`INSERT OR IGNORE INTO raw_documents (kb_id, file_path, file_name, mime_type, size)
		 VALUES (?, ?, ?, ?, ?)`,
		kbID, filePath, fileName, mimeType, size,
	)
	return err
}

func (db *DB) ListRawDocuments(kbID int64) ([]models.RawDocument, error) {
	rows, err := db.conn.Query(
		`SELECT id, kb_id, file_path, file_name, COALESCE(mime_type,''), size, processed, COALESCE(summary,''), created_at
		 FROM raw_documents WHERE kb_id = ? ORDER BY created_at DESC`, kbID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.RawDocument
	for rows.Next() {
		var d models.RawDocument
		if err := rows.Scan(&d.ID, &d.KBID, &d.FilePath, &d.FileName, &d.MimeType, &d.Size, &d.Processed, &d.Summary, &d.CreatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (db *DB) ListUnprocessedRawDocuments(kbID int64) ([]models.RawDocument, error) {
	rows, err := db.conn.Query(
		`SELECT id, kb_id, file_path, file_name, COALESCE(mime_type,''), size, processed, COALESCE(summary,''), created_at
		 FROM raw_documents WHERE kb_id = ? AND processed = 0 ORDER BY created_at`, kbID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []models.RawDocument
	for rows.Next() {
		var d models.RawDocument
		if err := rows.Scan(&d.ID, &d.KBID, &d.FilePath, &d.FileName, &d.MimeType, &d.Size, &d.Processed, &d.Summary, &d.CreatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (db *DB) MarkRawProcessed(id int64, summary string) error {
	_, err := db.conn.Exec(
		"UPDATE raw_documents SET processed = 1, summary = ? WHERE id = ?",
		summary, id,
	)
	return err
}

func (db *DB) GetRawDocument(id int64) (*models.RawDocument, error) {
	d := &models.RawDocument{}
	err := db.conn.QueryRow(
		`SELECT id, kb_id, file_path, file_name, COALESCE(mime_type,''), size, processed, COALESCE(summary,''), created_at
		 FROM raw_documents WHERE id = ?`, id,
	).Scan(&d.ID, &d.KBID, &d.FilePath, &d.FileName, &d.MimeType, &d.Size, &d.Processed, &d.Summary, &d.CreatedAt)
	return d, err
}

// Backlinks

func (db *DB) UpsertBacklink(sourceID, targetID int64, context string) error {
	_, err := db.conn.Exec(
		`INSERT INTO backlinks (source_article_id, target_article_id, context) VALUES (?, ?, ?)
		 ON CONFLICT(source_article_id, target_article_id) DO UPDATE SET context = excluded.context`,
		sourceID, targetID, context,
	)
	return err
}

func (db *DB) GetBacklinksForArticle(articleID int64) ([]models.Backlink, error) {
	rows, err := db.conn.Query(
		`SELECT b.source_article_id, b.target_article_id, COALESCE(b.context,''),
		        COALESCE(s.title,''), COALESCE(t.title,'')
		 FROM backlinks b
		 LEFT JOIN articles s ON s.id = b.source_article_id
		 LEFT JOIN articles t ON t.id = b.target_article_id
		 WHERE b.target_article_id = ?`, articleID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Backlink
	for rows.Next() {
		var bl models.Backlink
		if err := rows.Scan(&bl.SourceArticleID, &bl.TargetArticleID, &bl.Context, &bl.SourceTitle, &bl.TargetTitle); err != nil {
			return nil, err
		}
		links = append(links, bl)
	}
	return links, rows.Err()
}

func (db *DB) GetAllBacklinks(kbID int64) ([]models.Backlink, error) {
	rows, err := db.conn.Query(
		`SELECT b.source_article_id, b.target_article_id, COALESCE(b.context,''),
		        COALESCE(s.title,''), COALESCE(t.title,'')
		 FROM backlinks b
		 JOIN articles s ON s.id = b.source_article_id AND s.kb_id = ?
		 JOIN articles t ON t.id = b.target_article_id AND t.kb_id = ?`, kbID, kbID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.Backlink
	for rows.Next() {
		var bl models.Backlink
		if err := rows.Scan(&bl.SourceArticleID, &bl.TargetArticleID, &bl.Context, &bl.SourceTitle, &bl.TargetTitle); err != nil {
			return nil, err
		}
		links = append(links, bl)
	}
	return links, rows.Err()
}

func (db *DB) ClearBacklinksForArticle(sourceID int64) error {
	_, err := db.conn.Exec("DELETE FROM backlinks WHERE source_article_id = ?", sourceID)
	return err
}

func (db *DB) UpdateBacklinkCounts(kbID int64) error {
	_, err := db.conn.Exec(`
		UPDATE articles SET backlink_count = (
			SELECT COUNT(*) FROM backlinks WHERE target_article_id = articles.id
		) WHERE kb_id = ?`, kbID)
	return err
}

// Chat Messages

func (db *DB) InsertChatMessage(kbID int64, role, content string) (*models.ChatMessage, error) {
	res, err := db.conn.Exec(
		"INSERT INTO chat_messages (kb_id, role, content) VALUES (?, ?, ?)",
		kbID, role, content,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	msg := &models.ChatMessage{}
	err = db.conn.QueryRow(
		"SELECT id, kb_id, role, content, created_at FROM chat_messages WHERE id = ?", id,
	).Scan(&msg.ID, &msg.KBID, &msg.Role, &msg.Content, &msg.CreatedAt)
	return msg, err
}

func (db *DB) ListChatMessages(kbID int64, limit int) ([]models.ChatMessage, error) {
	rows, err := db.conn.Query(
		`SELECT id, kb_id, role, content, created_at FROM chat_messages
		 WHERE kb_id = ? ORDER BY created_at DESC LIMIT ?`, kbID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []models.ChatMessage
	for rows.Next() {
		var m models.ChatMessage
		if err := rows.Scan(&m.ID, &m.KBID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	// Reverse to get chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, rows.Err()
}

func (db *DB) ClearChatHistory(kbID int64) error {
	_, err := db.conn.Exec("DELETE FROM chat_messages WHERE kb_id = ?", kbID)
	return err
}

// Health Checks

func (db *DB) InsertHealthCheck(kbID int64, checkType, severity, message string, articleID int64) error {
	_, err := db.conn.Exec(
		"INSERT INTO health_checks (kb_id, type, severity, message, article_id) VALUES (?, ?, ?, ?, ?)",
		kbID, checkType, severity, message, articleID,
	)
	return err
}

func (db *DB) ListHealthChecks(kbID int64) ([]models.HealthCheck, error) {
	rows, err := db.conn.Query(
		`SELECT id, kb_id, type, severity, message, article_id, resolved, created_at
		 FROM health_checks WHERE kb_id = ? AND resolved = 0
		 ORDER BY CASE severity WHEN 'error' THEN 0 WHEN 'warning' THEN 1 ELSE 2 END, created_at DESC`, kbID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []models.HealthCheck
	for rows.Next() {
		var h models.HealthCheck
		if err := rows.Scan(&h.ID, &h.KBID, &h.Type, &h.Severity, &h.Message, &h.ArticleID, &h.Resolved, &h.CreatedAt); err != nil {
			return nil, err
		}
		checks = append(checks, h)
	}
	return checks, rows.Err()
}

func (db *DB) ResolveHealthCheck(id int64) error {
	_, err := db.conn.Exec("UPDATE health_checks SET resolved = 1 WHERE id = ?", id)
	return err
}

func (db *DB) ClearHealthChecks(kbID int64) error {
	_, err := db.conn.Exec("DELETE FROM health_checks WHERE kb_id = ?", kbID)
	return err
}

// Graph data helper

func (db *DB) GetGraphData(kbID int64) ([]models.Article, []models.Backlink, error) {
	articles, err := db.ListArticles(kbID)
	if err != nil {
		return nil, nil, err
	}
	backlinks, err := db.GetAllBacklinks(kbID)
	if err != nil {
		return nil, nil, err
	}
	return articles, backlinks, nil
}

// Slug helper - check uniqueness
func (db *DB) SlugExists(slug string) bool {
	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM knowledge_bases WHERE slug = ?", slug).Scan(&count)
	return count > 0
}

// Conn exposes the underlying connection for use by other packages (e.g., wiki sync)
func (db *DB) Conn() *sql.DB {
	return db.conn
}
