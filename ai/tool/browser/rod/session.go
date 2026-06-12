package rod

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/theapemachine/animal/ai/tool/browser/doc"
)

/*
SessionConfig tunes headless browser timeouts, content limits, and stealth presentation.
Values are applied at launch so every rod Session shares consistent guardrails across MCP tool calls.
*/
type SessionConfig struct {
	Headless      bool
	PageTimeout   time.Duration
	MaxContentLen int
	UserAgent     string
}

var _ doc.Browser = (*Session)(nil)

/*
Session is a stealth headless browser backed by go-rod.
*/
type Session struct {
	config  SessionConfig
	page    *rod.Page
	browser *rod.Browser
}

/*
OpenSession launches a stealth browser page.
*/
func OpenSession(config SessionConfig) (*Session, error) {
	launch := launcher.New().
		Headless(config.Headless).
		Set("disable-blink-features", "AutomationControlled")

	controlURL, err := launch.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connect browser: %w", err)
	}

	page, err := stealth.Page(browser)
	if err != nil {
		_ = browser.Close()
		return nil, fmt.Errorf("stealth page: %w", err)
	}

	if config.PageTimeout <= 0 {
		config.PageTimeout = 45 * time.Second
	}

	if config.MaxContentLen <= 0 {
		config.MaxContentLen = 120_000
	}

	page = page.Timeout(config.PageTimeout)

	if strings.TrimSpace(config.UserAgent) != "" {
		if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: config.UserAgent,
		}); err != nil {
			_ = browser.Close()
			return nil, fmt.Errorf("set user agent: %w", err)
		}
	}

	return &Session{
		config:  config,
		page:    page,
		browser: browser,
	}, nil
}

func (session *Session) pageWithContext(ctx context.Context) *rod.Page {
	return session.page.Context(ctx)
}

/*
Navigate loads a URL and returns the resolved location and title.
*/
func (session *Session) Navigate(
	ctx context.Context,
	params doc.NavigateParams,
) (doc.NavigateResult, error) {
	targetURL := strings.TrimSpace(params.URL)
	if targetURL == "" {
		return doc.NavigateResult{}, fmt.Errorf("browser: url is required")
	}

	page := session.pageWithContext(ctx)

	if err := page.Navigate(targetURL); err != nil {
		return doc.NavigateResult{}, fmt.Errorf("navigate: %w", err)
	}

	if err := page.WaitLoad(); err != nil {
		return doc.NavigateResult{}, fmt.Errorf("wait load: %w", err)
	}

	info, err := page.Info()
	if err != nil {
		return doc.NavigateResult{}, fmt.Errorf("page info: %w", err)
	}

	return doc.NavigateResult{
		URL:   info.URL,
		Title: info.Title,
	}, nil
}

/*
Evaluate runs JavaScript in the page and returns JSON-encoded output.
*/
func (session *Session) Evaluate(
	ctx context.Context,
	params doc.EvaluateParams,
) (doc.EvaluateResult, error) {
	expression := strings.TrimSpace(params.Expression)
	if expression == "" {
		return doc.EvaluateResult{}, fmt.Errorf("browser: expression is required")
	}

	value, err := session.pageWithContext(ctx).Eval(expression)
	if err != nil {
		return doc.EvaluateResult{}, fmt.Errorf("evaluate: %w", err)
	}

	encoded, err := json.Marshal(value.Value)
	if err != nil {
		return doc.EvaluateResult{}, fmt.Errorf("encode result: %w", err)
	}

	return doc.EvaluateResult{Result: string(encoded)}, nil
}

/*
Content returns visible text or HTML from the current page.
*/
func (session *Session) Content(
	ctx context.Context,
	params doc.ContentParams,
) (doc.ContentResult, error) {
	page := session.pageWithContext(ctx)

	info, err := page.Info()
	if err != nil {
		return doc.ContentResult{}, fmt.Errorf("page info: %w", err)
	}

	format := strings.ToLower(strings.TrimSpace(params.Format))
	if format == "" {
		format = "text"
	}

	var content string

	switch format {
	case "html":
		content, err = page.HTML()
	default:
		value, evalErr := page.Eval(`() => document.body?.innerText ?? ""`)
		if evalErr != nil {
			return doc.ContentResult{}, fmt.Errorf("text content: %w", evalErr)
		}

		content = value.Value.String()
		err = nil
	}

	if err != nil {
		return doc.ContentResult{}, fmt.Errorf("content: %w", err)
	}

	truncated := false
	if len(content) > session.config.MaxContentLen {
		content = content[:session.config.MaxContentLen]
		truncated = true
	}

	return doc.ContentResult{
		URL:       info.URL,
		Title:     info.Title,
		Content:   content,
		Truncated: truncated,
	}, nil
}

/*
Click activates the element matched by a CSS selector.
*/
func (session *Session) Click(
	ctx context.Context,
	params doc.ClickParams,
) (doc.ClickResult, error) {
	selector := strings.TrimSpace(params.Selector)
	if selector == "" {
		return doc.ClickResult{}, fmt.Errorf("browser: selector is required")
	}

	page := session.pageWithContext(ctx)
	element, err := page.Element(selector)
	if err != nil {
		return doc.ClickResult{}, fmt.Errorf("find selector: %w", err)
	}

	if err := element.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return doc.ClickResult{}, fmt.Errorf("click: %w", err)
	}

	if err := page.WaitLoad(); err != nil {
		return doc.ClickResult{}, fmt.Errorf("wait load: %w", err)
	}

	info, err := page.Info()
	if err != nil {
		return doc.ClickResult{}, fmt.Errorf("page info: %w", err)
	}

	return doc.ClickResult{URL: info.URL}, nil
}

/*
Wait blocks until a selector appears or the page finishes loading.
*/
func (session *Session) Wait(
	ctx context.Context,
	params doc.WaitParams,
) (doc.WaitResult, error) {
	page := session.pageWithContext(ctx)

	if strings.TrimSpace(params.Selector) != "" {
		timeout := time.Duration(params.TimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = session.config.PageTimeout
		}

		if _, err := page.Timeout(timeout).Element(params.Selector); err != nil {
			return doc.WaitResult{}, fmt.Errorf("wait selector: %w", err)
		}
	} else if err := page.WaitLoad(); err != nil {
		return doc.WaitResult{}, fmt.Errorf("wait load: %w", err)
	}

	info, err := page.Info()
	if err != nil {
		return doc.WaitResult{}, fmt.Errorf("page info: %w", err)
	}

	return doc.WaitResult{URL: info.URL}, nil
}

/*
Close shuts down the browser.
*/
func (session *Session) Close() error {
	if session.browser == nil {
		return nil
	}

	return session.browser.Close()
}
