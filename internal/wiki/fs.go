package wiki

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InitKBDirectory creates the standard directory structure for a knowledge base.
func InitKBDirectory(basePath string) error {
	dirs := []string{
		filepath.Join(basePath, "raw"),
		filepath.Join(basePath, "wiki"),
		filepath.Join(basePath, "wiki", "concepts"),
		filepath.Join(basePath, "wiki", "summaries"),
		filepath.Join(basePath, "wiki", "_meta"),
		filepath.Join(basePath, "output"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", d, err)
		}
	}

	// Create initial index.md if it doesn't exist
	indexPath := filepath.Join(basePath, "wiki", "index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		content := "---\ntitle: Index\ncategory: root\n---\n\n# Knowledge Base Index\n\nThis index is auto-maintained by the wiki compiler.\n"
		if err := os.WriteFile(indexPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write index: %w", err)
		}
	}
	return nil
}

// ReadArticle reads a markdown file from the wiki directory.
func ReadArticle(basePath, relPath string) ([]byte, error) {
	return os.ReadFile(filepath.Join(basePath, "wiki", relPath))
}

// WriteArticle writes a markdown file to the wiki directory, creating subdirectories as needed.
func WriteArticle(basePath, relPath string, content []byte) error {
	fullPath := filepath.Join(basePath, "wiki", relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, content, 0644)
}

// ReadRawDocument reads a file from the raw directory.
func ReadRawDocument(basePath, relPath string) ([]byte, error) {
	return os.ReadFile(filepath.Join(basePath, "raw", relPath))
}

// WriteRawDocument writes a file to the raw directory.
func WriteRawDocument(basePath, fileName string, content []byte) error {
	fullPath := filepath.Join(basePath, "raw", fileName)
	return os.WriteFile(fullPath, content, 0644)
}

// ListMarkdownFiles returns all .md files in the wiki directory, with paths relative to wiki/.
func ListMarkdownFiles(basePath string) ([]string, error) {
	wikiDir := filepath.Join(basePath, "wiki")
	var files []string
	err := filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip _meta directory
			if info.Name() == "_meta" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			rel, _ := filepath.Rel(wikiDir, path)
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

// ListRawFiles returns all files in the raw directory, with paths relative to raw/.
func ListRawFiles(basePath string) ([]string, error) {
	rawDir := filepath.Join(basePath, "raw")
	var files []string
	err := filepath.Walk(rawDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(rawDir, path)
		files = append(files, rel)
		return nil
	})
	return files, err
}

// ComputeHash returns the SHA256 hex hash of content.
func ComputeHash(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)
}

// DeleteArticle removes a wiki article file.
func DeleteArticle(basePath, relPath string) error {
	return os.Remove(filepath.Join(basePath, "wiki", relPath))
}

// FileSize returns the size of a file in the raw directory.
func FileSize(basePath, relPath string) (int64, error) {
	info, err := os.Stat(filepath.Join(basePath, "raw", relPath))
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// MimeTypeFromExt returns a basic MIME type based on file extension.
func MimeTypeFromExt(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".md", ".markdown":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".pdf":
		return "application/pdf"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
