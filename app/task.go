package app

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"time"
)

type Task struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Hint string `json:"hint"`
}

type TaskManager struct {
	tasks []Task
}

func NewTaskManager(r io.Reader) (*TaskManager, error) {
	var tasks []Task
	if err := json.NewDecoder(r).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("decode tasks: %w", err)
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks provided")
	}
	return &TaskManager{tasks: tasks}, nil
}

// GetTaskForDate returns a deterministic task for a given date string (e.g. "2023-10-25").
func (tm *TaskManager) GetTaskForDate(dateStr string) Task {
	h := fnv.New32a()
	h.Write([]byte(dateStr))
	idx := int(h.Sum32()) % len(tm.tasks)
	return tm.tasks[idx]
}

// GetTaskByID finds a task by ID.
func (tm *TaskManager) GetTaskByID(id string) (Task, bool) {
	for _, t := range tm.tasks {
		if t.ID == id {
			return t, true
		}
	}
	return Task{}, false
}

// GetTodayDateStr returns today's date in YYYY-MM-DD format (local time).
func GetTodayDateStr() string {
	return time.Now().Format("2006-01-02")
}
