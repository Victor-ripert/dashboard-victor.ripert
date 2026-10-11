package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"dashboard/gateway/internal/github"

	"github.com/gin-gonic/gin"
)

type githubActivityService interface {
	Activity(ctx context.Context, repository string, now time.Time) (github.Activity, error)
}

func registerGithubRoutes(api *gin.RouterGroup, requireAuth gin.HandlerFunc, service githubActivityService) {
	api.GET("/github/activity", requireAuth, githubActivityHandler(service))
}

func githubActivityHandler(service githubActivityService) gin.HandlerFunc {
	return func(c *gin.Context) {
		activity, err := service.Activity(c.Request.Context(), c.Query("repo"), time.Now().UTC())
		if errors.Is(err, github.ErrInvalidRepository) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			var apiErr *github.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "GitHub repository not found"})
				return
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": "GitHub service unavailable"})
			return
		}
		c.JSON(http.StatusOK, activity)
	}
}
