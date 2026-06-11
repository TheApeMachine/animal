package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/ai/tool/editor/doc"
	"github.com/theapemachine/animal/lease"
)

func testLeaseOptions() lease.Options {
	return lease.Options{
		KeySpace: lease.PathKeySpace{},
		IdleTTL:  15 * time.Minute,
	}
}

func newTestDocument(t *testing.T) (*Document, string) {
	t.Helper()

	root := t.TempDir()
	coordinator, err := lease.NewCoordinator(testLeaseOptions())
	if err != nil {
		t.Fatal(err)
	}

	document, err := NewDocument(root, coordinator)
	if err != nil {
		t.Fatal(err)
	}

	return document, root
}

/*
TestNewDocument verifies filesystem document construction.
*/
func TestNewDocument(t *testing.T) {
	Convey("Given a nil lease coordinator", t, func() {
		Convey("When NewDocument is called", func() {
			document, err := NewDocument(t.TempDir(), nil)

			Convey("Then it should reject the missing coordinator", func() {
				So(document, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})
	})
}

/*
TestDocumentRead verifies numbered line-range reads.
*/
func TestDocumentRead(t *testing.T) {
	Convey("Given a file with three lines", t, func() {
		document, root := newTestDocument(t)
		path := "pkg/main.go"
		abs := filepath.Join(root, path)

		So(os.MkdirAll(filepath.Dir(abs), 0o755), ShouldBeNil)
		So(os.WriteFile(abs, []byte("alpha\nbeta\ngamma\n"), 0o644), ShouldBeNil)

		Convey("When Read selects line 2", func() {
			result, err := document.Read(context.Background(), doc.ReadParams{
				Path:      path,
				StartLine: 2,
				EndLine:   2,
			})

			Convey("Then it should return numbered content for that line", func() {
				So(err, ShouldBeNil)
				So(result.StartLine, ShouldEqual, 2)
				So(result.EndLine, ShouldEqual, 2)
				So(result.Content, ShouldContainSubstring, "2| beta")
			})
		})
	})
}

/*
TestDocumentSearch verifies regular-expression search.
*/
func TestDocumentSearch(t *testing.T) {
	Convey("Given a file with matching content", t, func() {
		document, root := newTestDocument(t)
		path := "main.go"
		abs := filepath.Join(root, path)

		So(os.WriteFile(abs, []byte("foo\nbar baz\n"), 0o644), ShouldBeNil)

		Convey("When Search matches bar", func() {
			result, err := document.Search(context.Background(), doc.SearchParams{
				Path:    path,
				Pattern: `ba.`,
			})

			Convey("Then it should return one numbered match", func() {
				So(err, ShouldBeNil)
				So(len(result.Matches), ShouldEqual, 1)
			})
		})
	})
}

/*
TestDocumentReplace verifies unique exact replacements.
*/
func TestDocumentReplace(t *testing.T) {
	Convey("Given a file with a unique match", t, func() {
		document, root := newTestDocument(t)
		path := "main.go"
		abs := filepath.Join(root, path)

		So(os.WriteFile(abs, []byte("hello world\n"), 0o644), ShouldBeNil)

		Convey("When Replace swaps world for there", func() {
			replaceErr := document.Replace(context.Background(), doc.ReplaceParams{
				Path: path,
				Old:  "world",
				New:  "there",
			})

			Convey("Then the file should be updated", func() {
				So(replaceErr, ShouldBeNil)

				content, readErr := os.ReadFile(abs)
				So(readErr, ShouldBeNil)
				So(string(content), ShouldEqual, "hello there\n")
			})
		})
	})

	Convey("Given a file with ambiguous matches", t, func() {
		document, root := newTestDocument(t)
		path := "main.go"
		abs := filepath.Join(root, path)

		So(os.WriteFile(abs, []byte("foo foo\n"), 0o644), ShouldBeNil)

		Convey("When Replace targets foo", func() {
			replaceErr := document.Replace(context.Background(), doc.ReplaceParams{
				Path: path,
				Old:  "foo",
				New:  "bar",
			})

			Convey("Then it should reject the ambiguous replacement", func() {
				So(replaceErr, ShouldNotBeNil)
				So(replaceErr.Error(), ShouldContainSubstring, "occurs")
			})
		})
	})
}
