package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestDemoProjection verifies projection-aware memory example output.
*/
func TestDemoProjection(t *testing.T) {
	Convey("Given a memory demo", t, func() {
		var output bytes.Buffer
		demo, err := NewDemo(context.Background(), &output)
		So(err, ShouldBeNil)

		Convey("When the projection example runs", func() {
			err := demo.Projection()

			Convey("It should store projected memory", func() {
				text := output.String()

				So(err, ShouldBeNil)
				So(text, ShouldContainSubstring, "Projected memory")
				So(text, ShouldContainSubstring, "projection_embedding_dim=3")
				So(text, ShouldContainSubstring, "memory_embedding_dim=3")
				So(text, ShouldContainSubstring, "friction and quality signals")
			})
		})
	})
}

/*
BenchmarkDemoProjection measures the projection-aware memory example.
*/
func BenchmarkDemoProjection(benchmark *testing.B) {
	for benchmark.Loop() {
		demo, err := NewDemo(context.Background(), &strings.Builder{})
		if err != nil {
			benchmark.Fatal(err)
		}

		if err := demo.Projection(); err != nil {
			benchmark.Fatal(err)
		}
	}
}
