package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRecentCommitsNormalizesGitHubResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/repos/acme/project/commits" || request.URL.Query().Get("per_page") != "100" || request.URL.Query().Get("page") != "1" {
			t.Fatalf("unexpected request: %s", request.URL.String())
		}
		if request.Header.Get("User-Agent") == "" {
			t.Fatal("expected User-Agent header")
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`[{"sha":"abc123","html_url":"https://github.com/acme/project/commit/abc123","author":{"login":"octocat","avatar_url":"https://avatars.example/octocat"},"commit":{"message":"Add widget","author":{"name":"Octo Cat","date":"2026-10-08T09:30:00Z"}}}]`))
	}))
	defer server.Close()

	client := newClient(server.URL, server.Client())
	commits, err := client.RecentCommits(context.Background(), "acme/project")
	if err != nil {
		t.Fatalf("RecentCommits returned an error: %v", err)
	}
	if len(commits) != 1 || commits[0].Author != "octocat" || commits[0].SHA != "abc123" {
		t.Fatalf("unexpected normalized commit: %+v", commits)
	}
	if !commits[0].Date.Equal(time.Date(2026, 10, 8, 9, 30, 0, 0, time.UTC)) {
		t.Fatalf("unexpected commit date: %s", commits[0].Date)
	}
}

func TestRecentCommitsRejectsInvalidRepository(t *testing.T) {
	client := newClient("https://example.test", http.DefaultClient)
	_, err := client.RecentCommits(context.Background(), "https://github.com/acme/project")
	if !errors.Is(err, ErrInvalidRepository) {
		t.Fatalf("expected ErrInvalidRepository, got %v", err)
	}
}

func TestRecentCommitsReturnsGitHubStatus(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	client := newClient(server.URL, server.Client())
	_, err := client.RecentCommits(context.Background(), "acme/project")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected GitHub 404 error, got %v", err)
	}
}
