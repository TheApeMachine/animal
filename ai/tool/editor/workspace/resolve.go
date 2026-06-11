package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	envAnimalWorkspace   = "ANIMAL_AGENT_WORKSPACE"
	envAlcatrazWorkspace = "ALCATRAZ_AGENT_WORKSPACE"
)

/*
Resolve picks the agent workspace root directory.

Order: ANIMAL_AGENT_WORKSPACE, ALCATRAZ_AGENT_WORKSPACE, ./alcatraz,
../alcatraz, then the current working directory.
*/
func Resolve() (string, error) {
	for _, key := range []string{envAnimalWorkspace, envAlcatrazWorkspace} {
		if value := os.Getenv(key); value != "" {
			return absDir(value)
		}
	}

	for _, candidate := range []string{"alcatraz", filepath.Join("..", "alcatraz")} {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return absDir(candidate)
		}
	}

	return absDir(".")
}

func absDir(path string) (string, error) {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("workspace: resolve %q: %w", path, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("workspace: stat %q: %w", abs, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("workspace: %q is not a directory", abs)
	}

	return abs, nil
}

/*
Join resolves path relative to root and rejects escapes outside root.
*/
func Join(root, path string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("workspace: root is required")
	}

	if path == "" {
		return "", fmt.Errorf("workspace: path is required")
	}

	rootAbs, err := absDir(root)
	if err != nil {
		return "", err
	}

	clean := filepath.Clean(filepath.FromSlash(path))
	if filepath.IsAbs(clean) {
		clean = clean[1:]
	}

	joined := filepath.Join(rootAbs, clean)
	rel, err := filepath.Rel(rootAbs, joined)
	if err != nil {
		return "", fmt.Errorf("workspace: resolve %q: %w", path, err)
	}

	if rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		return "", fmt.Errorf("workspace: path %q escapes workspace root", path)
	}

	return joined, nil
}
