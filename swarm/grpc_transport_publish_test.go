package swarm

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/datura"
	"github.com/theapemachine/qpool"
)

/*
TestGRPCMeshTransportPublish verifies typed mesh traffic crosses process-local meshes.
*/
func TestGRPCMeshTransportPublish(t *testing.T) {
	Convey("Given two qpool meshes bridged by gRPC", t, func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		receiverPool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		senderPool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		receiverTransport, err := NewGRPCMeshTransport(ctx, "127.0.0.1:0", nil)
		So(err, ShouldBeNil)

		receiverOptions := testSwarmOptions()
		receiverOptions.MeshID = "grpc-transport-test"
		receiverOptions.Transport = receiverTransport

		receiverMesh, err := NewMesh(ctx, receiverPool, receiverOptions)
		So(err, ShouldBeNil)

		defer func() {
			So(receiverMesh.Close(), ShouldBeNil)
		}()

		subscriber, err := receiverMesh.Subscribe("actor-a", 8)
		So(err, ShouldBeNil)

		senderTransport, err := NewGRPCMeshTransport(
			ctx,
			"127.0.0.1:0",
			[]string{receiverTransport.Address()},
		)
		So(err, ShouldBeNil)

		senderOptions := testSwarmOptions()
		senderOptions.MeshID = "grpc-transport-test"
		senderOptions.Transport = senderTransport

		senderMesh, err := NewMesh(ctx, senderPool, senderOptions)
		So(err, ShouldBeNil)

		defer func() {
			So(senderMesh.Close(), ShouldBeNil)
		}()

		rumor := NewRumorAt(
			KindStatus,
			"actor-b",
			"Bob",
			"developer",
			time.Now(),
		)
		rumor.State = "remote"

		Convey("When the sender publishes a rumor", func() {
			err := senderMesh.Publish("actor-b", rumor)
			So(err, ShouldBeNil)

			artifact, err := waitBroadcastConsumer(ctx, subscriber, time.Second)
			So(err, ShouldBeNil)

			payload := datura.As[Rumor](artifact)

			Convey("Then the receiver mesh should receive the typed rumor", func() {
				So(payload.ActorID, ShouldEqual, "actor-b")
				So(payload.State, ShouldEqual, "remote")
			})
		})
	})
}

func BenchmarkGRPCMeshTransportPublish(benchmark *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	receiverPool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
	senderPool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	receiverTransport, err := NewGRPCMeshTransport(ctx, "127.0.0.1:0", nil)

	if err != nil {
		benchmark.Fatal(err)
	}

	receiverOptions := testSwarmOptions()
	receiverOptions.MeshID = "grpc-transport-benchmark"
	receiverOptions.Transport = receiverTransport

	receiverMesh, err := NewMesh(ctx, receiverPool, receiverOptions)

	if err != nil {
		benchmark.Fatal(err)
	}

	benchmark.Cleanup(func() {
		if err := receiverMesh.Close(); err != nil {
			benchmark.Fatal(err)
		}
	})

	subscriber, err := receiverMesh.Subscribe("actor-a", 1024)

	if err != nil {
		benchmark.Fatal(err)
	}

	senderTransport, err := NewGRPCMeshTransport(
		ctx,
		"127.0.0.1:0",
		[]string{receiverTransport.Address()},
	)

	if err != nil {
		benchmark.Fatal(err)
	}

	senderOptions := testSwarmOptions()
	senderOptions.MeshID = "grpc-transport-benchmark"
	senderOptions.Transport = senderTransport

	senderMesh, err := NewMesh(ctx, senderPool, senderOptions)

	if err != nil {
		benchmark.Fatal(err)
	}

	benchmark.Cleanup(func() {
		if err := senderMesh.Close(); err != nil {
			benchmark.Fatal(err)
		}
	})

	rumor := NewRumorAt(
		KindStatus,
		"actor-b",
		"Bob",
		"developer",
		time.Unix(100, 0),
	)
	rumor.State = "remote"

	benchmark.ReportAllocs()
	benchmark.ResetTimer()

	for benchmark.Loop() {
		if err := senderMesh.Publish("actor-b", rumor); err != nil {
			benchmark.Fatal(err)
		}

		artifact, err := waitBroadcastConsumer(ctx, subscriber, time.Second)

		if err != nil {
			benchmark.Fatal(err)
		}

		_ = datura.As[Rumor](artifact)
	}
}
