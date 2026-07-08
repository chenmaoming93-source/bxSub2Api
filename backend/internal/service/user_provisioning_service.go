package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"golang.org/x/crypto/bcrypt"
)

type UserProvisioningTransactionRunner interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type UserProvisioningUserStore interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type UserProvisioningDefaultKeyStore interface {
	EnsureDefaultAPIKey(ctx context.Context, userID int64) (*APIKey, error)
}

type UserProvisioningInput struct {
	Email         string
	Username      string
	SignupSource  string
	Balance       float64
	Concurrency   int
	RPMLimit      int
	PasswordHash  string
	Notes         string
	AllowedGroups []int64
	Role          string
	Status        string
	PostCommit    []UserProvisioningPostCommitAction
}

type UserProvisioningPostCommitAction struct {
	Name string
	Run  func(context.Context, *User) error
}

type UserProvisioningResult struct {
	User             *User
	DefaultAPIKey    *APIKey
	Created          bool
	PostCommitErrors []error
}

type UserProvisioningService struct {
	txRunner UserProvisioningTransactionRunner
	users    UserProvisioningUserStore
	keys     UserProvisioningDefaultKeyStore
	makeHash func() (string, error)
}

type entUserProvisioningTransactionRunner struct{ client *dbent.Client }

func (r entUserProvisioningTransactionRunner) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	if dbent.TxFromContext(ctx) != nil {
		return fn(ctx)
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(dbent.NewTxContext(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit()
}

func NewEntUserProvisioningService(client *dbent.Client, users UserProvisioningUserStore, keys UserProvisioningDefaultKeyStore) *UserProvisioningService {
	if client == nil || users == nil || keys == nil {
		return nil
	}
	return NewUserProvisioningService(entUserProvisioningTransactionRunner{client: client}, users, keys)
}

func NewUserProvisioningService(txRunner UserProvisioningTransactionRunner, users UserProvisioningUserStore, keys UserProvisioningDefaultKeyStore) *UserProvisioningService {
	return &UserProvisioningService{txRunner: txRunner, users: users, keys: keys, makeHash: randomProvisioningPasswordHash}
}

func (s *UserProvisioningService) Provision(ctx context.Context, input UserProvisioningInput) (*UserProvisioningResult, error) {
	if s == nil || s.txRunner == nil || s.users == nil || s.keys == nil {
		return nil, fmt.Errorf("user provisioning dependencies are incomplete")
	}
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if input.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	result := &UserProvisioningResult{}
	err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		user, err := s.users.GetByEmail(txCtx, input.Email)
		if err != nil && !errors.Is(err, ErrUserNotFound) {
			return fmt.Errorf("lookup provisioning user: %w", err)
		}
		if errors.Is(err, ErrUserNotFound) {
			hash := input.PasswordHash
			if hash == "" {
				hash, err = s.makeHash()
				if err != nil {
					return fmt.Errorf("generate provisioning password hash: %w", err)
				}
			}
			user = &User{Email: input.Email, Username: input.Username, PasswordHash: hash, Notes: input.Notes, AllowedGroups: input.AllowedGroups, SignupSource: input.SignupSource, Balance: input.Balance, Concurrency: input.Concurrency, RPMLimit: input.RPMLimit, Role: input.Role, Status: input.Status}
			if user.Role == "" {
				user.Role = RoleUser
			}
			if user.Status == "" {
				user.Status = StatusActive
			}
			if err := s.users.Create(txCtx, user); err != nil {
				if !errors.Is(err, ErrEmailExists) {
					return fmt.Errorf("create provisioning user: %w", err)
				}
				user, err = s.users.GetByEmail(txCtx, input.Email)
				if err != nil {
					return fmt.Errorf("read provisioning user after conflict: %w", err)
				}
			} else {
				result.Created = true
			}
		}
		key, err := s.keys.EnsureDefaultAPIKey(txCtx, user.ID)
		if err != nil {
			return fmt.Errorf("ensure provisioning default api key: %w", err)
		}
		result.User, result.DefaultAPIKey = user, key
		return nil
	})
	if err != nil {
		return nil, err
	}

	if result.Created {
		for _, action := range input.PostCommit {
			if action.Run == nil {
				continue
			}
			if err := action.Run(ctx, result.User); err != nil {
				result.PostCommitErrors = append(result.PostCommitErrors, fmt.Errorf("post-commit action %s: %w", action.Name, err))
			}
		}
	}
	return result, nil
}

func randomProvisioningPasswordHash() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(raw)), bcrypt.DefaultCost)
	return string(hash), err
}
