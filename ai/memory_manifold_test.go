//go:build darwin && cgo

package ai

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/errnie"
)

func TestNewManifoldProjector(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given valid manifold config", t, func() {
		projector, err := NewManifoldProjector(context.Background(), testManifoldConfig())
		skipManifoldUnavailable(t, err)

		Convey("It should create a projector", func() {
			So(err, ShouldBeNil)
			if err != nil {
				return
			}

			So(projector, ShouldNotBeNil)
			So(projector.Close(), ShouldBeNil)
		})
	})

	Convey("Given invalid manifold config", t, func() {
		projector, err := NewManifoldProjector(context.Background(), ManifoldConfig{})

		Convey("It should reject construction", func() {
			So(projector, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})
}

func TestManifoldConfigValidate(t *testing.T) {
	Convey("Given valid manifold config", t, func() {
		Convey("It should pass validation", func() {
			So(testManifoldConfig().Validate(), ShouldBeNil)
		})
	})
}

func TestManifoldProjectorProject(t *testing.T) {
	Convey("Given a manifold projector", t, func() {
		projector, err := NewManifoldProjector(context.Background(), testManifoldConfig())
		skipManifoldUnavailable(t, err)
		So(err, ShouldBeNil)
		if err != nil {
			return
		}

		defer projector.Close()

		Convey("It should project text into latent memory signal", func() {
			projection, err := projector.Project(context.Background(), "make test proof memory")

			So(err, ShouldBeNil)
			So(projection.Embedding, ShouldHaveLength, 4)
			So(math.IsNaN(projection.Energy), ShouldBeFalse)
			So(math.IsNaN(projection.Surprise), ShouldBeFalse)
		})
	})
}

func TestManifoldProjectorProjectBatch(t *testing.T) {
	Convey("Given a manifold projector", t, func() {
		projector, err := NewManifoldProjector(context.Background(), testManifoldConfig())
		skipManifoldUnavailable(t, err)
		So(err, ShouldBeNil)
		if err != nil {
			return
		}

		defer projector.Close()

		Convey("It should project multiple text samples", func() {
			projections, err := projector.ProjectBatch(context.Background(), []string{
				"make test proof memory",
				"swarm friction observation",
			})

			So(err, ShouldBeNil)
			So(projections, ShouldHaveLength, 2)
			So(projections[0].Embedding, ShouldHaveLength, 4)
			So(projections[1].Embedding, ShouldHaveLength, 4)
		})
	})
}

func TestManifoldInput(t *testing.T) {
	Convey("Given text and dimension", t, func() {
		input := manifoldInput("Make test proof memory", 8)

		Convey("It should produce normalized deterministic input", func() {
			So(input, ShouldHaveLength, 8)
			So(math.Abs(vectorNorm(input)-1), ShouldBeLessThan, 1e-9)
			So(input, ShouldResemble, manifoldInput("Make test proof memory", 8))
		})
	})
}

func BenchmarkManifoldProjectorProject(benchmark *testing.B) {
	projector, err := NewManifoldProjector(context.Background(), testManifoldConfig())
	if err != nil {
		skipManifoldBenchmarkUnavailable(benchmark, err)

		benchmark.Fatal(err)
	}
	defer projector.Close()

	for benchmark.Loop() {
		if _, err := projector.Project(context.Background(), "make test proof memory"); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func skipManifoldUnavailable(t *testing.T, err error) {
	t.Helper()

	for cause := err; cause != nil; cause = errors.Unwrap(cause) {
		if strings.Contains(cause.Error(), "Metal device unavailable") {
			t.Skipf("nomagique manifold requires Metal: %v", cause)
		}
	}
}

func skipManifoldBenchmarkUnavailable(benchmark *testing.B, err error) {
	benchmark.Helper()

	for cause := err; cause != nil; cause = errors.Unwrap(cause) {
		if strings.Contains(cause.Error(), "Metal device unavailable") {
			benchmark.Skipf("nomagique manifold requires Metal: %v", cause)
		}
	}
}

func testManifoldConfig() ManifoldConfig {
	return ManifoldConfig{
		Architecture: []int{8, 6, 4},
		TargetDim:    0,
		Batch:        2,
		Alpha:        0.05,
	}
}

func vectorNorm(values []float64) float64 {
	sumSquares := 0.0

	for _, value := range values {
		sumSquares += value * value
	}

	return math.Sqrt(sumSquares)
}
