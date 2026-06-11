package workspace

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestJoin verifies workspace-relative path joining.
*/
func TestJoin(t *testing.T) {
	Convey("Given a workspace root", t, func() {
		root := t.TempDir()

		Convey("When Join resolves an escaping path", func() {
			_, err := Join(root, "../outside.txt")

			Convey("Then it should reject the escape", func() {
				So(err, ShouldNotBeNil)
			})
		})

		Convey("When Join resolves a nested relative path", func() {
			path := filepath.Join("nested", "file.txt")
			abs := filepath.Join(root, path)

			So(os.MkdirAll(filepath.Dir(abs), 0o755), ShouldBeNil)
			So(os.WriteFile(abs, []byte("ok"), 0o644), ShouldBeNil)

			joined, err := Join(root, path)

			Convey("Then it should return the absolute workspace path", func() {
				So(err, ShouldBeNil)
				So(joined, ShouldEqual, abs)
			})
		})
	})
}

/*
TestResolve verifies workspace root resolution.
*/
func TestResolve(t *testing.T) {
	Convey("Given ANIMAL_AGENT_WORKSPACE is set", t, func() {
		root := t.TempDir()
		t.Setenv("ANIMAL_AGENT_WORKSPACE", root)

		Convey("When Resolve is called", func() {
			resolved, err := Resolve()

			Convey("Then it should return the configured workspace", func() {
				So(err, ShouldBeNil)
				So(resolved, ShouldEqual, root)
			})
		})
	})
}
