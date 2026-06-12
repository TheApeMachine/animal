package editor

import "github.com/theapemachine/animal/ai/tool/editor/doc"

/*
Document re-exports the workspace editing contract for callers outside the doc package.
*/
type Document = doc.Document

/*
DocumentInfo re-exports on-disk file metadata returned by workspace listing and stat operations.
*/
type DocumentInfo = doc.DocumentInfo

/*
ReadParams re-exports the file and line-range selector for workspace reads.
*/
type ReadParams = doc.ReadParams

/*
ReadResult re-exports numbered file content with resolved line bounds.
*/
type ReadResult = doc.ReadResult

/*
WriteParams re-exports whole-file or line-range write instructions.
*/
type WriteParams = doc.WriteParams

/*
WriteResult re-exports the line bounds affected by a workspace write.
*/
type WriteResult = doc.WriteResult

/*
ReplaceParams re-exports exact-match replacement instructions inside one file.
*/
type ReplaceParams = doc.ReplaceParams

/*
ReplaceResult re-exports confirmation of which file was updated by a replace.
*/
type ReplaceResult = doc.ReplaceResult

/*
SearchParams re-exports regular-expression scan settings for one workspace file.
*/
type SearchParams = doc.SearchParams

/*
SearchResult re-exports numbered lines that matched a search pattern.
*/
type SearchResult = doc.SearchResult

/*
DeleteParams re-exports the workspace file path targeted for removal.
*/
type DeleteParams = doc.DeleteParams

/*
FileChangingNotice re-exports the advisory signal that a file may still be mutating externally.
*/
type FileChangingNotice = doc.FileChangingNotice
