package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestDemoCognitive verifies DMT cognitive memory example output.
*/
func TestDemoCognitive(t *testing.T) {
	Convey("Given a memory demo", t, func() {
		var output bytes.Buffer
		demo, err := NewDemo(context.Background(), &output)
		So(err, ShouldBeNil)

		Convey("When the cognitive example runs", func() {
			err := demo.Cognitive()

			Convey("It should recall DMT memory and relationships", func() {
				text := output.String()

				So(err, ShouldBeNil)
				So(text, ShouldContainSubstring, "DMT cognitive memory")
				So(text, ShouldContainSubstring, "episodic buffers")
				So(text, ShouldContainSubstring, "Relationship recall")
				So(text, ShouldContainSubstring, "complements")
			})
		})
	})
}

/*
BenchmarkDemoCognitive measures the DMT cognitive memory example.
*/
func BenchmarkDemoCognitive(benchmark *testing.B) {
	for benchmark.Loop() {
		demo, err := NewDemo(context.Background(), &strings.Builder{})
		if err != nil {
			benchmark.Fatal(err)
		}

		if err := demo.Cognitive(); err != nil {
			benchmark.Fatal(err)
		}
	}
}
