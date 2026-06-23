package swarm

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestContentionDetectorDetectDeadlock verifies two actors waiting on each other are detected.
*/
func TestContentionDetectorDetectDeadlock(t *testing.T) {
	Convey("Given two actors waiting on each other's prefixes", t, func() {
		view, err := NewView(time.Minute)
		So(err, ShouldBeNil)

		base := time.Unix(100, 0)
		contention, err := testContention("actor-a", "lanes/b/", "actor-b", base)
		So(err, ShouldBeNil)
		So(view.MergeContention(contention), ShouldBeNil)

		contention, err = testContention("actor-b", "lanes/a/", "actor-a", base.Add(time.Second))
		So(err, ShouldBeNil)
		So(view.MergeContention(contention), ShouldBeNil)

		detector, err := NewContentionDetector(
			context.Background(),
			view,
			ContentionDetectorOptions{
				Window:              10 * time.Second,
				StarvationThreshold: 3,
			},
		)
		So(err, ShouldBeNil)

		Convey("When Detect is called", func() {
			issues, err := detector.Detect(base.Add(2 * time.Second))

			Convey("Then a deadlock issue should be returned", func() {
				So(err, ShouldBeNil)
				So(len(issues), ShouldEqual, 1)
				So(issues[0].Kind, ShouldEqual, ContentionIssueDeadlock)
				So(issues[0].ActorID, ShouldEqual, "actor-a")
				So(issues[0].HolderID, ShouldEqual, "actor-b")
			})
		})
	})
}
