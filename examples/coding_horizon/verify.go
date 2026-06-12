package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

/*
Verifier runs machine proof commands inside the workspace root.
*/
type Verifier struct {
	root string
}

func newVerifier(root string) *Verifier {
	return &Verifier{root: root}
}

func (verifier *Verifier) GoTest() (string, error) {
	if _, err := os.Stat(filepath.Join(verifier.root, "go.mod")); err != nil {
		return "", fmt.Errorf("coding horizon: workspace has no go.mod: %w", err)
	}

	command := exec.Command("go", "test", "./...")
	command.Dir = verifier.root
	command.Env = append(os.Environ(), "GOFLAGS=-ldflags=-checklinkname=0")

	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output

	runErr := command.Run()
	text := strings.TrimSpace(output.String())

	if runErr != nil {
		return text, fmt.Errorf("coding horizon: go test failed: %w", runErr)
	}

	return text, nil
}

func (verifier *Verifier) GoTestPackage(relativePackage string) (string, error) {
	target := "./" + strings.TrimPrefix(filepath.ToSlash(relativePackage), "./")
	command := exec.Command("go", "test", target)
	command.Dir = verifier.root
	command.Env = append(os.Environ(), "GOFLAGS=-ldflags=-checklinkname=0")

	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output

	runErr := command.Run()
	text := strings.TrimSpace(output.String())

	if runErr != nil {
		return text, fmt.Errorf("coding horizon: go test %s failed: %w", target, runErr)
	}

	return text, nil
}

func packageFromPath(relative string) string {
	dir := filepath.Dir(filepath.ToSlash(relative))
	if dir == "." {
		return "."
	}

	return "./" + dir
}
