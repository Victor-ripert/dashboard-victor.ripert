package proxy

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type GitHubServiceProxy struct {
	baseURL string
	client  *http.Client
}

func (service *GitHubServiceProxy) BaseURL() string { return service.baseURL }

func NewGitHubServiceProxy(baseURL string) *GitHubServiceProxy {
	return &GitHubServiceProxy{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 15 * time.Second,
			// OAuth callbacks redirect to the frontend. The gateway must relay
			// that redirect instead of following it from inside the container.
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func RegisterGitHubRoutes(api *gin.RouterGroup, requireAuth gin.HandlerFunc, service *GitHubServiceProxy) {
	api.GET("/github/activity", requireAuth, service.forward("/widgets/activity"))
	api.GET("/github/connect", requireAuth, service.forward("/oauth/connect"))
	api.GET("/github/callback", service.forward("/oauth/callback"))
	api.DELETE("/github/connection", requireAuth, service.forward("/connection"))
}

func (service *GitHubServiceProxy) forward(path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		target, err := url.Parse(service.baseURL + path)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Invalid GitHub service URL"})
			return
		}
		target.RawQuery = c.Request.URL.RawQuery
		request, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, target.String(), c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Could not reach GitHub service"})
			return
		}
		request.Header.Set("Accept", c.GetHeader("Accept"))
		if authorization := c.GetHeader("Authorization"); authorization != "" {
			request.Header.Set("Authorization", authorization)
		}
		response, err := service.client.Do(request)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "GitHub service unavailable"})
			return
		}
		defer response.Body.Close()
		for key, values := range response.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}
		c.Status(response.StatusCode)
		_, _ = io.Copy(c.Writer, response.Body)
	}
}
