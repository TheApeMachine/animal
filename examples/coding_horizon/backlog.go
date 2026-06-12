package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type taskKind string

const (
	taskKindGoal    taskKind = "goal"
	taskKindHygiene taskKind = "hygiene"
)

type taskStatus string

const (
	taskStatusPending taskStatus = "pending"
	taskStatusActive  taskStatus = "active"
	taskStatusDone    taskStatus = "done"
	taskStatusBlocked taskStatus = "blocked"
)

/*
Task is one atomic unit of work the horizon loop can prove independently.
*/
type Task struct {
	ID           string     `json:"id"`
	Kind         taskKind   `json:"kind"`
	Title        string     `json:"title"`
	Rationale    string     `json:"rationale"`
	TargetFiles  []string   `json:"target_files"`
	Acceptance   string     `json:"acceptance"`
	Status       taskStatus `json:"status"`
	Evidence     []string   `json:"evidence,omitempty"`
	VerifyOutput string     `json:"verify_output,omitempty"`
}

/*
Backlog orders goal work ahead of hygiene work and tracks completion.
*/
type Backlog struct {
	goal      string
	tasks     []Task
	nextIndex int
}

func newBacklog(goal string) *Backlog {
	return &Backlog{
		goal:  strings.TrimSpace(goal),
		tasks: make([]Task, 0),
	}
}

func (backlog *Backlog) Add(task Task) {
	if task.Status == "" {
		task.Status = taskStatusPending
	}

	backlog.tasks = append(backlog.tasks, task)
}

func (backlog *Backlog) Merge(tasks []Task) {
	for _, task := range tasks {
		if backlog.hasID(task.ID) {
			continue
		}

		backlog.Add(task)
	}
}

func (backlog *Backlog) hasID(id string) bool {
	for _, task := range backlog.tasks {
		if task.ID == id {
			return true
		}
	}

	return false
}

func (backlog *Backlog) Next() (*Task, bool) {
	for index := range backlog.tasks {
		if backlog.tasks[index].Status != taskStatusPending {
			continue
		}

		if backlog.tasks[index].Kind == taskKindGoal {
			backlog.tasks[index].Status = taskStatusActive

			return &backlog.tasks[index], true
		}
	}

	for index := range backlog.tasks {
		if backlog.tasks[index].Status != taskStatusPending {
			continue
		}

		backlog.tasks[index].Status = taskStatusActive

		return &backlog.tasks[index], true
	}

	return nil, false
}

func (backlog *Backlog) MarkDone(taskID string, verifyOutput string) {
	for index := range backlog.tasks {
		if backlog.tasks[index].ID != taskID {
			continue
		}

		backlog.tasks[index].Status = taskStatusDone
		backlog.tasks[index].VerifyOutput = verifyOutput

		return
	}
}

func (backlog *Backlog) MarkBlocked(taskID string, reason string) {
	for index := range backlog.tasks {
		if backlog.tasks[index].ID != taskID {
			continue
		}

		backlog.tasks[index].Status = taskStatusBlocked
		backlog.tasks[index].VerifyOutput = reason

		return
	}
}

func (backlog *Backlog) Goal() string {
	return backlog.goal
}

func (backlog *Backlog) GoalSatisfied() bool {
	if backlog.goal == "" {
		return true
	}

	for _, task := range backlog.tasks {
		if task.Kind != taskKindGoal {
			continue
		}

		if task.Status != taskStatusDone {
			return false
		}
	}

	return backlog.hasGoalTasks()
}

func (backlog *Backlog) hasGoalTasks() bool {
	for _, task := range backlog.tasks {
		if task.Kind == taskKindGoal {
			return true
		}
	}

	return false
}

func (backlog *Backlog) HygieneEmpty() bool {
	for _, task := range backlog.tasks {
		if task.Kind != taskKindHygiene {
			continue
		}

		if task.Status == taskStatusPending || task.Status == taskStatusActive {
			return false
		}
	}

	return true
}

func (backlog *Backlog) PendingCount() int {
	count := 0

	for _, task := range backlog.tasks {
		if task.Status == taskStatusPending {
			count++
		}
	}

	return count
}

func (backlog *Backlog) Summary() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("goal=%q pending=%d\n", backlog.goal, backlog.PendingCount()))

	for _, task := range backlog.tasks {
		builder.WriteString(fmt.Sprintf("- [%s] %s (%s): %s\n", task.Status, task.ID, task.Kind, task.Title))
	}

	return builder.String()
}

func (backlog *Backlog) JSON() string {
	payload, err := json.Marshal(backlog.tasks)
	if err != nil {
		return "[]"
	}

	return string(payload)
}
