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

/*
NavigateParams identifies the absolute URL a browser session should load next.
*/
type NavigateParams struct {
	URL string `json:"url" jsonschema:"required,description=Absolute URL to load"`
}

/*
NavigateResult reports the resolved location and document title after navigation completes.
*/
type NavigateResult struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

/*
EvaluateParams carries JavaScript evaluated in the active page context.
The expression's return value is JSON-serialized for transport back to the agent.
*/
type EvaluateParams struct {
	Expression string `json:"expression" jsonschema:"required,description=JavaScript expression evaluated in the page context; return value is JSON-serialized"`
}

/*
EvaluateResult holds the JSON-encoded output of a page JavaScript evaluation.
*/
type EvaluateResult struct {
	Result string `json:"result"`
}

/*
ContentParams selects whether page extraction returns plain text or HTML.
Omitting format defaults to text so agents receive readable excerpts without markup noise.
*/
type ContentParams struct {
	Format string `json:"format,omitempty" jsonschema:"description=One of text or html; defaults to text"`
}

/*
ContentResult returns extracted page body text or HTML together with location metadata.
Truncated is set when output exceeds the session MaxContentLen to keep MCP payloads bounded.
*/
type ContentResult struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated,omitempty"`
}

/*
ClickParams names the CSS selector whose first matching element should be activated.
*/
type ClickParams struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector to click"`
}

/*
ClickResult reports the page URL after the click and any subsequent navigation settle.
*/
type ClickResult struct {
	URL string `json:"url"`
}

/*
WaitParams blocks until a selector appears or, when omitted, until the page finishes loading.
TimeoutMs caps selector waits so runaway pages cannot stall the agent loop indefinitely.
*/
type WaitParams struct {
	Selector  string `json:"selector,omitempty" jsonschema:"description=CSS selector to wait for; omit to wait for load only"`
	TimeoutMs int    `json:"timeout_ms,omitempty" jsonschema:"description=Optional timeout in milliseconds"`
}

/*
WaitResult reports the page URL once the wait condition succeeds.
*/
type WaitResult struct {
	URL string `json:"url"`
}
