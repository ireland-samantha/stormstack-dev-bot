// Package executor provides output parsing utilities.
package executor

import (
	"regexp"
	"strings"
)

// TestFailure represents a parsed test failure.
type TestFailure struct {
	TestName string
	File     string
	Line     int
	Message  string
	Expected string
	Actual   string
}

// BuildError represents a parsed build error.
type BuildError struct {
	File    string
	Line    int
	Column  int
	Message string
	Type    string // "error" or "warning"
}

// AnalyzeOutput analyzes command output for failures and errors.
func AnalyzeOutput(output string) *AnalysisResult {
	result := &AnalysisResult{
		Raw: output,
	}

	// Detect build system
	switch {
	case strings.Contains(output, "BUILD FAILURE") || strings.Contains(output, "[ERROR]"):
		result.Type = "maven"
		result.BuildErrors = parseMavenErrors(output)
	case strings.Contains(output, "FAILED") && strings.Contains(output, "go test"):
		result.Type = "go"
		result.TestFailures = parseGoTestFailures(output)
	case strings.Contains(output, "npm ERR!"):
		result.Type = "npm"
		result.BuildErrors = parseNpmErrors(output)
	case strings.Contains(output, "FAIL") && (strings.Contains(output, "jest") || strings.Contains(output, "vitest")):
		result.Type = "jest"
		result.TestFailures = parseJestFailures(output)
	case strings.Contains(output, "error:") && strings.Contains(output, "cargo"):
		result.Type = "cargo"
		result.BuildErrors = parseCargoErrors(output)
	case strings.Contains(output, "FAILURES!") || strings.Contains(output, "Tests run:"):
		result.Type = "junit"
		result.TestFailures = parseJUnitFailures(output)
	default:
		result.Type = "unknown"
		result.BuildErrors = parseGenericErrors(output)
	}

	// Set success flag
	result.Success = len(result.BuildErrors) == 0 && len(result.TestFailures) == 0

	return result
}

// AnalysisResult contains the parsed output analysis.
type AnalysisResult struct {
	Type         string
	Success      bool
	BuildErrors  []BuildError
	TestFailures []TestFailure
	Raw          string
}

// Summary returns a human-readable summary.
func (r *AnalysisResult) Summary() string {
	if r.Success {
		return "Build/tests passed successfully."
	}

	var sb strings.Builder

	if len(r.BuildErrors) > 0 {
		sb.WriteString("Build Errors:\n")
		for i, err := range r.BuildErrors {
			if i >= 5 {
				sb.WriteString("  ... and more errors\n")
				break
			}
			if err.File != "" {
				sb.WriteString("  • " + err.File)
				if err.Line > 0 {
					sb.WriteString(":" + string(rune('0'+err.Line)))
				}
				sb.WriteString(": ")
			} else {
				sb.WriteString("  • ")
			}
			sb.WriteString(err.Message + "\n")
		}
	}

	if len(r.TestFailures) > 0 {
		sb.WriteString("Test Failures:\n")
		for i, fail := range r.TestFailures {
			if i >= 5 {
				sb.WriteString("  ... and more failures\n")
				break
			}
			sb.WriteString("  • " + fail.TestName + "\n")
			if fail.Message != "" {
				sb.WriteString("    " + fail.Message + "\n")
			}
		}
	}

	return sb.String()
}

// parseMavenErrors parses Maven build output.
func parseMavenErrors(output string) []BuildError {
	var errors []BuildError

	// Maven error pattern: [ERROR] /path/to/File.java:[line,col] error message
	errorRe := regexp.MustCompile(`\[ERROR\]\s+(/[^:]+):?\[?(\d+)?,?(\d+)?\]?\s*(.+)`)

	for _, match := range errorRe.FindAllStringSubmatch(output, -1) {
		err := BuildError{
			File:    match[1],
			Message: match[4],
			Type:    "error",
		}
		if match[2] != "" {
			err.Line = parseIntSafe(match[2])
		}
		if match[3] != "" {
			err.Column = parseIntSafe(match[3])
		}
		errors = append(errors, err)
	}

	return errors
}

// parseGoTestFailures parses Go test output.
func parseGoTestFailures(output string) []TestFailure {
	var failures []TestFailure

	// Go test failure pattern
	failRe := regexp.MustCompile(`--- FAIL: (\S+)`)
	fileRe := regexp.MustCompile(`\s+(\S+\.go):(\d+):\s*(.+)`)

	lines := strings.Split(output, "\n")
	var currentTest string

	for i, line := range lines {
		if match := failRe.FindStringSubmatch(line); match != nil {
			currentTest = match[1]
		}

		if currentTest != "" {
			if match := fileRe.FindStringSubmatch(line); match != nil {
				failures = append(failures, TestFailure{
					TestName: currentTest,
					File:     match[1],
					Line:     parseIntSafe(match[2]),
					Message:  match[3],
				})
			}
		}

		// Look for expected/actual
		if strings.Contains(line, "expected") || strings.Contains(line, "got") {
			if i+1 < len(lines) && currentTest != "" && len(failures) > 0 {
				last := &failures[len(failures)-1]
				if strings.Contains(line, "expected") {
					last.Expected = strings.TrimSpace(line)
				}
				if strings.Contains(line, "got") {
					last.Actual = strings.TrimSpace(line)
				}
			}
		}
	}

	return failures
}

// parseNpmErrors parses npm error output.
func parseNpmErrors(output string) []BuildError {
	var errors []BuildError

	errorRe := regexp.MustCompile(`npm ERR!\s*(.+)`)

	for _, match := range errorRe.FindAllStringSubmatch(output, -1) {
		if !strings.HasPrefix(match[1], "code") && !strings.HasPrefix(match[1], "errno") {
			errors = append(errors, BuildError{
				Message: match[1],
				Type:    "error",
			})
		}
	}

	return errors
}

// parseJestFailures parses Jest test output.
func parseJestFailures(output string) []TestFailure {
	var failures []TestFailure

	// Jest failure pattern
	failRe := regexp.MustCompile(`✕\s+(.+)`)
	fileRe := regexp.MustCompile(`at\s+\S+\s+\(([^:]+):(\d+):(\d+)\)`)

	for _, match := range failRe.FindAllStringSubmatch(output, -1) {
		failure := TestFailure{
			TestName: strings.TrimSpace(match[1]),
		}

		// Try to find file location
		if fileMatch := fileRe.FindStringSubmatch(output); fileMatch != nil {
			failure.File = fileMatch[1]
			failure.Line = parseIntSafe(fileMatch[2])
		}

		failures = append(failures, failure)
	}

	return failures
}

// parseCargoErrors parses Cargo (Rust) build output.
func parseCargoErrors(output string) []BuildError {
	var errors []BuildError

	// Cargo error pattern
	errorRe := regexp.MustCompile(`error(?:\[E\d+\])?: (.+)\n\s+-->\s+([^:]+):(\d+):(\d+)`)

	for _, match := range errorRe.FindAllStringSubmatch(output, -1) {
		errors = append(errors, BuildError{
			Message: match[1],
			File:    match[2],
			Line:    parseIntSafe(match[3]),
			Column:  parseIntSafe(match[4]),
			Type:    "error",
		})
	}

	return errors
}

// parseJUnitFailures parses JUnit test output.
func parseJUnitFailures(output string) []TestFailure {
	var failures []TestFailure

	// Look for test failure markers
	failRe := regexp.MustCompile(`(?:FAILURE!|Tests run:.*Failures: [1-9])`)
	testRe := regexp.MustCompile(`(\w+)\((\w+)\).*FAILED`)

	if failRe.MatchString(output) {
		for _, match := range testRe.FindAllStringSubmatch(output, -1) {
			failures = append(failures, TestFailure{
				TestName: match[2] + "." + match[1],
			})
		}
	}

	return failures
}

// parseGenericErrors tries to parse generic error patterns.
func parseGenericErrors(output string) []BuildError {
	var errors []BuildError

	// Generic patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)error:\s*(.+)`),
		regexp.MustCompile(`(?i)fatal:\s*(.+)`),
		regexp.MustCompile(`([^:\s]+):(\d+):\s*error:\s*(.+)`),
	}

	for _, pattern := range patterns {
		for _, match := range pattern.FindAllStringSubmatch(output, 10) {
			err := BuildError{Type: "error"}
			if len(match) == 2 {
				err.Message = match[1]
			} else if len(match) == 4 {
				err.File = match[1]
				err.Line = parseIntSafe(match[2])
				err.Message = match[3]
			}
			errors = append(errors, err)
		}
	}

	return errors
}

// parseIntSafe parses an integer, returning 0 on error.
func parseIntSafe(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
