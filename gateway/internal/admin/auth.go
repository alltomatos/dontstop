package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSetupComplete      = errors.New("setup already complete")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

// AdminUser is the console identity.
type AdminUser struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Session is a console session returned once with the bearer token.
type Session struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      AdminUser `json:"user"`
}

// SetupInput creates the first console owner.
type SetupInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginInput authenticates an existing console user.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthService is the PostgreSQL-backed console identity service.
type AuthService struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

// NewAuthService creates the console auth service.
func NewAuthService(pool *pgxpool.Pool) *AuthService {
	return &AuthService{pool: pool, now: time.Now}
}

// NeedsSetup reports whether no console user exists.
func (s *AuthService) NeedsSetup(ctx context.Context) (bool, error) {
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM admin_users`).Scan(&count); err != nil {
		return false, fmt.Errorf("count admin users: %w", err)
	}
	return count == 0, nil
}

// Setup creates the first owner and a session.
func (s *AuthService) Setup(ctx context.Context, input SetupInput) (Session, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Name == "" || input.Email == "" || len(input.Password) < 8 {
		return Session{}, fmt.Errorf("%w: name, email and an 8-character password are required", ErrInvalidInput)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, fmt.Errorf("begin setup: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('dontstop:setup'))`); err != nil {
		return Session{}, fmt.Errorf("lock setup: %w", err)
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM admin_users`).Scan(&count); err != nil {
		return Session{}, fmt.Errorf("count admin users: %w", err)
	}
	if count > 0 {
		return Session{}, ErrSetupComplete
	}

	hash, err := auth.HashKey(input.Password)
	if err != nil {
		return Session{}, fmt.Errorf("hash password: %w", err)
	}
	var user AdminUser
	if err := tx.QueryRow(ctx, `
		INSERT INTO admin_users (tenant_id, name, email, password_hash, role)
		VALUES ('00000000-0000-0000-0000-000000000000', $1, $2, $3, 'owner')
		RETURNING id::text, name, email, role, created_at
	`, input.Name, input.Email, hash).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.CreatedAt); err != nil {
		return Session{}, fmt.Errorf("insert admin user: %w", err)
	}
	session, err := s.createSession(ctx, tx, user)
	if err != nil {
		return Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit setup: %w", err)
	}
	return session, nil
}

// Login verifies an email/password pair and creates a session.
func (s *AuthService) Login(ctx context.Context, input LoginInput) (Session, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	var user AdminUser
	var passwordHash string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, name, email, role, created_at, password_hash
		FROM admin_users
		WHERE email = $1 AND disabled_at IS NULL
		ORDER BY created_at ASC
		LIMIT 1
	`, email).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.CreatedAt, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, fmt.Errorf("load admin user: %w", err)
	}
	if !auth.VerifyKey(passwordHash, input.Password) {
		return Session{}, ErrInvalidCredentials
	}
	if _, err := s.pool.Exec(ctx, `UPDATE admin_users SET last_login_at = now() WHERE id = $1::uuid`, user.ID); err != nil {
		return Session{}, fmt.Errorf("update last login: %w", err)
	}
	return s.createSession(ctx, s.pool, user)
}

// Authenticate resolves a session token to a manage-scoped principal.
func (s *AuthService) Authenticate(ctx context.Context, token string) (auth.Principal, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return auth.Principal{}, ErrUnauthorized
	}
	var principal auth.Principal
	var expiresAt time.Time
	var disabledAt *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT s.id::text, s.tenant_id::text, s.expires_at, u.disabled_at
		FROM admin_sessions s
		JOIN admin_users u ON u.id = s.user_id
		WHERE s.token_hash = $1
	`, hashToken(token)).Scan(&principal.KeyID, &principal.TenantID, &expiresAt, &disabledAt)
	if errors.Is(err, pgx.ErrNoRows) || disabledAt != nil || !s.now().Before(expiresAt) {
		return auth.Principal{}, ErrUnauthorized
	}
	if err != nil {
		return auth.Principal{}, fmt.Errorf("load admin session: %w", err)
	}
	_, _ = s.pool.Exec(ctx, `UPDATE admin_sessions SET last_seen_at = now() WHERE id = $1::uuid`, principal.KeyID)
	principal.Scopes = []string{"manage"}
	return principal, nil
}

// Logout revokes a session token.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM admin_sessions WHERE token_hash = $1`, hashToken(strings.TrimSpace(token)))
	if err != nil {
		return fmt.Errorf("delete admin session: %w", err)
	}
	return nil
}

type sessionQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *AuthService) createSession(ctx context.Context, querier sessionQuerier, user AdminUser) (Session, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Session{}, fmt.Errorf("generate session token: %w", err)
	}
	token := hex.EncodeToString(raw)
	expiresAt := s.now().Add(8 * time.Hour)
	var sessionID string
	if err := querier.QueryRow(ctx, `
		INSERT INTO admin_sessions (tenant_id, user_id, token_hash, expires_at)
		VALUES ('00000000-0000-0000-0000-000000000000', $1::uuid, $2, $3)
		RETURNING id::text
	`, user.ID, hashToken(token), expiresAt).Scan(&sessionID); err != nil {
		return Session{}, fmt.Errorf("insert admin session: %w", err)
	}
	return Session{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
