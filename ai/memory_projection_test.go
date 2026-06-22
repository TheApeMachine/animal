package ai

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/errnie"
)

func TestNewProjectedMemory(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given memory and projector", t, func() {
		memory, err := NewLocalMemory(context.Background())
		So(err, ShouldBeNil)
		projector := &fakeMemoryProjector{}

		Convey("It should create projected memory", func() {
			projectedMemory, err := NewProjectedMemory(context.Background(), memory, projector)

			So(err, ShouldBeNil)
			So(projectedMemory, ShouldNotBeNil)
			So(projectedMemory.Close(), ShouldBeNil)
		})
	})
}

func TestProjectedMemoryRecall(t *testing.T) {
	Convey("Given projected memory with stored record", t, func() {
		ctx := context.Background()
		memory, err := NewLocalMemory(ctx)
		So(err, ShouldBeNil)

		projectedMemory, err := NewProjectedMemory(ctx, memory, &fakeMemoryProjector{})
		So(err, ShouldBeNil)
		defer projectedMemory.Close()

		So(projectedMemory.Remember(ctx, MemoryConsolidation{
			Records: []MemoryRecord{
				{ID: "projection-recall", Scope: "goal-projection", Text: "latent proof", Importance: 0.9},
			},
		}), ShouldBeNil)

		Convey("It should delegate recall", func() {
			packet, err := projectedMemory.Recall(ctx, MemoryRecallPlan{
				Queries: []MemoryQuery{
					{Scope: "goal-projection", Text: "proof", Limit: 4, TextWeight: 1},
				},
			})

			So(err, ShouldBeNil)
			So(packet.Documents, ShouldHaveLength, 1)
		})
	})
}

func TestProjectedMemoryRemember(t *testing.T) {
	Convey("Given projected memory", t, func() {
		ctx := context.Background()
		memory, err := NewLocalMemory(ctx)
		So(err, ShouldBeNil)

		projectedMemory, err := NewProjectedMemory(ctx, memory, &fakeMemoryProjector{})
		So(err, ShouldBeNil)
		defer projectedMemory.Close()

		Convey("It should store projected embeddings", func() {
			err := projectedMemory.Remember(ctx, MemoryConsolidation{
				Records: []MemoryRecord{
					{ID: "projection-remember", Scope: "goal-projection", Text: "latent proof", Importance: 0.9},
				},
			})

			So(err, ShouldBeNil)

			packet, err := projectedMemory.Recall(ctx, MemoryRecallPlan{
				Queries: []MemoryQuery{
					{Scope: "goal-projection", Text: "latent", Limit: 4, TextWeight: 1},
				},
			})

			So(err, ShouldBeNil)
			So(packet.Documents[0].Embedding, ShouldResemble, []float32{1, 2, 3})
		})
	})
}

func TestProjectedMemoryForget(t *testing.T) {
	Convey("Given projected memory", t, func() {
		ctx := context.Background()
		memory, err := NewLocalMemory(ctx)
		So(err, ShouldBeNil)

		projectedMemory, err := NewProjectedMemory(ctx, memory, &fakeMemoryProjector{})
		So(err, ShouldBeNil)
		defer projectedMemory.Close()

		So(projectedMemory.Remember(ctx, MemoryConsolidation{
			Records: []MemoryRecord{
				{ID: "projection-forget", Scope: "goal-projection", Text: "forget proof", Importance: 0.9},
			},
		}), ShouldBeNil)

		Convey("It should delegate forget", func() {
			So(projectedMemory.Forget(ctx, []string{"projection-forget"}), ShouldBeNil)
		})
	})
}

func TestProjectedMemoryClose(t *testing.T) {
	Convey("Given projected memory", t, func() {
		ctx := context.Background()
		memory, err := NewLocalMemory(ctx)
		So(err, ShouldBeNil)
		projector := &fakeMemoryProjector{}

		projectedMemory, err := NewProjectedMemory(ctx, memory, projector)
		So(err, ShouldBeNil)

		Convey("It should close projector and memory", func() {
			So(projectedMemory.Close(), ShouldBeNil)
			So(projector.closed, ShouldBeTrue)
		})
	})
}

func BenchmarkProjectedMemoryRemember(benchmark *testing.B) {
	ctx := context.Background()
	memory, err := NewLocalMemory(ctx)
	if err != nil {
		benchmark.Fatal(err)
	}

	projectedMemory, err := NewProjectedMemory(ctx, memory, &fakeMemoryProjector{})
	if err != nil {
		benchmark.Fatal(err)
	}
	defer projectedMemory.Close()

	for benchmark.Loop() {
		err := projectedMemory.Remember(ctx, MemoryConsolidation{
			Records: []MemoryRecord{
				{ID: "projection-bench", Scope: "goal-projection", Text: "latent proof", Importance: 0.9},
			},
		})

		if err != nil {
			benchmark.Fatal(err)
		}
	}
}

type fakeMemoryProjector struct {
	closed bool
}

func (projector *fakeMemoryProjector) Project(
	ctx context.Context,
	text string,
) (MemoryProjection, error) {
	projections, err := projector.ProjectBatch(ctx, []string{text})
	if err != nil {
		return MemoryProjection{}, err
	}

	return projections[0], nil
}

func (projector *fakeMemoryProjector) ProjectBatch(
	ctx context.Context,
	texts []string,
) ([]MemoryProjection, error) {
	projections := make([]MemoryProjection, len(texts))

	for index := range texts {
		projections[index] = MemoryProjection{
			Embedding: []float32{1, 2, 3},
			Energy:    0.5,
			Surprise:  0.25,
		}
	}

	return projections, nil
}

func (projector *fakeMemoryProjector) Close() error {
	projector.closed = true

	return nil
}
