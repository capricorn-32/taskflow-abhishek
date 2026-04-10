package repository

import (
	"encoding/json"
	"fmt"
	"strings"
)

type TaskPriority uint8

const (
	TaskPriorityLow TaskPriority = iota + 1
	TaskPriorityMedium
	TaskPriorityHigh
)

var taskPriorityNames = map[TaskPriority]string{
	TaskPriorityLow:    "low",
	TaskPriorityMedium: "medium",
	TaskPriorityHigh:   "high",
}

var taskPriorityValues = map[string]TaskPriority{
	"low":    TaskPriorityLow,
	"medium": TaskPriorityMedium,
	"high":   TaskPriorityHigh,
}

func (p TaskPriority) IsValid() bool {
	_, ok := taskPriorityNames[p]
	return ok
}

func (p TaskPriority) String() string {
	name, ok := taskPriorityNames[p]
	if !ok {
		return "unknown"
	}
	return name
}

func (p TaskPriority) GreaterThan(other TaskPriority) bool {
	return p > other
}

func ParseTaskPriority(s string) (TaskPriority, error) {
	p, ok := taskPriorityValues[strings.ToLower(strings.TrimSpace(s))]
	if !ok {
		return 0, fmt.Errorf("invalid priority %q: must be low, medium, or high", s)
	}
	return p, nil
}

func (p TaskPriority) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *TaskPriority) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseTaskPriority(s)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}
