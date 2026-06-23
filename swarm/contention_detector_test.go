package swarm

import (
	"context"
	"testing"
	"time"
)

func BenchmarkContentionDetectorDetect(benchmark *testing.B) {
	view, err := NewView(time.Minute)

	if err != nil {
		benchmark.Fatal(err)
	}

	base := time.Unix(100, 0)

	for index := range 64 {
		contention, err := testContention(
			"actor-a",
			"lanes/b/",
			"actor-b",
			base.Add(time.Duration(index)*time.Millisecond),
		)

		if err != nil {
			benchmark.Fatal(err)
		}

		if err := view.MergeContention(contention); err != nil {
			benchmark.Fatal(err)
		}
	}

	detector, err := NewContentionDetector(
		context.Background(),
		view,
		ContentionDetectorOptions{
			Window:              time.Minute,
			StarvationThreshold: 3,
		},
	)

	if err != nil {
		benchmark.Fatal(err)
	}

	benchmark.ReportAllocs()

	for benchmark.Loop() {
		if _, err := detector.Detect(base.Add(time.Second)); err != nil {
			benchmark.Fatal(err)
		}
	}
}
