package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"taskflow/backend/internal/app"
	"taskflow/backend/internal/config"
)

type testServer struct {
	baseURL string
	close   func()
}

func setupTestServer(t *testing.T) *testServer {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = "postgres://taskflow:taskflow@localhost:5432/taskflow?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("skipping integration test, cannot create DB pool: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("skipping integration test, DB unavailable: %v", err)
	}

	if err := resetDatabase(t, pool); err != nil {
		pool.Close()
		t.Fatalf("failed to reset DB: %v", err)
	}
	pool.Close()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	migrationsPath := "file://" + filepath.Clean(filepath.Join(wd, "..", "..", "migrations"))
	cfg := config.Config{
		HTTPAddr:        ":0",
		DatabaseURL:     dbURL,
		JWTSecret:       "integration-secret",
		JWTIssuer:       "taskflow-test",
		JWTDuration:     24 * time.Hour,
		LogLevel:        slog.LevelError,
		AutoMigrate:     true,
		MigrationsPath:  migrationsPath,
		DefaultPageSize: 20,
		MaxPageSize:     100,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	application, err := app.New(cfg, logger)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}

	ts := httptest.NewServer(application.Router())
	return &testServer{
		baseURL: ts.URL,
		close: func() {
			ts.Close()
			application.Close()
		},
	}
}

func resetDatabase(t *testing.T, pool *pgxpool.Pool) error {
	t.Helper()
	_, err := pool.Exec(context.Background(), `TRUNCATE TABLE tasks, projects, users RESTART IDENTITY CASCADE`)
	return err
}

func postJSON(t *testing.T, url string, payload any, token string) (int, map[string]any) {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	result := map[string]any{}
	if resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	return resp.StatusCode, result
}

func patchJSON(t *testing.T, url string, payload any, token string) (int, map[string]any) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	result := map[string]any{}
	if resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	return resp.StatusCode, result
}

func extractToken(t *testing.T, payload map[string]any) string {
	t.Helper()
	token, ok := payload["token"].(string)
	if !ok || strings.TrimSpace(token) == "" {
		t.Fatalf("token missing from response payload: %#v", payload)
	}
	return token
}

func TestIntegrationRegisterLoginCreateProjectFlow(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	status, registerResp := postJSON(t, ts.baseURL+"/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if status != http.StatusCreated {
		t.Fatalf("expected 201 from register, got %d: %#v", status, registerResp)
	}

	status, loginResp := postJSON(t, ts.baseURL+"/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if status != http.StatusOK {
		t.Fatalf("expected 200 from login, got %d: %#v", status, loginResp)
	}
	aliceToken := extractToken(t, loginResp)

	status, projectResp := postJSON(t, ts.baseURL+"/projects", map[string]any{
		"name":        "Integration Project",
		"description": "Created by integration test",
	}, aliceToken)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 from create project, got %d: %#v", status, projectResp)
	}
	if _, ok := projectResp["id"].(string); !ok {
		t.Fatalf("project id missing in response: %#v", projectResp)
	}
}

func TestIntegrationDuplicateEmailReturnsValidationError(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	status, _ := postJSON(t, ts.baseURL+"/auth/register", map[string]any{
		"name":     "Bob",
		"email":    "bob@example.com",
		"password": "password123",
	}, "")
	if status != http.StatusCreated {
		t.Fatalf("expected first registration to succeed, got %d", status)
	}

	status, dupResp := postJSON(t, ts.baseURL+"/auth/register", map[string]any{
		"name":     "Bob Again",
		"email":    "bob@example.com",
		"password": "password123",
	}, "")
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for duplicate email, got %d: %#v", status, dupResp)
	}
}

func TestIntegrationProjectUpdateForbiddenForNonOwner(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	status, _ := postJSON(t, ts.baseURL+"/auth/register", map[string]any{
		"name":     "Owner",
		"email":    "owner@example.com",
		"password": "password123",
	}, "")
	if status != http.StatusCreated {
		t.Fatalf("expected owner registration to succeed, got %d", status)
	}
	_, ownerLogin := postJSON(t, ts.baseURL+"/auth/login", map[string]any{
		"email":    "owner@example.com",
		"password": "password123",
	}, "")
	ownerToken := extractToken(t, ownerLogin)

	status, projectResp := postJSON(t, ts.baseURL+"/projects", map[string]any{
		"name": "Owner Project",
	}, ownerToken)
	if status != http.StatusCreated {
		t.Fatalf("expected project creation to succeed, got %d: %#v", status, projectResp)
	}
	projectID := projectResp["id"].(string)

	status, _ = postJSON(t, ts.baseURL+"/auth/register", map[string]any{
		"name":     "Other",
		"email":    "other@example.com",
		"password": "password123",
	}, "")
	if status != http.StatusCreated {
		t.Fatalf("expected other registration to succeed, got %d", status)
	}
	_, otherLogin := postJSON(t, ts.baseURL+"/auth/login", map[string]any{
		"email":    "other@example.com",
		"password": "password123",
	}, "")
	otherToken := extractToken(t, otherLogin)

	status, resp := patchJSON(t, ts.baseURL+"/projects/"+projectID, map[string]any{
		"name":        "Hijacked",
		"description": "should be forbidden",
	}, otherToken)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for non-owner project update, got %d: %#v", status, resp)
	}
}
