package swarm

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestViewMerge verifies gossip merge and prefix availability queries.
*/
func TestViewMerge(t *testing.T) {
	Convey("Given a fresh view", t, func() {
		view, err := NewView(30 * time.Second)
		So(err, ShouldBeNil)

		claim := NewRumorAt(KindClaim, "actor-a", "Ada", "developer", time.Now())
		claim.Prefix = "lanes/a/"

		Convey("When a claim rumor is merged", func() {
			mergeErr := view.Merge(claim)
			So(mergeErr, ShouldBeNil)

			Convey("Then the prefix should no longer be free", func() {
				So(view.IsPrefixFree("lanes/a/"), ShouldBeFalse)
				holder, ok := view.ClaimHolder("lanes/a/")
				So(ok, ShouldBeTrue)
				So(holder, ShouldEqual, "actor-a")
			})
		})

		Convey("When a release rumor arrives after a claim", func() {
			So(view.Merge(claim), ShouldBeNil)

			release := NewRumorAt(KindRelease, "actor-a", "Ada", "developer", time.Now().Add(time.Millisecond))
			release.Prefix = "lanes/a/"

			mergeErr := view.Merge(release)

			Convey("Then the prefix should be free again", func() {
				So(mergeErr, ShouldBeNil)
				So(view.IsPrefixFree("lanes/a/"), ShouldBeTrue)
			})
		})
	})
}

/*
TestViewPurgeExpired verifies gossip entries expire after the TTL window.
*/
func TestViewPurgeExpired(t *testing.T) {
	Convey("Given a view with a short gossip TTL", t, func() {
		view, err := NewView(10 * time.Millisecond)
		So(err, ShouldBeNil)

		stamp := time.Now().Add(-20 * time.Millisecond)
		claim := NewRumorAt(KindClaim, "actor-a", "Ada", "developer", stamp)
		claim.Prefix = "lanes/a/"
		So(view.Merge(claim), ShouldBeNil)

		Convey("When PurgeExpired runs", func() {
			view.PurgeExpired(time.Now())

			Convey("Then stale claims should be removed", func() {
				So(view.IsPrefixFree("lanes/a/"), ShouldBeTrue)
			})
		})
	})
}
