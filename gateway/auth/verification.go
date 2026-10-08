package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dashboard/gateway/mailer"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const (
	codeLifetime    = 10 * time.Minute
	maxCodeAttempts = 5
	resendCooldown  = time.Minute
	// An unverified account is deleted this long after registration, even if new
	// codes keep being requested, so nobody can block an email they don't own.
	maxUnverifiedAge = 30 * time.Minute
	// Time allowed between creating an account and saving its first code
	firstCodeGracePeriod = time.Minute
)

var invalidCode = gin.H{"error": "Invalid or expired code"}

type verifyRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

type resendRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// verify checks the code a new user received by email and logs them in.
func (s *Service) verify(c *gin.Context) {
	var req verifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "An email and a code are required"})
		return
	}
	ctx := c.Request.Context()
	s.deleteAbandonedUsers(ctx)

	user, err := s.store.FindUserByEmail(ctx, normalizeEmail(req.Email))
	if errors.Is(err, ErrUserNotFound) || (err == nil && user.EmailVerified) {
		c.JSON(http.StatusBadRequest, invalidCode)
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not verify the code"})
		return
	}

	code, err := s.store.ConsumeVerificationAttempt(ctx, user.ID, maxCodeAttempts, s.now())
	if errors.Is(err, ErrCodeNotFound) {
		c.JSON(http.StatusBadRequest, invalidCode)
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not verify the code"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(code.CodeHash), []byte(strings.TrimSpace(req.Code))) != nil {
		if code.Attempts >= maxCodeAttempts {
			// Last attempt used: the registration failed, free the email
			if err := s.store.DeleteUser(ctx, user.ID); err != nil {
				log.Printf("auth: deleting user %d after too many wrong codes: %v", user.ID, err)
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "Too many wrong codes. Please register again."})
			return
		}
		c.JSON(http.StatusBadRequest, invalidCode)
		return
	}

	if err := s.store.MarkEmailVerified(ctx, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not verify the code"})
		return
	}
	user.EmailVerified = true
	s.respondWithToken(c, http.StatusOK, user)
}

// resendCode emails a new code to an unverified user. It answers 204 for unknown
// or already verified emails too, so it doesn't reveal which emails have an account.
func (s *Service) resendCode(c *gin.Context) {
	var req resendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A valid email is required"})
		return
	}
	ctx := c.Request.Context()
	s.deleteAbandonedUsers(ctx)

	user, err := s.store.FindUserByEmail(ctx, normalizeEmail(req.Email))
	if errors.Is(err, ErrUserNotFound) || (err == nil && user.EmailVerified) {
		c.Status(http.StatusNoContent)
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not send a new code"})
		return
	}

	last, err := s.store.FindVerificationCode(ctx, user.ID)
	if err != nil && !errors.Is(err, ErrCodeNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not send a new code"})
		return
	}
	if err == nil {
		if wait := resendCooldown - s.now().Sub(last.SentAt); wait > 0 {
			seconds := int(math.Ceil(wait.Seconds()))
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Please wait before requesting a new code", "retry_after": seconds})
			return
		}
	}

	if err := s.sendVerificationCode(ctx, user); err != nil {
		log.Printf("auth: sending a new verification code to user %d: %v", user.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not send the verification email"})
		return
	}
	c.Status(http.StatusNoContent)
}

// sendVerificationCode emails a new code to the user, then saves its hash,
// replacing the previous code.
func (s *Service) sendVerificationCode(ctx context.Context, user User) error {
	code, err := generateCode()
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.mailer.Send(ctx, verificationEmail(user.Email, code)); err != nil {
		return err
	}

	now := s.now()
	return s.store.SaveVerificationCode(ctx, VerificationCode{
		UserID:    user.ID,
		CodeHash:  string(hash),
		ExpiresAt: now.Add(codeLifetime),
		SentAt:    now,
	})
}

// deleteAbandonedUsers frees the emails of registrations that can't be verified
// anymore. A failure is only logged: it must not block the request.
func (s *Service) deleteAbandonedUsers(ctx context.Context) {
	now := s.now()
	err := s.store.DeleteAbandonedUsers(ctx, AbandonedFilter{
		Now:                 now,
		CreatedBefore:       now.Add(-maxUnverifiedAge),
		NoCodeCreatedBefore: now.Add(-firstCodeGracePeriod),
		MaxAttempts:         maxCodeAttempts,
	})
	if err != nil {
		log.Printf("auth: deleting abandoned registrations: %v", err)
	}
}

// generateCode returns a random 6-digit code, "000000" to "999999".
func generateCode() (string, error) {
	// crypto/rand: the code must not be predictable
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func verificationEmail(to, code string) mailer.Message {
	return mailer.Message{
		To:      to,
		Subject: "Your Dashboard verification code",
		Body: fmt.Sprintf("Hello,\n\n"+
			"Your verification code is: %s\n\n"+
			"It expires in %d minutes.\n"+
			"If you didn't create a Dashboard account, you can ignore this email.\n",
			code, int(codeLifetime.Minutes())),
	}
}
