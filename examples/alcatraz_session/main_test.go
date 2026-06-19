package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/errnie"
)

func TestNewScriptedStreamer(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given stream chunks", t, func() {
		streamer, err := NewScriptedStreamer(context.Background(), "make", " test", "\n")

		Convey("It should create a streamer", func() {
			So(err, ShouldBeNil)
			So(streamer, ShouldNotBeNil)
			So(streamer.chunks, ShouldResemble, []string{"make", " test", "\n"})
		})
	})

	Convey("Given no stream chunks", t, func() {
		streamer, err := NewScriptedStreamer(context.Background())

		Convey("It should reject construction", func() {
			So(streamer, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})
}

func TestStreamWithSink(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a scripted streamer", t, func() {
		streamer, err := NewScriptedStreamer(context.Background(), "printf ", "'ok\\n'", "\n")
		So(err, ShouldBeNil)

		Convey("It should stream every chunk to the sink", func() {
			var builder strings.Builder

			err := streamer.StreamWithSink(
				"",
				provider.NewContext(context.Background()),
				provider.NewParams(),
				collectSink(&builder),
			)

			So(err, ShouldBeNil)
			So(builder.String(), ShouldEqual, "printf 'ok\\n'\n")
		})

		Convey("It should return sink errors", func() {
			expected := errors.New("sink failed")

			err := streamer.StreamWithSink(
				"",
				provider.NewContext(context.Background()),
				provider.NewParams(),
				func(chunk string) error {
					return expected
				},
			)

			So(err, ShouldEqual, expected)
		})
	})

	Convey("Given a zero-value streamer", t, func() {
		streamer := &ScriptedStreamer{}

		Convey("It should reject streaming", func() {
			err := streamer.StreamWithSink(
				"",
				provider.NewContext(context.Background()),
				provider.NewParams(),
				func(chunk string) error { return nil },
			)

			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})
}

func TestRequireDocker(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	originalDockerInfo := dockerInfo
	defer func() {
		dockerInfo = originalDockerInfo
	}()

	Convey("Given Docker responds", t, func() {
		dockerInfo = func(ctx context.Context) ([]byte, error) {
			return []byte("ok"), nil
		}

		Convey("It should allow the example to continue", func() {
			err := requireDocker(context.Background())

			So(err, ShouldBeNil)
		})
	})

	Convey("Given Docker is unavailable", t, func() {
		dockerInfo = func(ctx context.Context) ([]byte, error) {
			return []byte("daemon unavailable"), errors.New("docker failed")
		}

		Convey("It should return an IO error", func() {
			err := requireDocker(context.Background())

			So(err, ShouldNotBeNil)
			So(errnie.IsIO(err), ShouldBeTrue)
			So(err.Error(), ShouldContainSubstring, "daemon unavailable")
		})
	})
}

func BenchmarkNewScriptedStreamer(benchmark *testing.B) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	for benchmark.Loop() {
		_, _ = NewScriptedStreamer(context.Background(), "make", " test", "\n")
	}
}

func BenchmarkStreamWithSink(benchmark *testing.B) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	streamer, err := NewScriptedStreamer(context.Background(), "make", " test", "\n")
	if err != nil {
		benchmark.Fatal(err)
	}

	var builder strings.Builder

	for benchmark.Loop() {
		builder.Reset()

		_ = streamer.StreamWithSink(
			"",
			provider.NewContext(context.Background()),
			provider.NewParams(),
			collectSink(&builder),
		)
	}
}

func BenchmarkRequireDocker(benchmark *testing.B) {
	originalDockerInfo := dockerInfo
	defer func() {
		dockerInfo = originalDockerInfo
	}()

	dockerInfo = func(ctx context.Context) ([]byte, error) {
		return []byte("ok"), nil
	}

	for benchmark.Loop() {
		_ = requireDocker(context.Background())
	}
}

func collectSink(builder *strings.Builder) func(string) error {
	return func(chunk string) error {
		builder.WriteString(chunk)

		return nil
	}
}
