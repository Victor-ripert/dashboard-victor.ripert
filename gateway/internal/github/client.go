package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidRepository = errors.New("repository must use the owner/repository format")

var repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

const (
	githubPageSize    = 100
	githubMaxPages    = 10
	recentCommitLimit = 10
)

type Commit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Date      time.Time `json:"date"`
	URL       string    `json:"url"`
}

type Counts struct {
	Today    int `json:"today"`
	ThisWeek int `json:"this_week"`
}

type Activity struct {
	Repository string   `json:"repository"`
	Commits    []Commit `json:"commits"`
	Counts     Counts   `json:"counts"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type apiCommit struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Author  *struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"author"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name string    `json:"name"`
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

func NewClient() *Client {
	return &Client{
		baseURL: "https://api.github.com",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func newClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}
}

func (client *Client) RecentCommits(ctx context.Context, repository string) ([]Commit, error) {
	activity, err := client.Activity(ctx, repository, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return activity.Commits, nil
}

func (client *Client) Activity(ctx context.Context, repository string, now time.Time) (Activity, error) {
	if !repositoryPattern.MatchString(repository) {
		return Activity{}, ErrInvalidRepository
	}

	parts := strings.Split(repository, "/")
	now = now.UTC()
	weekStart := startOfWeek(now)
	items, err := client.fetchCommits(ctx, parts[0], parts[1], weekStart)
	if err != nil {
		return Activity{}, err
	}

	counts := countCommits(items, now)
	if len(items) > recentCommitLimit {
		items = items[:recentCommitLimit]
	}
	return Activity{Repository: repository, Commits: items, Counts: counts}, nil
}

func (client *Client) fetchCommits(ctx context.Context, owner, repository string, since time.Time) ([]Commit, error) {
	commits := make([]Commit, 0, 10)
	for page := 1; page <= githubMaxPages; page++ {
		query := url.Values{}
		query.Set("per_page", strconv.Itoa(githubPageSize))
		query.Set("page", strconv.Itoa(page))
		query.Set("since", since.UTC().Format(time.RFC3339))
		endpoint := fmt.Sprintf("%s/repos/%s/%s/commits?%s", client.baseURL, url.PathEscape(owner), url.PathEscape(repository), query.Encode())
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", "application/vnd.github+json")
		request.Header.Set("User-Agent", "dashboard-github-service")

		response, err := client.httpClient.Do(request)
		if err != nil {
			return nil, err
		}
		var payload []apiCommit
		decodeErr := json.NewDecoder(response.Body).Decode(&payload)
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, &APIError{StatusCode: response.StatusCode}
		}
		if decodeErr != nil {
			return nil, decodeErr
		}

		for _, item := range payload {
			commits = append(commits, normalizeCommit(item))
		}
		if len(payload) < githubPageSize {
			break
		}
	}
	return commits, nil
}

func countCommits(commits []Commit, now time.Time) Counts {
	now = now.UTC()
	dayStart := startOfDay(now)
	weekStart := startOfWeek(now)
	counts := Counts{}
	for _, commit := range commits {
		date := commit.Date.UTC()
		if date.Before(weekStart) || date.After(now) {
			continue
		}
		counts.ThisWeek++
		if !date.Before(dayStart) {
			counts.Today++
		}
	}
	return counts
}

func normalizeCommit(item apiCommit) Commit {
	author := item.Commit.Author.Name
	avatarURL := ""
	if item.Author != nil {
		if item.Author.Login != "" {
			author = item.Author.Login
		}
		avatarURL = item.Author.AvatarURL
	}
	return Commit{
		SHA:       item.SHA,
		Message:   item.Commit.Message,
		Author:    author,
		AvatarURL: avatarURL,
		Date:      item.Commit.Author.Date,
		URL:       item.HTMLURL,
	}
}

func startOfDay(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func startOfWeek(value time.Time) time.Time {
	day := startOfDay(value)
	daysSinceMonday := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -daysSinceMonday)
}

type APIError struct {
	StatusCode int
}

func (err *APIError) Error() string {
	return fmt.Sprintf("GitHub API returned status %d", err.StatusCode)
}
