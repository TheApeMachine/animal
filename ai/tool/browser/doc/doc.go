package doc

import "context"

/*
Browser exposes page navigation and JavaScript-driven extraction.
*/
type Browser interface {
	Navigate(ctx context.Context, params NavigateParams) (NavigateResult, error)
	Evaluate(ctx context.Context, params EvaluateParams) (EvaluateResult, error)
	Content(ctx context.Context, params ContentParams) (ContentResult, error)
	Click(ctx context.Context, params ClickParams) (ClickResult, error)
	Wait(ctx context.Context, params WaitParams) (WaitResult, error)
	Close() error
}

type NavigateParams struct {
	URL string `json:"url" jsonschema:"required,description=Absolute URL to load"`
}

type NavigateResult struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type EvaluateParams struct {
	Expression string `json:"expression" jsonschema:"required,description=JavaScript expression evaluated in the page context; return value is JSON-serialized"`
}

type EvaluateResult struct {
	Result string `json:"result"`
}

type ContentParams struct {
	Format string `json:"format,omitempty" jsonschema:"description=One of text or html; defaults to text"`
}

type ContentResult struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated,omitempty"`
}

type ClickParams struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector to click"`
}

type ClickResult struct {
	URL string `json:"url"`
}

type WaitParams struct {
	Selector  string `json:"selector,omitempty" jsonschema:"description=CSS selector to wait for; omit to wait for load only"`
	TimeoutMs int    `json:"timeout_ms,omitempty" jsonschema:"description=Optional timeout in milliseconds"`
}

type WaitResult struct {
	URL string `json:"url"`
}
