package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestDemoAnalog verifies structural analog memory output.
*/
func TestDemoAnalog(t *testing.T) {
	Convey("Given a memory demo", t, func() {
		var output bytes.Buffer
		demo, err := NewDemo(context.Background(), &output)
		So(err, ShouldBeNil)

		Convey("When the analog example runs", func() {
			err := demo.Analog()

			Convey("It should recall the closest prior task shape", func() {
				text := output.String()

				So(err, ShouldBeNil)
				So(text, ShouldContainSubstring, "DMT structural analog")
				So(text, ShouldContainSubstring, "workspace-lease-route")
				So(text, ShouldContainSubstring, "unleased task")
			})
		})
	})
}

/*
BenchmarkDemoAnalog measures the analog memory use case.
*/
func BenchmarkDemoAnalog(benchmark *testing.B) {
	for benchmark.Loop() {
		demo, err := NewDemo(context.Background(), &strings.Builder{})
		if err != nil {
			benchmark.Fatal(err)
		}

		if err := demo.Analog(); err != nil {
			benchmark.Fatal(err)
		}
	}
}
