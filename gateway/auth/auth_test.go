package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"dashboard/gateway/mailer"

	"github.com/gin-gonic/gin"
)

// memoryStore is a Store kept in memory, with the same rules as PostgresStore.
type memoryStore struct {
	mu     sync.Mutex
	now    func() time.Time
	users  map[int64]*memoryUser
	codes  map[int64]VerificationCode
	nextID int64
}

type memoryUser struct {
	User
	createdAt time.Time
}

func newMemoryStore(now func() time.Time) *memoryStore {
	return &memoryStore{now: now, users: map[int64]*memoryUser{}, codes: map[int64]VerificationCode{}}
}

func (s *memoryStore) CreateUser(_ context.Context, email, passwordHash string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.Email == email {
			return User{}, ErrEmailTaken
		}
	}
	s.nextID++
	u := &memoryUser{User: User{ID: s.nextID, Email: email, PasswordHash: passwordHash}, createdAt: s.now()}
	s.users[u.ID] = u
	return u.User, nil
}

func (s *memoryStore) FindUserByEmail(_ context.Context, email string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.Email == email {
			return u.User, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (s *memoryStore) DeleteUser(_ context.Context, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, userID)
	delete(s.codes, userID)
	return nil
}

func (s *memoryStore) MarkEmailVerified(_ context.Context, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[userID]; ok {
		u.EmailVerified = true
	}
	delete(s.codes, userID)
	return nil
}

func (s *memoryStore) DeleteAbandonedUsers(_ context.Context, f AbandonedFilter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, u := range s.users {
		if u.EmailVerified {
			continue
		}
		code, hasCode := s.codes[id]
		usable := hasCode && code.ExpiresAt.After(f.Now) && code.Attempts < f.MaxAttempts
		if !u.createdAt.After(f.CreatedBefore) || (!u.createdAt.After(f.NoCodeCreatedBefore) && !usable) {
			delete(s.users, id)
			delete(s.codes, id)
		}
	}
	return nil
}

func (s *memoryStore) SaveVerificationCode(_ context.Context, code VerificationCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[code.UserID]; !ok {
		return errors.New("user does not exist")
	}
	code.Attempts = 0
	s.codes[code.UserID] = code
	return nil
}

func (s *memoryStore) FindVerificationCode(_ context.Context, userID int64) (VerificationCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	code, ok := s.codes[userID]
	if !ok {
		return VerificationCode{}, ErrCodeNotFound
	}
	return code, nil
}

func (s *memoryStore) ConsumeVerificationAttempt(_ context.Context, userID int64, maxAttempts int, now time.Time) (VerificationCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	code, ok := s.codes[userID]
	if !ok || code.Attempts >= maxAttempts || !code.ExpiresAt.After(now) {
		return VerificationCode{}, ErrCodeNotFound
	}
	code.Attempts++
	s.codes[userID] = code
	return code, nil
}

// fakeMailer records the emails instead of sending them.
type fakeMailer struct {
	sent []mailer.Message
	fail bool
}

func (m *fakeMailer) Send(_ context.Context, msg mailer.Message) error {
	if m.fail {
		return errors.New("smtp server down")
	}
	m.sent = append(m.sent, msg)
	return nil
}

type testEnv struct {
	t      *testing.T
	router *gin.Engine
	store  *memoryStore
	mail   *fakeMailer
	clock  time.Time
}

func newTestEnv(t *testing.T) *testEnv {
	gin.SetMode(gin.TestMode)
	env := &testEnv{t: t, mail: &fakeMailer{}, clock: time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)}
	now := func() time.Time { return env.clock }
	env.store = newMemoryStore(now)

	service := New(env.store, env.mail, "test-secret")
	service.now = now
	env.router = gin.New()
	service.Routes(env.router)
	return env
}

func (e *testEnv) advance(d time.Duration) { e.clock = e.clock.Add(d) }

func (e *testEnv) request(method, path string, body any, token string) (int, map[string]any) {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)

	var response map[string]any
	json.Unmarshal(rec.Body.Bytes(), &response)
	return rec.Code, response
}

func (e *testEnv) post(path string, body any) (int, map[string]any) {
	e.t.Helper()
	return e.request(http.MethodPost, path, body, "")
}

var sixDigits = regexp.MustCompile(`\b\d{6}\b`)

// lastCode returns the code in the last email sent to the address.
func (e *testEnv) lastCode(email string) string {
	e.t.Helper()
	for i := len(e.mail.sent) - 1; i >= 0; i-- {
		if e.mail.sent[i].To == email {
			return sixDigits.FindString(e.mail.sent[i].Body)
		}
	}
	e.t.Fatalf("no email sent to %s", email)
	return ""
}

func (e *testEnv) expect(t *testing.T, gotStatus, wantStatus int, response map[string]any) {
	t.Helper()
	if gotStatus != wantStatus {
		t.Fatalf("status = %d, want %d (response %v)", gotStatus, wantStatus, response)
	}
}

const (
	email    = "victor@example.com"
	password = "password123"
)

func wrongCode(code string) string {
	if code == "000000" {
		return "111111"
	}
	return "000000"
}

func TestRegisterThenVerifyLogsIn(t *testing.T) {
	env := newTestEnv(t)

	status, res := env.post("/auth/register", gin.H{"email": "Victor@Example.com", "password": password})
	env.expect(t, status, http.StatusCreated, res)
	if res["verification_required"] != true || res["email"] != email || res["token"] != nil {
		t.Fatalf("register response = %v", res)
	}
	code := env.lastCode(email)
	if len(code) != 6 {
		t.Fatalf("no 6-digit code in the email: %q", env.mail.sent[0].Body)
	}

	status, res = env.post("/auth/login", gin.H{"email": email, "password": password})
	env.expect(t, status, http.StatusForbidden, res)

	status, res = env.post("/auth/verify", gin.H{"email": email, "code": wrongCode(code)})
	env.expect(t, status, http.StatusBadRequest, res)

	status, res = env.post("/auth/verify", gin.H{"email": email, "code": code})
	env.expect(t, status, http.StatusOK, res)
	token, _ := res["token"].(string)
	if token == "" {
		t.Fatalf("verify returned no token: %v", res)
	}

	status, res = env.request(http.MethodGet, "/auth/me", nil, token)
	env.expect(t, status, http.StatusOK, res)

	status, res = env.post("/auth/login", gin.H{"email": email, "password": password})
	env.expect(t, status, http.StatusOK, res)

	// The code can't be used twice
	status, res = env.post("/auth/verify", gin.H{"email": email, "code": code})
	env.expect(t, status, http.StatusBadRequest, res)
}

func TestRegisterRefusesUsedEmail(t *testing.T) {
	env := newTestEnv(t)
	env.post("/auth/register", gin.H{"email": email, "password": password})

	// Pending verification
	status, res := env.post("/auth/register", gin.H{"email": email, "password": "attacker-password"})
	env.expect(t, status, http.StatusConflict, res)

	// Verified
	env.post("/auth/verify", gin.H{"email": email, "code": env.lastCode(email)})
	status, res = env.post("/auth/register", gin.H{"email": email, "password": password})
	env.expect(t, status, http.StatusConflict, res)
}

func TestTooManyWrongCodesDeletesAccount(t *testing.T) {
	env := newTestEnv(t)
	env.post("/auth/register", gin.H{"email": email, "password": password})
	code := env.lastCode(email)

	for i := 1; i < maxCodeAttempts; i++ {
		status, res := env.post("/auth/verify", gin.H{"email": email, "code": wrongCode(code)})
		env.expect(t, status, http.StatusBadRequest, res)
	}
	status, res := env.post("/auth/verify", gin.H{"email": email, "code": wrongCode(code)})
	env.expect(t, status, http.StatusBadRequest, res)
	if res["error"] != "Too many wrong codes. Please register again." {
		t.Errorf("last attempt error = %v", res["error"])
	}

	// The right code doesn't work anymore, and the email is free again
	status, res = env.post("/auth/verify", gin.H{"email": email, "code": code})
	env.expect(t, status, http.StatusBadRequest, res)
	status, res = env.post("/auth/register", gin.H{"email": email, "password": password})
	env.expect(t, status, http.StatusCreated, res)
}

func TestExpiredCodeFreesEmail(t *testing.T) {
	env := newTestEnv(t)
	env.post("/auth/register", gin.H{"email": email, "password": password})
	code := env.lastCode(email)

	env.advance(codeLifetime + time.Second)
	status, res := env.post("/auth/verify", gin.H{"email": email, "code": code})
	env.expect(t, status, http.StatusBadRequest, res)

	status, res = env.post("/auth/login", gin.H{"email": email, "password": password})
	env.expect(t, status, http.StatusUnauthorized, res)

	status, res = env.post("/auth/register", gin.H{"email": email, "password": password})
	env.expect(t, status, http.StatusCreated, res)
}

func TestResendCode(t *testing.T) {
	env := newTestEnv(t)
	env.post("/auth/register", gin.H{"email": email, "password": password})
	firstCode := env.lastCode(email)

	status, res := env.post("/auth/resend-code", gin.H{"email": email})
	env.expect(t, status, http.StatusTooManyRequests, res)
	if res["retry_after"] != float64(60) {
		t.Errorf("retry_after = %v, want 60", res["retry_after"])
	}

	env.advance(resendCooldown + time.Second)
	status, res = env.post("/auth/resend-code", gin.H{"email": email})
	env.expect(t, status, http.StatusNoContent, res)
	if len(env.mail.sent) != 2 {
		t.Fatalf("emails sent = %d, want 2", len(env.mail.sent))
	}
	newCode := env.lastCode(email)

	if firstCode != newCode {
		status, res = env.post("/auth/verify", gin.H{"email": email, "code": firstCode})
		env.expect(t, status, http.StatusBadRequest, res)
	}
	status, res = env.post("/auth/verify", gin.H{"email": email, "code": newCode})
	env.expect(t, status, http.StatusOK, res)
}

func TestResendCodeDoesNotRevealAccounts(t *testing.T) {
	env := newTestEnv(t)

	status, res := env.post("/auth/resend-code", gin.H{"email": "nobody@example.com"})
	env.expect(t, status, http.StatusNoContent, res)

	env.post("/auth/register", gin.H{"email": email, "password": password})
	env.post("/auth/verify", gin.H{"email": email, "code": env.lastCode(email)})
	env.advance(resendCooldown + time.Second)
	status, res = env.post("/auth/resend-code", gin.H{"email": email})
	env.expect(t, status, http.StatusNoContent, res)

	if len(env.mail.sent) != 1 {
		t.Errorf("emails sent = %d, want only the registration one", len(env.mail.sent))
	}
}

func TestUnverifiedAccountHasMaximumAge(t *testing.T) {
	env := newTestEnv(t)
	env.post("/auth/register", gin.H{"email": email, "password": password})

	// Keep asking for new codes: the account still disappears after maxUnverifiedAge
	for elapsed := time.Duration(0); elapsed+9*time.Minute < maxUnverifiedAge; elapsed += 9 * time.Minute {
		env.advance(9 * time.Minute)
		status, res := env.post("/auth/resend-code", gin.H{"email": email})
		env.expect(t, status, http.StatusNoContent, res)
	}
	code := env.lastCode(email)
	env.advance(maxUnverifiedAge)

	status, res := env.post("/auth/verify", gin.H{"email": email, "code": code})
	env.expect(t, status, http.StatusBadRequest, res)
	status, res = env.post("/auth/register", gin.H{"email": email, "password": password})
	env.expect(t, status, http.StatusCreated, res)
}

func TestEmailFailureFreesEmail(t *testing.T) {
	env := newTestEnv(t)
	env.mail.fail = true

	status, res := env.post("/auth/register", gin.H{"email": email, "password": password})
	env.expect(t, status, http.StatusBadGateway, res)

	env.mail.fail = false
	status, res = env.post("/auth/register", gin.H{"email": email, "password": password})
	env.expect(t, status, http.StatusCreated, res)
}

func TestVerifiedUserFromBeforeTheFeatureCanLogIn(t *testing.T) {
	env := newTestEnv(t)
	// Simulates an account the migration marked as verified
	env.post("/auth/register", gin.H{"email": email, "password": password})
	env.store.MarkEmailVerified(context.Background(), 1)
	env.advance(maxUnverifiedAge + time.Hour)

	status, res := env.post("/auth/login", gin.H{"email": email, "password": password})
	env.expect(t, status, http.StatusOK, res)
}
