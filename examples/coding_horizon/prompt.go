package main

const (
	scoutSystemPrompt = `You are the RECON phase of an AI-native coding loop.

Strengths you must exploit:
- Read and search the workspace exhaustively before proposing changes.
- Cite exact path:line evidence for every claim about existing code.

Weaknesses you must compensate for:
- Do not guess file contents; always read or search first.
- Do not plan edits in this phase; gather evidence only.

Tools allowed: read_file, search.
Output: concise evidence brief with bullet lines formatted as path:line — observation.`

	surgeonSystemPrompt = `You are the MUTATE phase of an AI-native coding loop.

Rules:
- Execute only the approved atomic slice for the active task.
- read_file the primary target before replace.
- Use replace with a unique exact old fragment; never rewrite whole files speculatively.
- If the workspace differs from the plan, stop and explain instead of improvising.

Tools allowed: read_file, search, replace.`

	plannerSystemPrompt = `You are the PLAN phase of an AI-native coding loop.

Design constraints for AI agents:
- One primary file per slice.
- Each slice must be independently verifiable with go test.
- Prefer minimal diffs over rewrites.
- Include exact old_fragment and new_fragment strings for replace when possible.`

	intakeSystemPrompt = `You are the INTAKE phase of an AI-native coding loop.

Decompose the user goal into atomic, independently verifiable goal tasks.
Each task must name concrete target_files and an acceptance criterion provable by tests or inspection.

Do not mimic human team roles. Optimize for machine verification and narrow context windows.`

	auditorSystemPrompt = `You are the AUDIT phase of an AI-native coding loop.

Assume the latest change is wrong until evidence says otherwise.
Judge goal satisfaction conservatively. Count remaining hygiene work.`
)

func scoutUserPrompt(task Task, digest *RepoDigest, recon string) string {
	return "Repo digest:\n" + digestBrief(digest) + "\n\nActive task:\n" + taskBrief(task) + "\n\nProduce an evidence brief for this task only."
}

func plannerUserPrompt(task Task, digest *RepoDigest, recon string) string {
	return "Repo digest:\n" + digestBrief(digest) + "\n\nActive task:\n" + taskBrief(task) + "\n\nRecon evidence:\n" + recon + "\n\nPlan one atomic replace slice."
}

func surgeonUserPrompt(task Task, plan planSlice) string {
	return "Active task:\n" + taskBrief(task) + "\n\nApproved slice:\n" + planBrief(plan) + "\n\nApply the slice using tools. If blocked, explain precisely."
}

func intakeUserPrompt(goal string, digest *RepoDigest) string {
	return "Goal:\n" + goal + "\n\nRepo digest:\n" + digestBrief(digest) + "\n\nEmit goal_tasks only; hygiene is handled separately."
}

func auditorUserPrompt(goal string, backlog *Backlog, verifyOutput string) string {
	return "Goal:\n" + goal + "\n\nBacklog:\n" + backlog.Summary() + "\n\nLatest verify output:\n" + verifyOutput
}

func taskBrief(task Task) string {
	return "id=" + task.ID + " kind=" + string(task.Kind) + " title=" + task.Title + " acceptance=" + task.Acceptance + " targets=" + joinCSV(task.TargetFiles)
}

func planBrief(plan planSlice) string {
	return "primary_file=" + plan.PrimaryFile + " old=" + plan.OldFragment + " new=" + plan.NewFragment + " steps=" + joinCSV(plan.Steps)
}

func joinCSV(values []string) string {
	if len(values) == 0 {
		return ""
	}

	result := values[0]
	for index := 1; index < len(values); index++ {
		result += ", " + values[index]
	}

	return result
}
