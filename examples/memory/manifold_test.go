//go:build darwin && cgo

package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestDemoManifold verifies manifold-projected memory example output.
*/
func TestDemoManifold(t *testing.T) {
	Convey("Given a memory demo", t, func() {
		var output bytes.Buffer
		demo, err := NewDemo(context.Background(), &output)
		So(err, ShouldBeNil)

		Convey("When the manifold example runs", func() {
			err := demo.Manifold()
			skipManifoldUnavailable(t, err)

			Convey("It should project memory before storage", func() {
				text := output.String()

				So(err, ShouldBeNil)
				So(text, ShouldContainSubstring, "Manifold-projected memory")
				So(text, ShouldContainSubstring, "projection_embedding_dim=4")
				So(text, ShouldContainSubstring, "memory_embedding_dim=4")
				So(text, ShouldContainSubstring, "latent energy surprise")
			})
		})
	})
}

/*
BenchmarkDemoManifold measures the optional manifold memory example.
*/
func BenchmarkDemoManifold(benchmark *testing.B) {
	for benchmark.Loop() {
		var output bytes.Buffer
		demo, err := NewDemo(context.Background(), &output)
		if err != nil {
			benchmark.Fatal(err)
		}

		if err := demo.Manifold(); err != nil {
			skipManifoldBenchmarkUnavailable(benchmark, err)

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
