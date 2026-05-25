package editor

import (
	"context"
	"time"
)

type Document interface {
	Info() DocumentInfo
	Read(context.Context, ReadParams) (string, error)
	Write(context.Context, WriteParams) error
	Replace(context.Context, ReplaceParams) error
	Search(context.Context, SearchParams) (SearchResult, error)
	Delete(context.Context, DeleteParams) error
}

type DocumentInfo struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Owner   string    `json:"owner"`
	Group   string    `json:"group"`
	Mode    string    `json:"mode"`
}

type ReadParams struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type ReadResult struct {
	Content   string `json:"content"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type WriteParams struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type WriteResult struct {
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`
}

type ReplaceParams struct {
	Path string `json:"path"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

type SearchParams struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern"`
}

type SearchResult struct {
	Matches []ReadResult `json:"matches"`
}

type DeleteParams struct {
	Path string `json:"path"`
}
