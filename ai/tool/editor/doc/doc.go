package doc

import (
	"context"
	"time"
)

/*
Document is the virtual workspace editing contract shared by MCP tools and backends.
*/
type Document interface {
	Info(context.Context, string) (DocumentInfo, error)
	Read(context.Context, ReadParams) (ReadResult, error)
	Write(context.Context, WriteParams) error
	Replace(context.Context, ReplaceParams) error
	Search(context.Context, SearchParams) (SearchResult, error)
	Delete(context.Context, DeleteParams) error
}

/*
DocumentInfo describes a workspace file on disk.
*/
type DocumentInfo struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Owner   string    `json:"owner"`
	Group   string    `json:"group"`
	Mode    string    `json:"mode"`
}

/*
ReadParams selects a workspace file and optional 1-based line range.
*/
type ReadParams struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

/*
ReadResult returns numbered file content and the resolved line bounds.
*/
type ReadResult struct {
	Content   string              `json:"content"`
	StartLine int                 `json:"start_line"`
	EndLine   int                 `json:"end_line"`
	Changing  *FileChangingNotice `json:"changing,omitempty"`
}

/*
FileChangingNotice tells a caller that another agent is actively changing path.
It is advisory: the tool call succeeds so the caller can wait or proceed without this file.
*/
type FileChangingNotice struct {
	Path        string `json:"path"`
	LeasePrefix string `json:"lease_prefix"`
	AgentID     string `json:"agent_id"`
	Message     string `json:"message"`
}

/*
WriteParams replaces an entire file or an inclusive line range.
*/
type WriteParams struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

/*
WriteResult reports the line bounds affected by a write.
*/
type WriteResult struct {
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`
}

/*
ReplaceParams identifies a unique exact replacement inside a file.
*/
type ReplaceParams struct {
	Path string `json:"path"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

/*
SearchParams scans one file with a regular expression.
*/
type SearchParams struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern"`
}

/*
SearchResult lists numbered lines that matched the pattern.
*/
type SearchResult struct {
	Matches  []ReadResult        `json:"matches"`
	Changing *FileChangingNotice `json:"changing,omitempty"`
}

/*
DeleteParams identifies a workspace file to remove.
*/
type DeleteParams struct {
	Path string `json:"path"`
}

/*
ReplaceResult confirms which file was updated.
*/
type ReplaceResult struct {
	Path string `json:"path"`
}
