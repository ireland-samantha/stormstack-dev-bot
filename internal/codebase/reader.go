// Package codebase provides file reading utilities.
package codebase

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Reader provides file reading operations within a repository.
type Reader struct {
	repoPath string
}

// NewReader creates a new file reader.
func NewReader(repoPath string) *Reader {
	return &Reader{repoPath: repoPath}
}

// ReadFile reads a file and returns its content.
func (r *Reader) ReadFile(path string) (string, error) {
	fullPath, err := r.resolvePath(path)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// ReadFileLines reads specific lines from a file.
func (r *Reader) ReadFileLines(path string, startLine, endLine int) (string, error) {
	fullPath, err := r.resolvePath(path)
	if err != nil {
		return "", err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if startLine > 0 && lineNum < startLine {
			continue
		}
		if endLine > 0 && lineNum > endLine {
			break
		}
		lines = append(lines, fmt.Sprintf("%4d | %s", lineNum, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return strings.Join(lines, "\n"), nil
}

// FileExists checks if a file exists.
func (r *Reader) FileExists(path string) bool {
	fullPath, err := r.resolvePath(path)
	if err != nil {
		return false
	}

	_, err = os.Stat(fullPath)
	return err == nil
}

// GetFileInfo returns information about a file.
func (r *Reader) GetFileInfo(path string) (*FileInfo, error) {
	fullPath, err := r.resolvePath(path)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	lineCount := 0
	if !stat.IsDir() {
		lineCount, _ = r.countLines(fullPath)
	}

	return &FileInfo{
		Path:      path,
		Size:      stat.Size(),
		IsDir:     stat.IsDir(),
		ModTime:   stat.ModTime().Unix(),
		LineCount: lineCount,
	}, nil
}

// FileInfo contains metadata about a file.
type FileInfo struct {
	Path      string
	Size      int64
	IsDir     bool
	ModTime   int64
	LineCount int
}

// resolvePath resolves a relative path to an absolute path within the repo.
func (r *Reader) resolvePath(path string) (string, error) {
	// Clean the path
	path = filepath.Clean(path)

	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")

	// Join with repo path
	fullPath := filepath.Join(r.repoPath, path)

	// Resolve to absolute path
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Security check: ensure path is within repo
	absRepoPath, err := filepath.Abs(r.repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve repo path: %w", err)
	}

	if !strings.HasPrefix(absPath, absRepoPath) {
		return "", fmt.Errorf("path escapes repository: %s", path)
	}

	return absPath, nil
}

// countLines counts the number of lines in a file.
func (r *Reader) countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}

	return count, scanner.Err()
}

// GetRepoPath returns the repository path.
func (r *Reader) GetRepoPath() string {
	return r.repoPath
}
