package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dashboard/gateway/auth"
	"dashboard/gateway/internal/github"

	"github.com/gin-gonic/gin"
)

type fakeGithubActivityService struct {
	activity github.Activity
	err      error
}

func (service fakeGithubActivityService) Activity(context.Context, string, time.Time) (github.Activity, error) {
	return service.activity, service.err
}

func TestGithubActivityHandlerReturnsNormalizedActivity(t *testing.T) {
	service := fakeGithubActivityService{activity: github.Activity{
		Repository: "acme/project",
		Commits:    []github.Commit{{SHA: "abc123", Message: "Add widget", Author: "octocat"}},
		Counts:     github.Counts{Today: 2, ThisWeek: 8},
	}}
	router := gin.New()
	router.GET("/api/github/activity", githubActivityHandler(service))

	request := httptest.NewRequest(http.MethodGet, "/api/github/activity?repo=acme/project", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := recorder.Body.String(); body == "" || !strings.Contains(body, `"this_week":8`) || !strings.Contains(body, `"sha":"abc123"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGithubActivityHandlerMapsValidationAndGitHubErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "invalid repository", err: github.ErrInvalidRepository, wantStatus: http.StatusBadRequest},
		{name: "missing repository", err: github.ErrInvalidRepository, wantStatus: http.StatusBadRequest},
		{name: "missing GitHub repository", err: &github.APIError{StatusCode: http.StatusNotFound}, wantStatus: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/api/github/activity", githubActivityHandler(fakeGithubActivityService{err: test.err}))
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/github/activity", nil))
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}

func TestGithubActivityRouteRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")
	registerGithubRoutes(api, auth.New(nil, nil, "test-secret").RequireAuth(), fakeGithubActivityService{})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/github/activity?repo=acme/project", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestGithubActivityHandlerMapsUnexpectedErrorsToBadGateway(t *testing.T) {
	router := gin.New()
	router.GET("/api/github/activity", githubActivityHandler(fakeGithubActivityService{err: errors.New("network failure")}))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/github/activity?repo=acme/project", nil))
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadGateway)
	}
}

func TestGithubActivityHandlerReturnsRateLimitRetryAfter(t *testing.T) {
	router := gin.New()
	router.GET("/api/github/activity", githubActivityHandler(fakeGithubActivityService{err: &github.APIError{
		StatusCode:  http.StatusForbidden,
		RetryAfter:  7 * time.Second,
		RateLimited: true,
	}}))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/github/activity?repo=acme/project", nil))
	if recorder.Code != http.StatusTooManyRequests || !strings.Contains(recorder.Body.String(), `"retry_after":7`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
