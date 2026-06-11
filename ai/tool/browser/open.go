package browser

import (
	"fmt"

	browserrod "github.com/theapemachine/animal/ai/tool/browser/rod"
)

/*
Open launches a stealth browser session from package config.
*/
func Open(config Config) (*browserrod.Session, error) {
	session, err := browserrod.OpenSession(browserrod.SessionConfig{
		Headless:      config.Headless,
		PageTimeout:   config.PageTimeout,
		MaxContentLen: config.MaxContentLen,
		UserAgent:     config.UserAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("browser: %w", err)
	}

	return session, nil
}
