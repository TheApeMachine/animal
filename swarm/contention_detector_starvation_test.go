package swarm

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestContentionDetectorDetectStarvation verifies repeated failed claims are detected.
*/
func TestContentionDetectorDetectStarvation(t *testing.T) {
	Convey("Given repeated contention for one actor and prefix", t, func() {
		view, err := NewView(time.Minute)
		So(err, ShouldBeNil)

		base := time.Unix(100, 0)
		contention, err := testContention("actor-a", "lanes/b/", "actor-b", base)
		So(err, ShouldBeNil)
		So(view.MergeContention(contention), ShouldBeNil)

		contention, err = testContention("actor-a", "lanes/b/", "actor-b", base.Add(time.Second))
		So(err, ShouldBeNil)
		So(view.MergeContention(contention), ShouldBeNil)

		contention, err = testContention("actor-a", "lanes/b/", "actor-b", base.Add(2*time.Second))
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
			issues, err := detector.Detect(base.Add(3 * time.Second))

			Convey("Then a starvation issue should be returned", func() {
				So(err, ShouldBeNil)
				So(len(issues), ShouldEqual, 1)
				So(issues[0].Kind, ShouldEqual, ContentionIssueStarvation)
				So(issues[0].Attempts, ShouldEqual, 3)
			})
		})
	})
}
