package internal

import (
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestSnapshotLoad verifies the initial snapshot value is readable.
*/
func TestSnapshotLoad(t *testing.T) {
	Convey("Given a snapshot seeded with one counter", t, func() {
		snapshot := NewSnapshot(1)

		Convey("When Load is called", func() {
			value := snapshot.Load()

			Convey("Then it should return the seeded value", func() {
				So(value, ShouldEqual, 1)
			})
		})
	})
}

/*
TestSnapshotUpdate verifies copy-on-write updates publish new values.
*/
func TestSnapshotUpdate(t *testing.T) {
	Convey("Given a snapshot seeded with one counter", t, func() {
		snapshot := NewSnapshot(1)

		Convey("When Update increments the value", func() {
			snapshot.Update(func(value int) int {
				return value + 1
			})

			Convey("Then Load should return the updated value", func() {
				So(snapshot.Load(), ShouldEqual, 2)
			})
		})
	})
}

/*
TestSnapshotUpdateConcurrent verifies concurrent writers converge safely.
*/
func TestSnapshotUpdateConcurrent(t *testing.T) {
	Convey("Given a snapshot seeded with zero", t, func() {
		snapshot := NewSnapshot(0)
		waitGroup := sync.WaitGroup{}

		for range 32 {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				snapshot.Update(func(value int) int {
					return value + 1
				})
			}()
		}

		waitGroup.Wait()

		Convey("When all goroutines finish", func() {
			Convey("Then the counter should reach thirty-two", func() {
				So(snapshot.Load(), ShouldEqual, 32)
			})
		})
	})
}
