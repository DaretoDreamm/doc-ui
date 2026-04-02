package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/frkn/doc-ui/internal/models"
	_ "github.com/glebarez/go-sqlite"
)

type DB struct {
	conn *sql.DB
}

func New(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			base_url TEXT NOT NULL,
			template TEXT NOT NULL DEFAULT 'minimal',
			color_palette TEXT NOT NULL DEFAULT 'ocean',
			status TEXT NOT NULL DEFAULT 'pending',
			page_count INTEGER DEFAULT 0,
			error_msg TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS pages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			url TEXT NOT NULL,
			path TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '',
			nav_order INTEGER DEFAULT 0,
			depth INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(project_id, url)
		);

		CREATE INDEX IF NOT EXISTS idx_pages_project ON pages(project_id);
	`)
	if err != nil {
		return err
	}

	db.conn.Exec("ALTER TABLE projects ADD COLUMN color_palette TEXT NOT NULL DEFAULT 'ocean'")
	db.conn.Exec("ALTER TABLE projects ADD COLUMN typography TEXT NOT NULL DEFAULT 'modern'")
	return nil
}

func (db *DB) CreateProject(name, baseURL, template, colorPalette, typography string) (*models.Project, error) {
	res, err := db.conn.Exec(
		"INSERT INTO projects (name, base_url, template, color_palette, typography, status) VALUES (?, ?, ?, ?, ?, 'pending')",
		name, baseURL, template, colorPalette, typography,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return db.GetProject(id)
}

func (db *DB) GetProject(id int64) (*models.Project, error) {
	p := &models.Project{}
	err := db.conn.QueryRow(
		"SELECT id, name, base_url, template, COALESCE(color_palette,'ocean'), COALESCE(typography,'modern'), status, page_count, COALESCE(error_msg,''), created_at, updated_at FROM projects WHERE id = ?", id,
	).Scan(&p.ID, &p.Name, &p.BaseURL, &p.Template, &p.ColorPalette, &p.Typography, &p.Status, &p.PageCount, &p.ErrorMsg, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (db *DB) ListProjects() ([]models.Project, error) {
	rows, err := db.conn.Query(
		"SELECT id, name, base_url, template, COALESCE(color_palette,'ocean'), COALESCE(typography,'modern'), status, page_count, COALESCE(error_msg,''), created_at, updated_at FROM projects ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.Template, &p.ColorPalette, &p.Typography, &p.Status, &p.PageCount, &p.ErrorMsg, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (db *DB) UpdateProjectStatus(id int64, status string, pageCount int, errorMsg string) error {
	_, err := db.conn.Exec(
		"UPDATE projects SET status = ?, page_count = ?, error_msg = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		status, pageCount, errorMsg, id,
	)
	return err
}

func (db *DB) GetDoneProjectByURL(baseURL string) (*models.Project, error) {
	p := &models.Project{}
	err := db.conn.QueryRow(
		"SELECT id, name, base_url, template, COALESCE(color_palette,'ocean'), COALESCE(typography,'modern'), status, page_count, COALESCE(error_msg,''), created_at, updated_at FROM projects WHERE base_url = ? AND status = 'done' AND page_count > 1 ORDER BY page_count DESC LIMIT 1", baseURL,
	).Scan(&p.ID, &p.Name, &p.BaseURL, &p.Template, &p.ColorPalette, &p.Typography, &p.Status, &p.PageCount, &p.ErrorMsg, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (db *DB) CopyPages(srcProjectID, dstProjectID int64) (int, error) {
	res, err := db.conn.Exec(
		"INSERT INTO pages (project_id, url, path, title, content, nav_order, depth) SELECT ?, url, path, title, content, nav_order, depth FROM pages WHERE project_id = ?",
		dstProjectID, srcProjectID,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (db *DB) DeleteProject(id int64) error {
	_, err := db.conn.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

func (db *DB) DeletePages(projectID int64) error {
	_, err := db.conn.Exec("DELETE FROM pages WHERE project_id = ?", projectID)
	return err
}

func (db *DB) InsertPage(projectID int64, url, path, title, content string, navOrder, depth int) error {
	_, err := db.conn.Exec(
		"INSERT OR IGNORE INTO pages (project_id, url, path, title, content, nav_order, depth) VALUES (?, ?, ?, ?, ?, ?, ?)",
		projectID, url, path, title, content, navOrder, depth,
	)
	return err
}

func (db *DB) GetPages(projectID int64) ([]models.Page, error) {
	rows, err := db.conn.Query(
		"SELECT id, project_id, url, path, title, content, nav_order, depth, created_at FROM pages WHERE project_id = ? ORDER BY nav_order",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []models.Page
	for rows.Next() {
		var p models.Page
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.URL, &p.Path, &p.Title, &p.Content, &p.NavOrder, &p.Depth, &p.CreatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func (db *DB) SearchPages(projectID int64, query string, limit int) ([]models.Page, error) {
	q := "%" + query + "%"
	rows, err := db.conn.Query(
		`SELECT id, project_id, url, path, title, '', nav_order, depth, created_at
		 FROM pages WHERE project_id = ? AND (lower(title) LIKE lower(?) OR lower(content) LIKE lower(?))
		 ORDER BY CASE WHEN lower(title) LIKE lower(?) THEN 0 ELSE 1 END, nav_order
		 LIMIT ?`,
		projectID, q, q, q, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []models.Page
	for rows.Next() {
		var p models.Page
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.URL, &p.Path, &p.Title, &p.Content, &p.NavOrder, &p.Depth, &p.CreatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func (db *DB) GetPageByPath(projectID int64, path string) (*models.Page, error) {
	p := &models.Page{}
	err := db.conn.QueryRow(
		"SELECT id, project_id, url, path, title, content, nav_order, depth, created_at FROM pages WHERE project_id = ? AND path = ?",
		projectID, path,
	).Scan(&p.ID, &p.ProjectID, &p.URL, &p.Path, &p.Title, &p.Content, &p.NavOrder, &p.Depth, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}
