package ai

import (
	"context"
	"strings"
	"time"

	"github.com/theapemachine/animal/config"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
WorkflowStatus identifies a workflow, step, or slot terminal state.
*/
type WorkflowStatus string

const (
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
)

/*
WorkflowResult captures one execution of a configured workflow.
*/
type WorkflowResult struct {
	Status    WorkflowStatus
	StartedAt time.Time
	EndedAt   time.Time
	Steps     []WorkflowStepResult
}

/*
WorkflowStepResult captures one configured workflow phase.
*/
type WorkflowStepResult struct {
	ID          string
	Description string
	Status      WorkflowStatus
	StartedAt   time.Time
	EndedAt     time.Time
	Agents      []WorkflowAgentResult
}

/*
WorkflowAgentResult captures one spawned slot replica.
*/
type WorkflowAgentResult struct {
	AgentID  string
	Name     string
	Role     string
	TaskID   string
	ReadOnly bool
	Prefixes []string
	Claimed  []string
	Status   WorkflowStatus
	Error    string
}

/*
Workflow coordinates multi-agent collaboration using qpool jobs and optional broadcast leases.
*/
type Workflow struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	pool   *qpool.Q[any]
}

type workflowAgent struct {
	step          config.WorkflowStepYAML
	slot          config.WorkflowSlotYAML
	stepIndex     int
	slotIndex     int
	replicaIndex  int
	requireLeases bool
}

type workflowWait struct {
	agent workflowAgent
	wait  *qpool.ResultWait[any]
}

func NewWorkflow(ctx context.Context, pool *qpool.Q[any]) (*Workflow, error) {
	ctx, cancel := context.WithCancel(ctx)

	workflow := &Workflow{
		ctx:    ctx,
		cancel: cancel,
		pool:   pool,
	}

	return workflow, errnie.Require(map[string]any{
		"ctx":    workflow.ctx,
		"cancel": workflow.cancel,
		"pool":   workflow.pool,
	})
}

/*
Run executes a declarative workflow by spawning configured agents and enforcing lease gates.
*/
func (workflow *Workflow) Run(
	definition config.WorkflowYAML,
	registry *swarm.Registry,
) (WorkflowResult, error) {
	result := WorkflowResult{
		Status:    WorkflowStatusFailed,
		StartedAt: time.Now().UTC(),
		Steps:     make([]WorkflowStepResult, 0, len(definition.Steps)),
	}

	if err := workflow.validate(definition, registry); err != nil {
		result.EndedAt = time.Now().UTC()
		return result, err
	}

	resultTTL := time.Duration(definition.BroadcastGroupTTLSeconds) * time.Second

	for stepIndex, step := range definition.Steps {
		stepResult, err := workflow.runStep(
			definition, step, stepIndex, registry, resultTTL,
		)
		result.Steps = append(result.Steps, stepResult)

		if err != nil {
			result.EndedAt = time.Now().UTC()
			return result, err
		}
	}

	result.Status = WorkflowStatusCompleted
	result.EndedAt = time.Now().UTC()

	return result, nil
}

func (workflow *Workflow) runStep(
	definition config.WorkflowYAML,
	step config.WorkflowStepYAML,
	stepIndex int,
	registry *swarm.Registry,
	resultTTL time.Duration,
) (WorkflowStepResult, error) {
	result := WorkflowStepResult{
		ID:          step.ID,
		Description: step.Description,
		Status:      WorkflowStatusFailed,
		StartedAt:   time.Now().UTC(),
	}

	agents := workflow.stepAgents(definition, step, stepIndex)

	if step.ParallelAgents {
		return workflow.runParallel(result, agents, registry, resultTTL)
	}

	return workflow.runSequential(result, agents, registry)
}

func (workflow *Workflow) runParallel(
	result WorkflowStepResult,
	agents []workflowAgent,
	registry *swarm.Registry,
	resultTTL time.Duration,
) (WorkflowStepResult, error) {
	waits := make([]workflowWait, 0, len(agents))

	for _, agent := range agents {
		waits = append(waits, workflowWait{
			agent: agent,
			wait: workflow.pool.Schedule(
				workflow.jobID(agent),
				func(ctx context.Context) (any, error) {
					return workflow.runAgent(ctx, agent, registry)
				},
				qpool.WithTTL(resultTTL),
			),
		})
	}

	agentResults, err := workflow.collect(waits)
	result.Agents = agentResults
	result.EndedAt = time.Now().UTC()

	if err != nil {
		return result, err
	}

	result.Status = WorkflowStatusCompleted

	return result, nil
}

func (workflow *Workflow) runSequential(
	result WorkflowStepResult,
	agents []workflowAgent,
	registry *swarm.Registry,
) (WorkflowStepResult, error) {
	result.Agents = make([]WorkflowAgentResult, 0, len(agents))

	for _, agent := range agents {
		agentResult, err := workflow.runAgent(workflow.ctx, agent, registry)
		result.Agents = append(result.Agents, agentResult)

		if err != nil {
			result.EndedAt = time.Now().UTC()
			return result, err
		}
	}

	result.Status = WorkflowStatusCompleted
	result.EndedAt = time.Now().UTC()

	return result, nil
}

func (workflow *Workflow) collect(
	waits []workflowWait,
) ([]WorkflowAgentResult, error) {
	results := make([]WorkflowAgentResult, 0, len(waits))
	failures := make([]string, 0)

	for _, queued := range waits {
		artifact, err := queued.wait.Get(workflow.ctx)

		if err != nil {
			failures = append(failures, err.Error())
			results = append(results, workflow.failedAgentResult(queued.agent, err.Error()))
			continue
		}

		agentResult, err := qpool.ArtifactValue[WorkflowAgentResult](artifact)

		if err != nil {
			failures = append(failures, err.Error())
			results = append(results, workflow.failedAgentResult(queued.agent, err.Error()))
			continue
		}

		if agentResult.Status == WorkflowStatusFailed {
			failures = append(failures, agentResult.Error)
		}

		results = append(results, agentResult)
	}

	if len(failures) > 0 {
		return results, errnie.Err(
			errnie.Validation,
			strings.Join(failures, "; "),
			nil,
		)
	}

	return results, nil
}
