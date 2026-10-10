package app

import (
	"strings"
	"testing"
)

func TestTaskManager_DeterministicPick(t *testing.T) {
	jsonTasks := `[
		{"id": "t1", "text": "1"},
		{"id": "t2", "text": "2"},
		{"id": "t3", "text": "3"}
	]`
	tm, err := NewTaskManager(strings.NewReader(jsonTasks))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task1 := tm.GetTaskForDate("2023-10-25")
	task2 := tm.GetTaskForDate("2023-10-25")
	if task1.ID != task2.ID {
		t.Errorf("expected deterministic task selection, got %s and %s", task1.ID, task2.ID)
	}

	task3 := tm.GetTaskForDate("2023-10-26")
	if task1.ID == task3.ID {
		// there is a small chance of collision, but for 3 items and specific dates, let's just make sure it doesn't panic.
		t.Logf("collision on date change, %s == %s", task1.ID, task3.ID)
	}
}

func TestTaskManager_Empty(t *testing.T) {
	_, err := NewTaskManager(strings.NewReader(`[]`))
	if err == nil {
		t.Errorf("expected error for empty tasks")
	}
}
