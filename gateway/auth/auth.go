// Package auth provides email/password authentication for a gin router:
// registration with an emailed verification code, login routes that return
// a JWT, and a middleware that checks it.
package auth

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dashboard/gateway/mailer"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	tokenLifetime = 24 * time.Hour
	userKey       = "auth.user"

	minPasswordLength = 8
	// bcrypt ignores everything after 72 bytes
	maxPasswordLength = 72
)

type Service struct {
	store  Store
	mailer mailer.Mailer
	secret []byte
	now    func() time.Time // replaced in tests to control time
}

func New(store Store, sender mailer.Mailer, jwtSecret string) *Service {
	return &Service{store: store, mailer: sender, secret: []byte(jwtSecret), now: time.Now}
}

// Routes mounts the /auth/... routes on the router or group.
func (s *Service) Routes(r gin.IRouter) {
	group := r.Group("/auth")
	group.POST("/register", s.register)
	group.POST("/verify", s.verify)
	group.POST("/resend-code", s.resendCode)
	group.POST("/login", s.login)
	group.GET("/me", s.RequireAuth(), s.me)
}

type credentials struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func bindCredentials(c *gin.Context) (credentials, bool) {
	var creds credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A valid email and a password are required"})
		return creds, false
	}
	creds.Email = normalizeEmail(creds.Email)
	return creds, true
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// register creates an unverified account and emails it a verification code.
// The user is logged in by /auth/verify, not here.

func (s *Service) register(c *gin.Context) {
	creds, ok := bindCredentials(c)
	if !ok {
		return
	}
	if len(creds.Password) < minPasswordLength || len(creds.Password) > maxPasswordLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be between 8 and 72 characters"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create account"})
		return
	}

	ctx := c.Request.Context()
	// Frees the email if an earlier registration was abandoned
	s.deleteAbandonedUsers(ctx)

	user, err := s.store.CreateUser(ctx, creds.Email, string(hash))
	if errors.Is(err, ErrEmailTaken) {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create account"})
		return
	}

	if err := s.sendVerificationCode(ctx, user); err != nil {
		log.Printf("auth: sending the verification code to user %d: %v", user.ID, err)
		// The registration failed: free the email so the user can try again
		if err := s.store.DeleteUser(ctx, user.ID); err != nil {
			log.Printf("auth: deleting user %d after a failed registration: %v", user.ID, err)
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not send the verification email. Please try again."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"email": user.Email, "verification_required": true})
}

func (s *Service) login(c *gin.Context) {
	creds, ok := bindCredentials(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	// An abandoned registration must answer like an unknown email, not "not verified"
	s.deleteAbandonedUsers(ctx)

	user, err := s.store.FindUserByEmail(ctx, creds.Email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not log in"})
		return
	}
	// Same answer for an unknown email and a wrong password
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(creds.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	// Checked after the password, so it doesn't reveal which emails have an account
	if !user.EmailVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Email not verified"})
		return
	}

	s.respondWithToken(c, http.StatusOK, user)
}

func (s *Service) me(c *gin.Context) {
	c.JSON(http.StatusOK, CurrentUser(c))
}

// RequireAuth rejects requests that carry no valid "Authorization: Bearer <token>" header.
func (s *Service) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !found {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing bearer token"})
			return
		}
		user, err := s.parseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		c.Set(userKey, user)
		c.Next()
	}
}

// CurrentUser returns the user set by RequireAuth.
func CurrentUser(c *gin.Context) User {
	user, _ := c.MustGet(userKey).(User)
	return user
}

type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (s *Service) respondWithToken(c *gin.Context, status int, user User) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenLifetime)),
		},
	}).SignedString(s.secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create token"})
		return
	}
	c.JSON(status, gin.H{"token": token, "user": user})
}

func (s *Service) parseToken(token string) (User, error) {
	var parsed claims
	_, err := jwt.ParseWithClaims(token, &parsed, func(*jwt.Token) (any, error) {
		return s.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return User{}, err
	}
	id, err := strconv.ParseInt(parsed.Subject, 10, 64)
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Email: parsed.Email}, nil
}
