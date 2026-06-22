package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestDemoFriction verifies swarm friction memory output.
*/
func TestDemoFriction(t *testing.T) {
	Convey("Given a memory demo", t, func() {
		var output bytes.Buffer
		demo, err := NewDemo(context.Background(), &output)
		So(err, ShouldBeNil)

		Convey("When the friction example runs", func() {
			err := demo.Friction()

			Convey("It should recall friction records and risks", func() {
				text := output.String()

				So(err, ShouldBeNil)
				So(text, ShouldContainSubstring, "DMT friction backlog")
				So(text, ShouldContainSubstring, "Lease blocker")
				So(text, ShouldContainSubstring, "Quality drift")
				So(text, ShouldContainSubstring, "risks")
			})
		})
	})
}

/*
BenchmarkDemoFriction measures the friction memory use case.
*/
func BenchmarkDemoFriction(benchmark *testing.B) {
	for benchmark.Loop() {
		demo, err := NewDemo(context.Background(), &strings.Builder{})
		if err != nil {
			benchmark.Fatal(err)
		}

		if err := demo.Friction(); err != nil {
			benchmark.Fatal(err)
		}
	}
}
