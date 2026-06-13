package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/animal/ai/agent"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/ai/tool/editor"
	editoragent "github.com/theapemachine/animal/ai/tool/editor/agent"
	"github.com/theapemachine/animal/examples/support"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/qpool"
)

/*
Config controls the long-horizon coding loop.
*/
type Config struct {
	Workspace string
	Goal      string
	MaxCycles int
	DryRun    bool
}

/*
Orchestrator runs observe → intake → cycle(recon, plan, mutate, prove, audit).
*/
type Orchestrator struct {
	ctx            context.Context
	pool           *qpool.Q[any]
	config         Config
	observer       *Observer
	verifier       *Verifier
	backlog        *Backlog
	digest         *RepoDigest
	participant    *swarm.Participant
	editorServer   *editor.Server
	scoutSession   agentRunner
	surgeonSession agentRunner
	openai         *provider.OpenAI
	cycle          int
}

type agentRunner struct {
	runner *agent.Runner
}

func newOrchestrator(ctx context.Context, pool *qpool.Q[any], config Config) (*Orchestrator, error) {
	workspace, err := filepath.Abs(config.Workspace)
	if err != nil {
		return nil, fmt.Errorf("coding horizon: workspace path: %w", err)
	}

	if setErr := os.Setenv("ANIMAL_AGENT_WORKSPACE", workspace); setErr != nil {
		return nil, fmt.Errorf("coding horizon: set workspace env: %w", setErr)
	}

	registry, err := swarm.NewRegistry(
		ctx,
		pool,
		support.DefaultSwarmOptions("coding-horizon"),
		support.DefaultLeaseOptions(),
	)
	if err != nil {
		return nil, err
	}

	codingAgent, err := ai.NewAgent(ctx, pool, "surgeon", "Horizon", registry, []string{"lanes/coding/"})
	if err != nil {
		return nil, err
	}

	editorServer, err := editor.NewServer(ctx, pool)
	if err != nil {
		return nil, err
	}

	scoutAccess := editoragent.Access{ID: "scout", ReadOnly: true}
	surgeonAccess := editoragent.Access{ID: "surgeon", RequireLease: true, LeasePrefixes: []string{"lanes/coding/"}}

	scoutSession, scoutErr := editorServer.ClientSession(ctx, scoutAccess)
	if scoutErr != nil {
		return nil, scoutErr
	}

	surgeonSession, surgeonErr := editorServer.ClientSession(ctx, surgeonAccess)
	if surgeonErr != nil {
		return nil, surgeonErr
	}

	endpoint, apiKey, model := support.OpenAIConfig()

	openaiProvider, providerErr := provider.NewOpenAI(ctx, pool, endpoint, apiKey, model)
	if providerErr != nil {
		return nil, providerErr
	}

	orchestrator := &Orchestrator{
		ctx:          ctx,
		pool:         pool,
		config:       config,
		observer:     newObserver(workspace),
		verifier:     newVerifier(workspace),
		backlog:      newBacklog(config.Goal),
		participant:  codingAgent.Participant(),
		editorServer: editorServer,
		scoutSession: agentRunner{runner: agent.NewRunner(endpoint, apiKey, model, scoutSession, 10)},
		surgeonSession: agentRunner{
			runner: agent.NewRunner(endpoint, apiKey, model, surgeonSession, 8),
		},
		openai: openaiProvider,
	}

	return orchestrator, nil
}

func (orchestrator *Orchestrator) Close() {
	orchestrator.editorServer.Close()
}

func (orchestrator *Orchestrator) Run() error {
	digest, err := orchestrator.observer.Digest()
	if err != nil {
		return err
	}

	orchestrator.digest = digest

	fmt.Printf("coding horizon workspace: %s\n", digest.Root)
	fmt.Printf("module=%s go_files=%d test_files=%d\n", digest.Module, digest.GoFiles, digest.TestFiles)

	if intakeErr := orchestrator.intake(); intakeErr != nil {
		return intakeErr
	}

	orchestrator.announce(topicGoal, orchestrator.backlog.Summary())

	for orchestrator.cycle < orchestrator.config.MaxCycles {
		task, ok := orchestrator.backlog.Next()
		if !ok {
			break
		}

		orchestrator.cycle++
		fmt.Printf("\n--- cycle %d / %d: %s ---\n", orchestrator.cycle, orchestrator.config.MaxCycles, task.Title)

		if cycleErr := orchestrator.runCycle(task); cycleErr != nil {
			orchestrator.backlog.MarkBlocked(task.ID, cycleErr.Error())
			orchestrator.announce(topicTask, fmt.Sprintf("blocked %s: %v", task.ID, cycleErr))
			fmt.Fprintf(os.Stderr, "cycle blocked: %v\n", cycleErr)

			continue
		}
	}

	if auditErr := orchestrator.finalAudit(); auditErr != nil {
		return auditErr
	}

	fmt.Println("\n--- backlog ---")
	fmt.Print(orchestrator.backlog.Summary())

	return nil
}

func (orchestrator *Orchestrator) intake() error {
	orchestrator.backlog.Merge(hygieneTasksFromDigest(orchestrator.digest))

	if strings.TrimSpace(orchestrator.config.Goal) == "" {
		fmt.Println("no goal provided; running hygiene-only horizon")
		orchestrator.announce(topicGoal, "hygiene-only horizon")

		return nil
	}

	if orchestrator.config.DryRun {
		orchestrator.backlog.Add(Task{
			ID:          "goal-1",
			Kind:        taskKindGoal,
			Title:       orchestrator.config.Goal,
			Rationale:   "Dry-run placeholder goal task for deterministic pipeline testing.",
			TargetFiles: orchestrator.digest.SamplePaths,
			Acceptance:  "Tests pass after a verified change or task is explicitly blocked with evidence.",
		})

		return nil
	}

	var intake intakeResult

	decodeErr := decodeStructured(
		orchestrator.openai,
		orchestrator.ctx,
		intakeSystemPrompt,
		intakeUserPrompt(orchestrator.config.Goal, orchestrator.digest),
		provider.StructuredOutput{
			Name:        "coding_intake",
			Description: "Goal decomposition for AI-native coding horizon",
			Schema:      intakeSchema,
			Strict:      false,
		},
		&intake,
	)

	if decodeErr != nil {
		return decodeErr
	}

	for _, task := range intake.GoalTasks {
		task.Kind = taskKindGoal
		orchestrator.backlog.Add(task)
	}

	orchestrator.announce(topicGoal, intake.Summary)

	return nil
}

func (orchestrator *Orchestrator) runCycle(task *Task) error {
	if orchestrator.config.DryRun {
		fmt.Println("dry-run: skipping recon/plan/mutate phases")

		verifyOutput, verifyErr := orchestrator.verifyTask(task)
		if verifyErr != nil {
			return verifyErr
		}

		orchestrator.backlog.MarkDone(task.ID, verifyOutput)
		orchestrator.announce(topicVerify, verifyOutput)

		return nil
	}

	recon, reconErr := orchestrator.scoutSession.runner.Run(
		orchestrator.ctx,
		scoutSystemPrompt,
		scoutUserPrompt(*task, orchestrator.digest, ""),
	)
	if reconErr != nil {
		return reconErr
	}

	task.Evidence = []string{recon}

	var plan planSlice

	planErr := decodeStructured(
		orchestrator.openai,
		orchestrator.ctx,
		plannerSystemPrompt,
		plannerUserPrompt(*task, orchestrator.digest, recon),
		provider.StructuredOutput{
			Name:        "coding_plan",
			Description: "Atomic replace slice for one task",
			Schema:      planSchema,
			Strict:      false,
		},
		&plan,
	)
	if planErr != nil {
		return planErr
	}

	if strings.TrimSpace(plan.StopReason) != "" && strings.TrimSpace(plan.PrimaryFile) == "" {
		return fmt.Errorf("plan stopped: %s", plan.StopReason)
	}

	if prefixErr := orchestrator.claimTaskLease(task); prefixErr != nil {
		return prefixErr
	}

	defer orchestrator.releaseTaskLease(task)

	_, mutateErr := orchestrator.surgeonSession.runner.Run(
		orchestrator.ctx,
		surgeonSystemPrompt,
		surgeonUserPrompt(*task, plan),
	)
	if mutateErr != nil {
		return mutateErr
	}

	verifyOutput, verifyErr := orchestrator.verifyTask(task)
	if verifyErr != nil {
		return verifyErr
	}

	orchestrator.backlog.MarkDone(task.ID, verifyOutput)
	orchestrator.announce(topicTask, fmt.Sprintf("done %s", task.ID))
	orchestrator.announce(topicVerify, verifyOutput)

	return nil
}

func (orchestrator *Orchestrator) verifyTask(task *Task) (string, error) {
	if len(task.TargetFiles) == 0 {
		return orchestrator.verifier.GoTest()
	}

	return orchestrator.verifier.GoTestPackage(packageFromPath(task.TargetFiles[0]))
}

func (orchestrator *Orchestrator) claimTaskLease(task *Task) error {
	prefix := "lanes/coding/"

	if len(task.TargetFiles) > 0 {
		dir := filepath.Dir(filepath.ToSlash(task.TargetFiles[0]))
		if dir != "." {
			prefix = filepath.ToSlash(dir) + "/"
		}
	}

	return orchestrator.participant.TryClaim(prefix)
}

func (orchestrator *Orchestrator) releaseTaskLease(task *Task) {
	prefix := "lanes/coding/"

	if len(task.TargetFiles) > 0 {
		dir := filepath.Dir(filepath.ToSlash(task.TargetFiles[0]))
		if dir != "." {
			prefix = filepath.ToSlash(dir) + "/"
		}
	}

	_ = orchestrator.participant.Release(prefix)
}

func (orchestrator *Orchestrator) finalAudit() error {
	verifyOutput, verifyErr := orchestrator.verifier.GoTest()
	if verifyErr != nil {
		fmt.Fprintf(os.Stderr, "final verify failed: %v\n%s\n", verifyErr, verifyOutput)
	}

	if orchestrator.config.DryRun || orchestrator.config.Goal == "" {
		return verifyErr
	}

	var verdict auditVerdict

	decodeErr := decodeStructured(
		orchestrator.openai,
		orchestrator.ctx,
		auditorSystemPrompt,
		auditorUserPrompt(orchestrator.config.Goal, orchestrator.backlog, verifyOutput),
		provider.StructuredOutput{
			Name:        "coding_audit",
			Description: "Final audit for goal satisfaction",
			Schema:      auditSchema,
			Strict:      false,
		},
		&verdict,
	)

	if decodeErr != nil {
		return decodeErr
	}

	fmt.Printf("audit: goal_met=%v continue=%v summary=%s\n", verdict.GoalMet, verdict.Continue, verdict.Summary)

	return verifyErr
}

func (orchestrator *Orchestrator) announce(topic, payload string) {
	if orchestrator.participant == nil {
		return
	}

	if err := orchestrator.participant.Announce(topic, payload); err != nil {
		fmt.Fprintf(os.Stderr, "announce %s: %v\n", topic, err)
	}
}
