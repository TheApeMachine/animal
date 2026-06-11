package browser

import "time"

const (
	defaultPageTimeout   = 45 * time.Second
	defaultMaxContentLen = 120_000
)

/*
Config controls the stealth headless browser session.
*/
type Config struct {
	Headless      bool
	PageTimeout   time.Duration
	MaxContentLen int
	UserAgent     string
}

/*
DefaultConfig returns browser defaults suited for autonomous enrichment.
*/
func DefaultConfig() Config {
	return Config{
		Headless:      true,
		PageTimeout:   defaultPageTimeout,
		MaxContentLen: defaultMaxContentLen,
	}
}
