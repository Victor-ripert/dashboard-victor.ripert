package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	pageSize    = 100
	maxPages    = 10
	commitLimit = 10
)

var repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

var (
	errInvalidRepository = errors.New("repository must use the owner/repository format")
	errNotConnected      = errors.New("GitHub account is not connected")
	errInvalidState      = errors.New("invalid or expired GitHub OAuth state")
)

type GitHubConfig struct{ ClientID, ClientSecret, RedirectURL, FrontendURL, JWTSecret string }
type GitHubService struct {
	config GitHubConfig
	store  *ConnectionStore
	http   *http.Client
	cache  *activityCache
}
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
type apiError struct {
	status      int
	retryAfter  int
	rateLimited bool
}

func (e *apiError) Error() string { return fmt.Sprintf("GitHub API returned status %d", e.status) }

func NewGitHubService(config GitHubConfig, store *ConnectionStore) *GitHubService {
	return &GitHubService{config: config, store: store, http: &http.Client{Timeout: 10 * time.Second}, cache: newActivityCache(time.Minute)}
}

func registerRoutes(router *gin.Engine, service *GitHubService) {
	router.GET("/about.json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"name": "github", "widgets": []gin.H{{"name": "activity", "description": "Recent commits and activity for a repository", "params": []gin.H{{"name": "repo", "type": "string"}}}}})
	})
	router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/widgets/activity", service.requireAuth(), service.activity)
	router.GET("/oauth/connect", service.requireAuth(), service.connect)
	router.GET("/oauth/callback", service.callback)
	router.DELETE("/connection", service.requireAuth(), service.disconnect)
}

func (service *GitHubService) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing bearer token"})
			return
		}
		claims := jwt.RegisteredClaims{}
		parsed, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) { return []byte(service.config.JWTSecret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
		if err != nil || !parsed.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user identity"})
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}
func userID(c *gin.Context) int64 { value, _ := c.Get("user_id"); id, _ := value.(int64); return id }

func (service *GitHubService) activity(c *gin.Context) {
	repo := c.Query("repo")
	token := ""
	if service.store != nil {
		var err error
		token, err = service.store.AccessToken(c, userID(c))
		if errors.Is(err, errNotConnected) {
			token = ""
		} else if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Could not load GitHub connection"})
			return
		}
	}
	activity, err := service.cachedActivity(c.Request.Context(), repo, token)
	if err != nil {
		service.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, activity)
}
func (service *GitHubService) cachedActivity(ctx context.Context, repo, token string) (Activity, error) {
	if !repositoryPattern.MatchString(repo) {
		return Activity{}, errInvalidRepository
	}
	key := repo + ":" + hash(token)
	now := time.Now().UTC()
	service.cache.mu.Lock()
	if item, ok := service.cache.items[key]; ok && now.Before(item.expires) {
		service.cache.mu.Unlock()
		return item.activity, nil
	}
	if call, ok := service.cache.pending[key]; ok {
		service.cache.mu.Unlock()
		<-call.done
		return call.activity, call.err
	}
	call := &activityCall{done: make(chan struct{})}
	service.cache.pending[key] = call
	service.cache.mu.Unlock()
	activity, err := service.fetchActivity(ctx, repo, token, now)
	service.cache.mu.Lock()
	call.activity, call.err = activity, err
	if err == nil {
		service.cache.items[key] = cacheItem{activity: activity, expires: now.Add(service.cache.ttl)}
	}
	delete(service.cache.pending, key)
	close(call.done)
	service.cache.mu.Unlock()
	return activity, err
}
func (service *GitHubService) fetchActivity(ctx context.Context, repo, token string, now time.Time) (Activity, error) {
	parts := strings.Split(repo, "/")
	since := startOfWeek(now)
	all := make([]Commit, 0)
	for page := 1; page <= maxPages; page++ {
		query := url.Values{"per_page": {strconv.Itoa(pageSize)}, "page": {strconv.Itoa(page)}, "since": {since.Format(time.RFC3339)}}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+url.PathEscape(parts[0])+"/"+url.PathEscape(parts[1])+"/commits?"+query.Encode(), nil)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "dashboard-github-service")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res, err := service.http.Do(req)
		if err != nil {
			return Activity{}, err
		}
		var payload []apiCommit
		decodeErr := json.NewDecoder(res.Body).Decode(&payload)
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return Activity{}, &apiError{status: res.StatusCode, retryAfter: retryAfter(res.Header), rateLimited: res.StatusCode == 429 || res.Header.Get("Retry-After") != "" || res.Header.Get("X-RateLimit-Remaining") == "0"}
		}
		if decodeErr != nil {
			return Activity{}, decodeErr
		}
		for _, item := range payload {
			all = append(all, normalize(item))
		}
		if len(payload) < pageSize {
			break
		}
	}
	counts := Counts{}
	for _, commit := range all {
		date := commit.Date.UTC()
		if date.Before(since) || date.After(now) {
			continue
		}
		counts.ThisWeek++
		if !date.Before(startOfDay(now)) {
			counts.Today++
		}
	}
	if len(all) > commitLimit {
		all = all[:commitLimit]
	}
	return Activity{Repository: repo, Commits: all, Counts: counts}, nil
}
func normalize(item apiCommit) Commit {
	author := item.Commit.Author.Name
	avatar := ""
	if item.Author != nil {
		if item.Author.Login != "" {
			author = item.Author.Login
		}
		avatar = item.Author.AvatarURL
	}
	return Commit{SHA: item.SHA, Message: item.Commit.Message, Author: author, AvatarURL: avatar, Date: item.Commit.Author.Date, URL: item.HTMLURL}
}
func startOfDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
func startOfWeek(t time.Time) time.Time {
	d := startOfDay(t)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}
func hash(value string) string { sum := sha256.Sum256([]byte(value)); return fmt.Sprintf("%x", sum[:]) }
func retryAfter(header http.Header) int {
	seconds, _ := strconv.Atoi(header.Get("Retry-After"))
	return seconds
}
func (service *GitHubService) writeError(c *gin.Context, err error) {
	if errors.Is(err, errInvalidRepository) {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		if apiErr.status == http.StatusConflict {
			c.JSON(http.StatusConflict, gin.H{"error": "GitHub repository has no commits yet"})
			return
		}
		if apiErr.status == 401 {
			if service.store != nil {
				_ = service.store.Delete(c, userID(c))
			}
			c.JSON(401, gin.H{"error": "GitHub connection expired or was revoked"})
			return
		}
		if apiErr.status == 404 {
			c.JSON(404, gin.H{"error": "GitHub repository not found"})
			return
		}
		if apiErr.rateLimited {
			body := gin.H{"error": "GitHub rate limit exceeded"}
			if apiErr.retryAfter > 0 {
				body["retry_after"] = apiErr.retryAfter
			}
			c.JSON(429, body)
			return
		}
	}
	c.JSON(502, gin.H{"error": "GitHub service unavailable"})
}

type activityCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	items   map[string]cacheItem
	pending map[string]*activityCall
}
type cacheItem struct {
	activity Activity
	expires  time.Time
}
type activityCall struct {
	done     chan struct{}
	activity Activity
	err      error
}

func newActivityCache(ttl time.Duration) *activityCache {
	return &activityCache{ttl: ttl, items: map[string]cacheItem{}, pending: map[string]*activityCall{}}
}

type jwtClaims struct{ jwt.RegisteredClaims }

func (service *GitHubService) connect(c *gin.Context) {
	if service.store == nil || service.config.ClientID == "" || service.config.ClientSecret == "" {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "GitHub OAuth is not configured"})
		return
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		c.JSON(502, gin.H{"error": "Could not start GitHub authorization"})
		return
	}
	state := base64.RawURLEncoding.EncodeToString(raw)
	if err := service.store.SaveState(c, hash(state), userID(c), time.Now().Add(10*time.Minute)); err != nil {
		c.JSON(502, gin.H{"error": "Could not start GitHub authorization"})
		return
	}
	query := url.Values{"client_id": {service.config.ClientID}, "redirect_uri": {service.config.RedirectURL}, "scope": {"repo"}, "state": {state}}
	c.JSON(200, gin.H{"url": "https://github.com/login/oauth/authorize?" + query.Encode()})
}
func (service *GitHubService) callback(c *gin.Context) {
	state, code := c.Query("state"), c.Query("code")
	result := "error"
	if service.store != nil && state != "" && code != "" {
		if id, err := service.store.ConsumeState(c, hash(state), time.Now()); err == nil {
			if token, err := service.exchange(c, code); err == nil {
				if ghID, err := service.lookup(c, token); err == nil {
					if service.store.Save(c, id, ghID, token) == nil {
						result = "connected"
					}
				}
			}
		}
	}
	target, _ := url.Parse(service.config.FrontendURL)
	q := target.Query()
	q.Set("github", result)
	target.RawQuery = q.Encode()
	c.Redirect(302, target.String())
}
func (service *GitHubService) exchange(c *gin.Context, code string) (string, error) {
	form := url.Values{"client_id": {service.config.ClientID}, "client_secret": {service.config.ClientSecret}, "code": {code}, "redirect_uri": {service.config.RedirectURL}}
	req, _ := http.NewRequestWithContext(c, POST, "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := service.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.New("OAuth exchange failed")
	}
	var body struct {
		AccessToken string `json:"access_token"`
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	return body.AccessToken, err
}
func (service *GitHubService) lookup(c *gin.Context, token string) (string, error) {
	req, _ := http.NewRequestWithContext(c, http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := service.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.New("GitHub user lookup failed")
	}
	var body struct {
		ID int64 `json:"id"`
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	return strconv.FormatInt(body.ID, 10), err
}
func (service *GitHubService) disconnect(c *gin.Context) {
	if service.store == nil {
		c.Status(http.StatusNoContent)
		return
	}
	if err := service.store.Delete(c, userID(c)); err != nil {
		c.JSON(502, gin.H{"error": "Could not disconnect GitHub"})
		return
	}
	c.Status(204)
}

const POST = "POST"

type ConnectionStore struct {
	db  *sql.DB
	gcm cipher.AEAD
}

func NewConnectionStore(ctx context.Context, url string, key []byte) (*ConnectionStore, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	for _, stmt := range []string{`CREATE SCHEMA IF NOT EXISTS github`, `CREATE TABLE IF NOT EXISTS github.oauth_states (state_hash TEXT PRIMARY KEY,user_id BIGINT NOT NULL,expires_at TIMESTAMPTZ NOT NULL)`, `CREATE TABLE IF NOT EXISTS github.connections (user_id BIGINT PRIMARY KEY,github_user_id TEXT NOT NULL,encrypted_access_token BYTEA NOT NULL,nonce BYTEA NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT now(),updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			db.Close()
			return nil, err
		}
	}
	return &ConnectionStore{db: db, gcm: gcm}, nil
}
func (s *ConnectionStore) SaveState(ctx context.Context, h string, id int64, expires time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO github.oauth_states(state_hash,user_id,expires_at) VALUES($1,$2,$3) ON CONFLICT(state_hash) DO UPDATE SET user_id=EXCLUDED.user_id,expires_at=EXCLUDED.expires_at`, h, id, expires)
	return err
}
func (s *ConnectionStore) ConsumeState(ctx context.Context, h string, now time.Time) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `DELETE FROM github.oauth_states WHERE state_hash=$1 AND expires_at>$2 RETURNING user_id`, h, now).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errInvalidState
	}
	return id, err
}
func (s *ConnectionStore) Save(ctx context.Context, id int64, ghID, token string) error {
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	ciphertext := s.gcm.Seal(nil, nonce, []byte(token), nil)
	_, err := s.db.ExecContext(ctx, `INSERT INTO github.connections(user_id,github_user_id,encrypted_access_token,nonce) VALUES($1,$2,$3,$4) ON CONFLICT(user_id) DO UPDATE SET github_user_id=EXCLUDED.github_user_id,encrypted_access_token=EXCLUDED.encrypted_access_token,nonce=EXCLUDED.nonce,updated_at=now()`, id, ghID, ciphertext, nonce)
	return err
}
func (s *ConnectionStore) AccessToken(ctx context.Context, id int64) (string, error) {
	var ciphertext, nonce []byte
	err := s.db.QueryRowContext(ctx, `SELECT encrypted_access_token,nonce FROM github.connections WHERE user_id=$1`, id).Scan(&ciphertext, &nonce)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errNotConnected
	}
	if err != nil {
		return "", err
	}
	plain, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	return string(plain), err
}
func (s *ConnectionStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM github.connections WHERE user_id=$1`, id)
	return err
}
