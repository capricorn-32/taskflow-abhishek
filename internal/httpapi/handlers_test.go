package httpapi

import (
	"net/http/httptest"
	"testing"

	"taskflow/backend/internal/repository"
)

func TestPaginationDefaults(t *testing.T) {
	h := &Handler{defaultPageSize: 20, maxPageSize: 100}
	r := httptest.NewRequest("GET", "/projects", nil)

	page, limit := h.pagination(r)
	if page != 1 {
		t.Fatalf("expected default page 1, got %d", page)
	}
	if limit != 20 {
		t.Fatalf("expected default limit 20, got %d", limit)
	}
}

func TestPaginationClampsLimit(t *testing.T) {
	h := &Handler{defaultPageSize: 20, maxPageSize: 100}
	r := httptest.NewRequest("GET", "/projects?page=2&limit=1000", nil)

	page, limit := h.pagination(r)
	if page != 2 {
		t.Fatalf("expected page 2, got %d", page)
	}
	if limit != 100 {
		t.Fatalf("expected clamped limit 100, got %d", limit)
	}
}

func TestStatusAndPriorityValidation(t *testing.T) {
	if !isValidStatus("todo") || !isValidStatus("in_progress") || !isValidStatus("done") {
		t.Fatal("expected valid statuses to pass")
	}
	if isValidStatus("blocked") {
		t.Fatal("expected invalid status to fail")
	}

	if !isValidPriority(repository.TaskPriorityLow) || !isValidPriority(repository.TaskPriorityMedium) || !isValidPriority(repository.TaskPriorityHigh) {
		t.Fatal("expected valid priorities to pass")
	}
	if isValidPriority(repository.TaskPriority(99)) {
		t.Fatal("expected invalid priority to fail")
	}
}
