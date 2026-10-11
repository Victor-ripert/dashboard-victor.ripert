package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

func TestCountCommitsUsesUTCDayAndMondayWeekBoundaries(t *testing.T) {
	now := time.Date(2026, time.March, 4, 3, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	commits := []Commit{
		{Date: time.Date(2026, time.March, 3, 23, 59, 59, 0, time.UTC)},
		{Date: time.Date(2026, time.March, 4, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2026, time.March, 1, 23, 59, 59, 0, time.UTC)},
		{Date: time.Date(2026, time.March, 4, 2, 0, 0, 0, time.UTC)},
	}

	counts := countCommits(commits, now)
	if counts.Today != 1 || counts.ThisWeek != 2 {
		t.Fatalf("counts = %+v, want today=1 this_week=2", counts)
	}
}

func TestCountCommitsIncludesMondayMidnightAndExcludesPreviousWeek(t *testing.T) {
	now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
	commits := []Commit{
		{Date: time.Date(2026, time.March, 9, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2026, time.March, 8, 23, 59, 59, 0, time.UTC)},
	}

	counts := countCommits(commits, now)
	if counts.Today != 1 || counts.ThisWeek != 1 {
		t.Fatalf("counts = %+v, want today=1 this_week=1", counts)
	}
}

func TestActivityReturnsZeroCountsForEmptyRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte("[]"))
	}))
	defer server.Close()

	client := newClient(server.URL, server.Client())
	activity, err := client.Activity(context.Background(), "acme/project", time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Activity returned an error: %v", err)
	}
	if len(activity.Commits) != 0 || activity.Counts != (Counts{}) {
		t.Fatalf("activity = %+v, want empty commits and zero counts", activity)
	}
}

func TestActivityPaginatesUntilResultsEnd(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		page, _ := strconv.Atoi(request.URL.Query().Get("page"))
		response.Header().Set("Content-Type", "application/json")
		if page == 1 {
			payload := make([]string, githubPageSize)
			for index := range payload {
				payload[index] = `{"sha":"commit","commit":{"message":"work","author":{"name":"author","date":"2026-03-09T10:00:00Z"}}}`
			}
			_, _ = response.Write([]byte("[" + strings.Join(payload, ",") + "]"))
			return
		}
		_, _ = response.Write([]byte(`[{"sha":"last","commit":{"message":"last","author":{"name":"author","date":"2026-03-09T11:00:00Z"}}}]`))
	}))
	defer server.Close()

	client := newClient(server.URL, server.Client())
	activity, err := client.Activity(context.Background(), "acme/project", time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Activity returned an error: %v", err)
	}
	if requests != 2 || activity.Counts.ThisWeek != githubPageSize+1 || len(activity.Commits) != recentCommitLimit {
		t.Fatalf("requests=%d activity=%+v, want 2 requests, 101 counted, 10 returned", requests, activity)
	}
}
