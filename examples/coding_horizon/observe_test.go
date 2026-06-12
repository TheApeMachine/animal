package main

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestObserverDigest verifies deterministic workspace scanning.
*/
func TestObserverDigest(t *testing.T) {
	Convey("Given a tiny Go module workspace", t, func() {
		root := t.TempDir()
		So(os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.22\n"), 0o644), ShouldBeNil)
		So(os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644), ShouldBeNil)

		observer := newObserver(root)

		Convey("When Digest is called", func() {
			digest, err := observer.Digest()

			Convey("Then it should summarize the module", func() {
				So(err, ShouldBeNil)
				So(digest.Module, ShouldEqual, "example.com/demo")
				So(digest.GoFiles, ShouldEqual, 1)
				So(len(digest.Untested), ShouldEqual, 1)
			})
		})
	})
}

/*
TestVerifierGoTest verifies machine proof in a workspace.
*/
func TestVerifierGoTest(t *testing.T) {
	Convey("Given a tiny passing Go test workspace", t, func() {
		root := t.TempDir()
		So(os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n\ngo 1.22\n"), 0o644), ShouldBeNil)
		So(os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc Value() int { return 1 }\n"), 0o644), ShouldBeNil)
		So(os.WriteFile(filepath.Join(root, "main_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) {\n\tif Value() != 1 {\n\t\tt.Fatal(\"expected 1\")\n\t}\n}\n"), 0o644), ShouldBeNil)

		verifier := newVerifier(root)

		Convey("When GoTest runs", func() {
			output, err := verifier.GoTest()

			Convey("Then tests should pass", func() {
				So(err, ShouldBeNil)
				So(output, ShouldContainSubstring, "ok")
			})
		})
	})
}
