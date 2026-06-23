package swarm

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestViewMergeContention verifies contention events are stored in order.
*/
func TestViewMergeContention(t *testing.T) {
	Convey("Given a view with contention events", t, func() {
		view, err := NewView(30 * time.Second)
		So(err, ShouldBeNil)

		second, err := testContention("actor-b", "lanes/b/", "actor-a", time.Unix(200, 0))
		So(err, ShouldBeNil)

		first, err := testContention("actor-a", "lanes/a/", "actor-b", time.Unix(100, 0))
		So(err, ShouldBeNil)

		So(view.MergeContention(second), ShouldBeNil)
		So(view.MergeContention(first), ShouldBeNil)

		Convey("When RecentContentions is called", func() {
			contentions := view.RecentContentions()

			Convey("Then contention events should be sorted by time", func() {
				So(len(contentions), ShouldEqual, 2)
				So(contentions[0].ActorID, ShouldEqual, "actor-a")
				So(contentions[1].ActorID, ShouldEqual, "actor-b")
			})
		})
	})
}

func testContention(
	actorID string,
	prefix string,
	holderID string,
	at time.Time,
) (Contention, error) {
	contention, err := NewContentionAt(
		actorID,
		actorID,
		"developer",
		prefix,
		"lease held",
		at,
	)

	if err != nil {
		return Contention{}, err
	}

	contention.HolderID = holderID
	contention.HolderPrefix = prefix

	return contention, nil
}
