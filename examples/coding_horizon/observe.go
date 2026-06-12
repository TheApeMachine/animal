package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxFileLines     = 400
	maxDigestEntries = 48
)

/*
RepoDigest is a machine-built map of the workspace used to ground every LLM phase.
*/
type RepoDigest struct {
	Root        string   `json:"root"`
	Module      string   `json:"module"`
	GoFiles     int      `json:"go_files"`
	TestFiles   int      `json:"test_files"`
	LargeFiles  []string `json:"large_files"`
	Untested    []string `json:"untested"`
	LockSignals []string `json:"lock_signals"`
	TopLevel    []string `json:"top_level"`
	SamplePaths []string `json:"sample_paths"`
}

/*
Observer walks the workspace and extracts deterministic signals for planning.
*/
type Observer struct {
	root string
}

func newObserver(root string) *Observer {
	return &Observer{root: root}
}

func (observer *Observer) Digest() (*RepoDigest, error) {
	digest := &RepoDigest{
		Root:        observer.root,
		LargeFiles:  make([]string, 0),
		Untested:    make([]string, 0),
		LockSignals: make([]string, 0),
		TopLevel:    make([]string, 0),
		SamplePaths: make([]string, 0),
	}

	module, moduleErr := observer.readModule()
	if moduleErr != nil {
		return nil, moduleErr
	}

	digest.Module = module

	entries, readErr := os.ReadDir(observer.root)
	if readErr != nil {
		return nil, fmt.Errorf("coding horizon: read workspace root: %w", readErr)
	}

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			digest.TopLevel = append(digest.TopLevel, entry.Name()+"/")
		}
	}

	testedSources := make(map[string]struct{})

	walkErr := filepath.WalkDir(observer.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() && observer.skipDir(entry.Name()) {
			return filepath.SkipDir
		}

		if entry.IsDir() {
			return nil
		}

		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}

		relative, relErr := filepath.Rel(observer.root, path)
		if relErr != nil {
			return relErr
		}

		relative = filepath.ToSlash(relative)

		if strings.HasSuffix(relative, "_test.go") {
			digest.TestFiles++

			return nil
		}

		digest.GoFiles++

		if len(digest.SamplePaths) < maxDigestEntries {
			digest.SamplePaths = append(digest.SamplePaths, relative)
		}

		lineCount, countErr := observer.countLines(path)
		if countErr != nil {
			return countErr
		}

		if lineCount > maxFileLines {
			digest.LargeFiles = append(digest.LargeFiles, fmt.Sprintf("%s (%d lines)", relative, lineCount))
		}

		testedSources[relative] = struct{}{}

		lockHits, scanErr := observer.scanLockSignals(path, relative)
		if scanErr != nil {
			return scanErr
		}

		digest.LockSignals = append(digest.LockSignals, lockHits...)

		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("coding horizon: walk workspace: %w", walkErr)
	}

	for source := range testedSources {
		testPath := strings.TrimSuffix(source, ".go") + "_test.go"
		fullTestPath := filepath.Join(observer.root, filepath.FromSlash(testPath))

		if _, statErr := os.Stat(fullTestPath); statErr != nil {
			digest.Untested = append(digest.Untested, source)
		}
	}

	return digest, nil
}

func (observer *Observer) skipDir(name string) bool {
	switch name {
	case ".git", "vendor", "node_modules", "dist", "build", ".cursor":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}

func (observer *Observer) readModule() (string, error) {
	modPath := filepath.Join(observer.root, "go.mod")
	file, err := os.Open(modPath)
	if err != nil {
		return "", nil
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", scanner.Err()
}

func (observer *Observer) countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}

	defer file.Close()

	lineCount := 0
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lineCount++
	}

	return lineCount, scanner.Err()
}

func (observer *Observer) scanLockSignals(path, relative string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	signals := make([]string, 0)
	text := string(content)

	for _, pattern := range []string{"sync.Mutex", "sync.RWMutex", "make(chan", " chan "} {
		if strings.Contains(text, pattern) {
			signals = append(signals, fmt.Sprintf("%s contains %q", relative, pattern))
		}
	}

	return signals, nil
}

func hygieneTasksFromDigest(digest *RepoDigest) []Task {
	tasks := make([]Task, 0)
	index := 0

	nextID := func(prefix string) string {
		index++
		return fmt.Sprintf("%s-%d", prefix, index)
	}

	for _, entry := range digest.LargeFiles {
		tasks = append(tasks, Task{
			ID:          nextID("split"),
			Kind:        taskKindHygiene,
			Title:       fmt.Sprintf("Split oversized file %s", entry),
			Rationale:   "AGENTS.md hard ceiling is 400 lines; large files degrade agent reasoning accuracy.",
			TargetFiles: []string{strings.Fields(entry)[0]},
			Acceptance:  "File is decomposed into focused units without behavior change and tests still pass.",
		})
	}

	for _, source := range digest.Untested {
		tasks = append(tasks, Task{
			ID:          nextID("test"),
			Kind:        taskKindHygiene,
			Title:       fmt.Sprintf("Add meaningful tests for %s", source),
			Rationale:   "Untested production code lacks machine-verifiable proof for future agent edits.",
			TargetFiles: []string{source},
			Acceptance:  "A _test.go mirror exists and exercises real behavior, not trivial assertions.",
		})
	}

	for _, signal := range digest.LockSignals {
		tasks = append(tasks, Task{
			ID:          nextID("lockfree"),
			Kind:        taskKindHygiene,
			Title:       fmt.Sprintf("Remove lock/channel usage: %s", signal),
			Rationale:   "This project targets qpool/disruptor lock-free coordination.",
			TargetFiles: []string{strings.Fields(signal)[0]},
			Acceptance:  "Mutex, RWMutex, and channel-based coordination are replaced with approved primitives.",
		})
	}

	if len(tasks) == 0 {
		tasks = append(tasks, Task{
			ID:          nextID("audit"),
			Kind:        taskKindHygiene,
			Title:       "Repository hygiene audit",
			Rationale:   "No static debt signals were found; perform a focused readability and dead-code pass.",
			TargetFiles: digest.SamplePaths,
			Acceptance:  "At least one concrete improvement lands with passing tests, or task is marked blocked with evidence.",
		})
	}

	return tasks
}

func digestBrief(digest *RepoDigest) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("module=%s go_files=%d test_files=%d\n", digest.Module, digest.GoFiles, digest.TestFiles))
	builder.WriteString(fmt.Sprintf("top_level=%s\n", strings.Join(digest.TopLevel, ", ")))

	if len(digest.LargeFiles) > 0 {
		builder.WriteString(fmt.Sprintf("large_files=%s\n", strings.Join(digest.LargeFiles, "; ")))
	}

	if len(digest.Untested) > 0 {
		builder.WriteString(fmt.Sprintf("untested=%s\n", strings.Join(digest.Untested, ", ")))
	}

	if len(digest.LockSignals) > 0 {
		builder.WriteString(fmt.Sprintf("lock_signals=%s\n", strings.Join(digest.LockSignals, "; ")))
	}

	builder.WriteString(fmt.Sprintf("sample_paths=%s\n", strings.Join(digest.SamplePaths, ", ")))

	return builder.String()
}
