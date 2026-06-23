package swarm

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestContentionBackoffDelay verifies exponential delay with bounded jitter.
*/
func TestContentionBackoffDelay(t *testing.T) {
	Convey("Given a contention backoff with deterministic jitter", t, func() {
		backoff, err := NewContentionBackoff(
			context.Background(),
			3,
			10*time.Millisecond,
			25*time.Millisecond,
			5*time.Millisecond,
		)
		So(err, ShouldBeNil)

		backoff.random = func() uint32 {
			return 2_000_000
		}

		Convey("When Delay is called for multiple attempts", func() {
			firstDelay, err := backoff.Delay(1)
			So(err, ShouldBeNil)

			secondDelay, err := backoff.Delay(2)
			So(err, ShouldBeNil)

			thirdDelay, err := backoff.Delay(3)

			Convey("Then delays should grow, cap, and include jitter", func() {
				So(err, ShouldBeNil)
				So(firstDelay, ShouldEqual, 12*time.Millisecond)
				So(secondDelay, ShouldEqual, 22*time.Millisecond)
				So(thirdDelay, ShouldEqual, 27*time.Millisecond)
			})
		})
	})
}

/*
TestContentionBackoffWait verifies Wait delegates to the configured sleeper.
*/
func TestContentionBackoffWait(t *testing.T) {
	Convey("Given a contention backoff with a captured sleeper", t, func() {
		backoff, err := NewContentionBackoff(
			context.Background(),
			2,
			time.Millisecond,
			time.Millisecond,
			0,
		)
		So(err, ShouldBeNil)

		var waited time.Duration
		backoff.sleep = func(ctx context.Context, delay time.Duration) error {
			waited = delay

			return nil
		}

		Convey("When Wait is called", func() {
			err := backoff.Wait(1)

			Convey("Then it should sleep for the calculated delay", func() {
				So(err, ShouldBeNil)
				So(waited, ShouldEqual, time.Millisecond)
			})
		})
	})
}

func BenchmarkContentionBackoffDelay(benchmark *testing.B) {
	backoff, err := NewContentionBackoff(
		context.Background(),
		3,
		time.Millisecond,
		16*time.Millisecond,
		time.Millisecond,
	)

	if err != nil {
		benchmark.Fatal(err)
	}

	benchmark.ReportAllocs()

	for benchmark.Loop() {
		if _, err := backoff.Delay(3); err != nil {
			benchmark.Fatal(err)
		}
	}
}
