package fs

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/theapemachine/animal/ai/tool/editor/doc"
	"github.com/theapemachine/animal/ai/tool/editor/workspace"
	"github.com/theapemachine/animal/lease"
)

/*
Document reads and writes files under a sandboxed workspace root.
*/
type Document struct {
	root string
}

/*
NewDocument opens a filesystem-backed document rooted at root.
*/
func NewDocument(root string, leaseCoordinator *lease.Coordinator) (*Document, error) {
	if leaseCoordinator == nil {
		return nil, fmt.Errorf("fs: lease coordinator is required")
	}

	abs, err := workspace.Join(root, ".")
	if err != nil {
		return nil, err
	}

	return &Document{root: abs}, nil
}

/*
Info returns metadata for a workspace-relative path.
*/
func (document *Document) Info(_ context.Context, path string) (doc.DocumentInfo, error) {
	abs, err := workspace.Join(document.root, path)
	if err != nil {
		return doc.DocumentInfo{}, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		return doc.DocumentInfo{}, err
	}

	return doc.DocumentInfo{
		Path:    path,
		Size:    info.Size(),
		Created: info.ModTime(),
		Updated: info.ModTime(),
		Mode:    info.Mode().String(),
	}, nil
}

/*
Read returns numbered content for an optional 1-based line range.
*/
func (document *Document) Read(_ context.Context, params doc.ReadParams) (doc.ReadResult, error) {
	abs, err := workspace.Join(document.root, params.Path)
	if err != nil {
		return doc.ReadResult{}, err
	}

	file, err := os.Open(abs)
	if err != nil {
		return doc.ReadResult{}, err
	}
	defer file.Close()

	lines, err := document.readLines(file)
	if err != nil {
		return doc.ReadResult{}, err
	}

	start, end := document.lineBounds(params.StartLine, params.EndLine, len(lines))
	if end < start {
		return doc.ReadResult{StartLine: start, EndLine: end}, nil
	}

	selected := lines[start-1 : end]

	return doc.ReadResult{
		Content:   document.formatNumbered(selected, start),
		StartLine: start,
		EndLine:   end,
	}, nil
}

/*
Write replaces an entire file or a 1-based inclusive line range.
*/
func (document *Document) Write(_ context.Context, params doc.WriteParams) error {
	abs, err := workspace.Join(document.root, params.Path)
	if err != nil {
		return err
	}

	if params.StartLine == 0 && params.EndLine == 0 {
		return os.WriteFile(abs, []byte(params.Content), 0o644)
	}

	existing, err := os.ReadFile(abs)
	if err != nil {
		return err
	}

	lines := document.splitLines(string(existing))
	start, end := document.lineBounds(params.StartLine, params.EndLine, len(lines))

	replacement := document.splitLines(params.Content)
	merged := append(append(lines[:start-1], replacement...), lines[end:]...)

	return os.WriteFile(abs, []byte(strings.Join(merged, "\n")), 0o644)
}

/*
Replace swaps a unique exact match inside a file.
*/
func (document *Document) Replace(_ context.Context, params doc.ReplaceParams) error {
	abs, err := workspace.Join(document.root, params.Path)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(abs)
	if err != nil {
		return err
	}

	text := string(content)
	count := strings.Count(text, params.Old)

	if count == 0 {
		return fmt.Errorf("replace: %q not found in %q", params.Old, params.Path)
	}

	if count == 1 {
		updated := strings.Replace(text, params.Old, params.New, 1)
		return os.WriteFile(abs, []byte(updated), 0o644)
	}

	return fmt.Errorf("replace: %q occurs %d times in %q; provide more context", params.Old, count, params.Path)
}

/*
Search scans a file for a regular expression and returns numbered matches.
*/
func (document *Document) Search(_ context.Context, params doc.SearchParams) (doc.SearchResult, error) {
	abs, err := workspace.Join(document.root, params.Path)
	if err != nil {
		return doc.SearchResult{}, err
	}

	file, err := os.Open(abs)
	if err != nil {
		return doc.SearchResult{}, err
	}
	defer file.Close()

	pattern, err := regexp.Compile(params.Pattern)
	if err != nil {
		return doc.SearchResult{}, fmt.Errorf("search: invalid pattern: %w", err)
	}

	var matches []doc.ReadResult
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		if !pattern.MatchString(line) {
			continue
		}

		matches = append(matches, doc.ReadResult{
			Content:   document.formatNumbered([]string{line}, lineNumber),
			StartLine: lineNumber,
			EndLine:   lineNumber,
		})
	}

	if err := scanner.Err(); err != nil {
		return doc.SearchResult{}, err
	}

	return doc.SearchResult{Matches: matches}, nil
}

/*
Delete removes a workspace-relative file.
*/
func (document *Document) Delete(_ context.Context, params doc.DeleteParams) error {
	abs, err := workspace.Join(document.root, params.Path)
	if err != nil {
		return err
	}

	return os.Remove(abs)
}

func (document *Document) readLines(file *os.File) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if lines == nil {
		lines = []string{}
	}

	return lines, nil
}

func (document *Document) lineBounds(start, end, total int) (int, int) {
	if total == 0 {
		return 1, 0
	}

	if start <= 0 {
		start = 1
	}

	if end <= 0 || end > total {
		end = total
	}

	if start > end {
		start = end
	}

	return start, end
}

func (document *Document) splitLines(content string) []string {
	if content == "" {
		return []string{}
	}

	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
}

func (document *Document) formatNumbered(lines []string, startLine int) string {
	if len(lines) == 0 {
		return ""
	}

	width := len(fmt.Sprintf("%d", startLine+len(lines)-1))
	parts := make([]string, len(lines))

	for index, line := range lines {
		parts[index] = fmt.Sprintf("%*d| %s", width, startLine+index, line)
	}

	return strings.Join(parts, "\n")
}

/*
Root returns the absolute workspace path.
*/
func (document *Document) Root() string {
	return document.root
}

/*
StatRoot returns workspace metadata for diagnostics.
*/
func (document *Document) StatRoot() (doc.DocumentInfo, error) {
	info, err := os.Stat(document.root)
	if err != nil {
		return doc.DocumentInfo{}, err
	}

	return doc.DocumentInfo{
		Path:    document.root,
		Size:    info.Size(),
		Created: info.ModTime(),
		Updated: time.Now(),
		Mode:    info.Mode().String(),
	}, nil
}
