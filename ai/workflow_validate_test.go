package ai

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/config"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/qpool"
)

/*
TestWorkflowValidate verifies invalid declarative workflow definitions fail before running.
*/
func TestWorkflowValidate(t *testing.T) {
	Convey("Given a workflow with required leases but no slot prefixes", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		defer pool.Close()

		registry, err := swarm.NewRegistry(ctx, pool, swarm.Options{
			MeshID:    "workflow-validate-test",
			GossipTTL: 30 * time.Second,
			MeshTTL:   time.Minute,
			Buffer:    16,
		}, lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		workflow, err := NewWorkflow(ctx, pool)
		So(err, ShouldBeNil)

		definition := config.WorkflowYAML{
			BroadcastGroupTTLSeconds: 60,
			FileLeasing:              true,
			Steps: []config.WorkflowStepYAML{
				{
					ID:                "build",
					RequireFileLeases: true,
					StopCondition:     "all_success",
					Slots: []config.WorkflowSlotYAML{
						{Persona: "developer", Replicas: 1},
					},
				},
			},
		}

		Convey("When validate is called", func() {
			err := workflow.validate(definition, registry)

			Convey("Then it should reject the missing lease contract", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "lease prefixes")
			})
		})
	})
}
