package tache

// TaskTree represents a task in tree structure
type TaskTree struct {
	Task     Task        `json:"task"`
	Subtasks []*TaskTree `json:"subtasks"`
}
