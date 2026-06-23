package ai

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/config"
	"github.com/theapemachine/qpool"
)

/*
TestWorkflowConversationAgents verifies conversation speakers become executable workflow agents.
*/
func TestWorkflowConversationAgents(t *testing.T) {
	Convey("Given a conversation workflow step", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		defer pool.Close()

		workflow, err := NewWorkflow(ctx, pool)
		So(err, ShouldBeNil)

		step := config.WorkflowStepYAML{
			ID:            "panel",
			Mode:          "conversation",
			StopCondition: "all_success",
			Conversation: &config.ConversationStepYAML{
				Rounds: 1,
				Speakers: []config.ConversationSpeakerSlotYAML{
					{Persona: "salon_optimist"},
					{Persona: "salon_critic"},
				},
			},
		}

		Convey("When stepAgents is called", func() {
			agents := workflow.stepAgents(config.WorkflowYAML{}, step, 0)

			Convey("Then it should preserve speaker order without requiring leases", func() {
				So(len(agents), ShouldEqual, 2)
				So(agents[0].slot.Persona, ShouldEqual, "salon_optimist")
				So(agents[1].slot.Persona, ShouldEqual, "salon_critic")
				So(agents[0].requireLeases, ShouldBeFalse)
				So(agents[1].slot.Replicas, ShouldEqual, 1)
			})
		})
	})
}
