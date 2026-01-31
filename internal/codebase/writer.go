// Package codebase provides file writing utilities.
package codebase

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Writer provides file writing operations within a repository.
type Writer struct {
	repoPath string
}

// NewWriter creates a new file writer.
func NewWriter(repoPath string) *Writer {
	return &Writer{repoPath: repoPath}
}

// WriteFile writes content to a file, creating directories as needed.
func (w *Writer) WriteFile(path, content string) error {
	fullPath, err := w.resolvePath(path)
	if err != nil {
		return err
	}

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// EditFile makes a targeted edit to a file.
func (w *Writer) EditFile(path, oldText, newText string) error {
	fullPath, err := w.resolvePath(path)
	if err != nil {
		return err
	}

	// Read existing content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(content)

	// Check that old text exists and is unique
	count := strings.Count(contentStr, oldText)
	if count == 0 {
		return fmt.Errorf("old_text not found in file")
	}
	if count > 1 {
		return fmt.Errorf("old_text found %d times in file (must be unique)", count)
	}

	// Replace
	newContent := strings.Replace(contentStr, oldText, newText, 1)

	// Write back
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// DeleteFile deletes a file.
func (w *Writer) DeleteFile(path string) error {
	fullPath, err := w.resolvePath(path)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// CreateDirectory creates a directory.
func (w *Writer) CreateDirectory(path string) error {
	fullPath, err := w.resolvePath(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// resolvePath resolves a relative path to an absolute path within the repo.
func (w *Writer) resolvePath(path string) (string, error) {
	// Clean the path
	path = filepath.Clean(path)

	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")

	// Join with repo path
	fullPath := filepath.Join(w.repoPath, path)

	// Resolve to absolute path
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Security check: ensure path is within repo
	absRepoPath, err := filepath.Abs(w.repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve repo path: %w", err)
	}

	if !strings.HasPrefix(absPath, absRepoPath) {
		return "", fmt.Errorf("path escapes repository: %s", path)
	}

	return absPath, nil
}

// GetRepoPath returns the repository path.
func (w *Writer) GetRepoPath() string {
	return w.repoPath
}
