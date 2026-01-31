// Package codebase provides code search utilities.
package codebase

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Searcher provides code search operations.
type Searcher struct {
	repoPath string
}

// NewSearcher creates a new code searcher.
func NewSearcher(repoPath string) *Searcher {
	return &Searcher{repoPath: repoPath}
}

// SearchResult represents a single search match.
type SearchResult struct {
	File    string
	Line    int
	Content string
}

// SearchCode searches for a pattern in the codebase.
func (s *Searcher) SearchCode(pattern, path string, caseSensitive bool, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 50
	}

	// Compile regex
	flags := ""
	if !caseSensitive {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	// Determine search root
	searchRoot := s.repoPath
	if path != "" {
		searchRoot = filepath.Join(s.repoPath, path)
	}

	var results []SearchResult

	err = filepath.WalkDir(searchRoot, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories
		if d.IsDir() {
			// Skip hidden directories and common non-code directories
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "target" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip binary files and large files
		if !isTextFile(filePath) {
			return nil
		}

		// Search in file
		matches, err := s.searchInFile(filePath, re)
		if err != nil {
			return nil // Skip errors
		}

		// Get relative path
		relPath, _ := filepath.Rel(s.repoPath, filePath)

		for _, match := range matches {
			if len(results) >= maxResults {
				return filepath.SkipAll
			}
			results = append(results, SearchResult{
				File:    relPath,
				Line:    match.Line,
				Content: match.Content,
			})
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, fmt.Errorf("search error: %w", err)
	}

	return results, nil
}

// searchInFile searches for matches in a single file.
func (s *Searcher) searchInFile(path string, re *regexp.Regexp) ([]SearchResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []SearchResult
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			results = append(results, SearchResult{
				Line:    lineNum,
				Content: strings.TrimSpace(line),
			})
		}
	}

	return results, scanner.Err()
}

// ListFiles lists files matching a glob pattern.
func (s *Searcher) ListFiles(pattern string) ([]string, error) {
	// Ensure pattern is relative
	pattern = strings.TrimPrefix(pattern, "/")

	// Use doublestar for glob matching
	fullPattern := filepath.Join(s.repoPath, pattern)
	matches, err := doublestar.FilepathGlob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	// Convert to relative paths and filter out directories
	var files []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}

		relPath, err := filepath.Rel(s.repoPath, match)
		if err != nil {
			continue
		}

		files = append(files, relPath)
	}

	// Sort for consistent output
	sort.Strings(files)

	return files, nil
}

// GetTree returns the directory structure.
func (s *Searcher) GetTree(path string, maxDepth int) (string, error) {
	if maxDepth <= 0 {
		maxDepth = 3
	}

	root := s.repoPath
	if path != "" {
		root = filepath.Join(s.repoPath, path)
	}

	var builder strings.Builder
	err := s.buildTree(&builder, root, "", 0, maxDepth)
	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

// buildTree recursively builds a tree representation.
func (s *Searcher) buildTree(builder *strings.Builder, path, prefix string, depth, maxDepth int) error {
	if depth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Filter and sort entries
	var filteredEntries []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files and common non-essential directories
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() && (name == "node_modules" || name == "vendor" || name == "target" || name == "build" || name == "__pycache__") {
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}

	for i, entry := range filteredEntries {
		isLast := i == len(filteredEntries)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		builder.WriteString(prefix + connector + entry.Name())
		if entry.IsDir() {
			builder.WriteString("/")
		}
		builder.WriteString("\n")

		if entry.IsDir() && depth < maxDepth {
			newPrefix := prefix
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			s.buildTree(builder, filepath.Join(path, entry.Name()), newPrefix, depth+1, maxDepth)
		}
	}

	return nil
}

// FindTests finds test files for a given source file.
func (s *Searcher) FindTests(sourceFile string) ([]string, error) {
	ext := filepath.Ext(sourceFile)
	base := strings.TrimSuffix(filepath.Base(sourceFile), ext)
	dir := filepath.Dir(sourceFile)

	var patterns []string

	switch ext {
	case ".java":
		// Java: Look for *Test.java in same package or test directory
		patterns = append(patterns,
			filepath.Join(dir, base+"Test.java"),
			filepath.Join(strings.Replace(dir, "main", "test", 1), base+"Test.java"),
			"**/"+base+"Test.java",
		)
	case ".go":
		// Go: Look for *_test.go in same directory
		patterns = append(patterns,
			filepath.Join(dir, base+"_test.go"),
			filepath.Join(dir, "*_test.go"),
		)
	case ".js", ".ts", ".jsx", ".tsx":
		// JavaScript/TypeScript: Look for *.test.* or *.spec.*
		patterns = append(patterns,
			filepath.Join(dir, base+".test"+ext),
			filepath.Join(dir, base+".spec"+ext),
			filepath.Join(dir, "__tests__", base+ext),
			"**/__tests__/"+base+".*",
			"**/"+base+".test.*",
			"**/"+base+".spec.*",
		)
	case ".py":
		// Python: Look for test_*.py or *_test.py
		patterns = append(patterns,
			filepath.Join(dir, "test_"+base+".py"),
			filepath.Join(dir, base+"_test.py"),
			"**/test_"+base+".py",
			"**/"+base+"_test.py",
		)
	default:
		// Generic: Look for *Test* or *_test* files
		patterns = append(patterns,
			"**/"+base+"*[Tt]est*",
		)
	}

	var testFiles []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		matches, err := s.ListFiles(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if !seen[match] {
				seen[match] = true
				testFiles = append(testFiles, match)
			}
		}
	}

	return testFiles, nil
}

// isTextFile checks if a file is likely a text file.
func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	// Common text file extensions
	textExtensions := map[string]bool{
		".go": true, ".java": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".py": true, ".rb": true, ".rs": true, ".c": true, ".cpp": true, ".h": true, ".hpp": true,
		".cs": true, ".php": true, ".swift": true, ".kt": true, ".scala": true,
		".html": true, ".css": true, ".scss": true, ".sass": true, ".less": true,
		".json": true, ".yaml": true, ".yml": true, ".toml": true, ".xml": true,
		".md": true, ".txt": true, ".rst": true, ".adoc": true,
		".sh": true, ".bash": true, ".zsh": true, ".fish": true,
		".sql": true, ".graphql": true, ".proto": true,
		".env": true, ".gitignore": true, ".dockerignore": true,
		".mod": true, ".sum": true, ".lock": true,
		"": true, // Files without extension (README, Makefile, etc.)
	}

	if textExtensions[ext] {
		return true
	}

	// Check common filenames without extensions
	base := filepath.Base(path)
	textFiles := map[string]bool{
		"Makefile": true, "Dockerfile": true, "Jenkinsfile": true,
		"README": true, "LICENSE": true, "CHANGELOG": true,
		"Gemfile": true, "Rakefile": true, "Vagrantfile": true,
	}

	return textFiles[base]
}

// FormatSearchResults formats search results for display.
func FormatSearchResults(results []SearchResult) string {
	var builder strings.Builder

	for _, r := range results {
		builder.WriteString(fmt.Sprintf("%s:%d: %s\n", r.File, r.Line, r.Content))
	}

	return builder.String()
}
