package main

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/errnie"
)

/*
TestNewSignalProjector verifies signal projector construction.
*/
func TestNewSignalProjector(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a context", t, func() {
		projector, err := NewSignalProjector(context.Background())

		Convey("It should create a projector", func() {
			So(err, ShouldBeNil)
			So(projector, ShouldNotBeNil)
			So(projector.Close(), ShouldBeNil)
		})
	})
}

/*
TestSignalProjectorProject verifies single text projection.
*/
func TestSignalProjectorProject(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a signal projector", t, func() {
		projector, err := NewSignalProjector(context.Background())
		So(err, ShouldBeNil)
		defer projector.Close()

		Convey("When text contains quality signals", func() {
			projection, err := projector.Project(
				context.Background(),
				"Friction and quality signals should be remembered.",
			)

			Convey("It should produce compact embedding signals", func() {
				So(err, ShouldBeNil)
				So(projection.Embedding, ShouldHaveLength, 3)
				So(projection.Energy, ShouldBeGreaterThan, 0)
				So(projection.Surprise, ShouldBeGreaterThan, 0)
			})
		})

		Convey("When text is empty", func() {
			_, err := projector.Project(context.Background(), "")

			Convey("It should reject the projection", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

/*
TestSignalProjectorProjectBatch verifies batch projection.
*/
func TestSignalProjectorProjectBatch(t *testing.T) {
	Convey("Given a signal projector", t, func() {
		projector, err := NewSignalProjector(context.Background())
		So(err, ShouldBeNil)
		defer projector.Close()

		Convey("When multiple texts are projected", func() {
			projections, err := projector.ProjectBatch(context.Background(), []string{
				"Goal drift is a signal.",
				"Quality friction is another signal.",
			})

			Convey("It should project every text", func() {
				So(err, ShouldBeNil)
				So(projections, ShouldHaveLength, 2)
				So(projections[0].Embedding, ShouldHaveLength, 3)
				So(projections[1].Embedding, ShouldHaveLength, 3)
			})
		})
	})
}

/*
TestSignalProjectorClose verifies closed projector behavior.
*/
func TestSignalProjectorClose(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a closed signal projector", t, func() {
		projector, err := NewSignalProjector(context.Background())
		So(err, ShouldBeNil)
		So(projector.Close(), ShouldBeNil)

		Convey("When projection is requested", func() {
			_, err := projector.Project(context.Background(), "quality signal")

			Convey("It should reject the closed projector", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

/*
BenchmarkSignalProjectorProject measures signal projection.
*/
func BenchmarkSignalProjectorProject(benchmark *testing.B) {
	projector, err := NewSignalProjector(context.Background())
	if err != nil {
		benchmark.Fatal(err)
	}
	defer projector.Close()

	for benchmark.Loop() {
		_, err := projector.Project(
			context.Background(),
			"Structured outputs surface friction and quality signals.",
		)

		if err != nil {
			benchmark.Fatal(err)
		}
	}
}
