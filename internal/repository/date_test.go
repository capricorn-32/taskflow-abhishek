package repository

import "testing"

func TestParseDatePointer(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		v, err := ParseDatePointer("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != nil {
			t.Fatal("expected nil pointer for empty date")
		}
	})

	t.Run("valid date parses", func(t *testing.T) {
		v, err := ParseDatePointer("2026-04-10")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v == nil {
			t.Fatal("expected non-nil date pointer")
		}
		if got := v.Format("2006-01-02"); got != "2026-04-10" {
			t.Fatalf("unexpected parsed date: %s", got)
		}
	})

	t.Run("invalid date fails", func(t *testing.T) {
		v, err := ParseDatePointer("10-04-2026")
		if err == nil {
			t.Fatal("expected parse error for invalid format")
		}
		if v != nil {
			t.Fatal("expected nil date pointer on parse failure")
		}
	})
}
