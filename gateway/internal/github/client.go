package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var ErrInvalidRepository = errors.New("repository must use the owner/repository format")

var repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type Commit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Date      time.Time `json:"date"`
	URL       string    `json:"url"`
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
	if !repositoryPattern.MatchString(repository) {
		return nil, ErrInvalidRepository
	}

	parts := strings.Split(repository, "/")
	endpoint := fmt.Sprintf("%s/repos/%s/%s/commits?per_page=10", client.baseURL, url.PathEscape(parts[0]), url.PathEscape(parts[1]))
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
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: response.StatusCode}
	}

	var payload []apiCommit
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}

	commits := make([]Commit, 0, len(payload))
	for _, item := range payload {
		author := item.Commit.Author.Name
		avatarURL := ""
		if item.Author != nil {
			if item.Author.Login != "" {
				author = item.Author.Login
			}
			avatarURL = item.Author.AvatarURL
		}
		commits = append(commits, Commit{
			SHA:       item.SHA,
			Message:   item.Commit.Message,
			Author:    author,
			AvatarURL: avatarURL,
			Date:      item.Commit.Author.Date,
			URL:       item.HTMLURL,
		})
	}
	return commits, nil
}

type APIError struct {
	StatusCode int
}

func (err *APIError) Error() string {
	return fmt.Sprintf("GitHub API returned status %d", err.StatusCode)
}
