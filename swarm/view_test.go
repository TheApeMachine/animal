package swarm

import (
	"sync"
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
TestViewMergeConcurrent verifies concurrent merges converge without data races.
*/
func TestViewMergeConcurrent(t *testing.T) {
	view, err := NewView(30 * time.Second)
	if err != nil {
		t.Fatalf("new view: %v", err)
	}

	claimA := NewRumorAt(KindClaim, "actor-a", "Ada", "developer", time.Now())
	claimA.Prefix = "lanes/a/"
	claimB := NewRumorAt(KindClaim, "actor-b", "Ben", "developer", time.Now())
	claimB.Prefix = "lanes/b/"

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()

		if mergeErr := view.Merge(claimA); mergeErr != nil {
			t.Errorf("merge claim a: %v", mergeErr)
		}
	}()

	go func() {
		defer waitGroup.Done()

		if mergeErr := view.Merge(claimB); mergeErr != nil {
			t.Errorf("merge claim b: %v", mergeErr)
		}
	}()

	waitGroup.Wait()

	holderA, okA := view.ClaimHolder("lanes/a/")
	if !okA || holderA != "actor-a" {
		t.Fatalf("claim holder a = %q, ok = %v", holderA, okA)
	}

	holderB, okB := view.ClaimHolder("lanes/b/")
	if !okB || holderB != "actor-b" {
		t.Fatalf("claim holder b = %q, ok = %v", holderB, okB)
	}
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

/*
TestViewPurgeExpiredSignalsAndMetrics verifies event entries expire after the TTL window.
*/
func TestViewPurgeExpiredSignalsAndMetrics(t *testing.T) {
	Convey("Given a view with stale signal and metric entries", t, func() {
		view, err := NewView(10 * time.Millisecond)
		So(err, ShouldBeNil)

		stamp := time.Now().Add(-20 * time.Millisecond)

		signal := NewSignalAt(SignalFriction, "actor-a", "Ada", "developer", stamp)
		signal.Summary = "context drift"
		So(view.MergeSignal(signal), ShouldBeNil)

		metric := NewMetricAt("actor-a", "Ada", "developer", stamp)
		metric.Name = "tests_passed"
		metric.Score = 1
		metric.Success = true
		So(view.MergeMetric(metric), ShouldBeNil)

		Convey("When PurgeExpired runs", func() {
			view.PurgeExpired(time.Now())

			Convey("Then stale signals and metrics should be removed", func() {
				So(len(view.RecentSignals()), ShouldEqual, 0)
				So(len(view.RecentMetrics()), ShouldEqual, 0)
			})
		})
	})
}
