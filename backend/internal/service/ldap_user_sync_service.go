package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ldapauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const ldapUserSyncWorkers = 8

// LDAPUserLookup exposes the service-account LDAP lookup needed by the sync job.
type LDAPUserLookup interface {
	LookupUser(context.Context, string) (*ldapauth.User, error)
}

// LDAPUserSyncResult summarizes one bounded synchronous synchronization run.
type LDAPUserSyncResult struct {
	Total          int             `json:"total"`
	LDAPCandidates int             `json:"ldap_candidates"`
	Synced         int             `json:"synced"`
	LocalCleared   int             `json:"local_cleared"`
	NotFound       int             `json:"not_found"`
	Failed         int             `json:"failed"`
	DurationMS     int64           `json:"duration_ms"`
	StartedAt      time.Time       `json:"started_at"`
	FinishedAt     time.Time       `json:"finished_at"`
	Errors         []LDAPSyncError `json:"errors,omitempty"`
}

// LDAPSyncError contains a non-sensitive per-user failure summary.
type LDAPSyncError struct {
	Email string `json:"email"`
	Kind  string `json:"kind"`
	Error string `json:"error"`
}

type LDAPUserSyncService struct {
	users             UserRepository
	lookup            LDAPUserLookup
	cacheInvalidator  APIKeyAuthCacheInvalidator
	localLoginAccount map[string]struct{}
}

func NewLDAPUserSyncService(
	users UserRepository,
	lookup LDAPUserLookup,
	cacheInvalidator APIKeyAuthCacheInvalidator,
	localLoginAccounts []string,
) *LDAPUserSyncService {
	local := make(map[string]struct{}, len(localLoginAccounts))
	for _, account := range localLoginAccounts {
		if normalized := normalizeLDAPSyncAccount(account); normalized != "" {
			local[normalized] = struct{}{}
		}
	}
	return &LDAPUserSyncService{
		users:             users,
		lookup:            lookup,
		cacheInvalidator:  cacheInvalidator,
		localLoginAccount: local,
	}
}

func (s *LDAPUserSyncService) Sync(ctx context.Context) (*LDAPUserSyncResult, error) {
	started := time.Now()
	result := &LDAPUserSyncResult{StartedAt: started}
	if s == nil || s.users == nil || s.lookup == nil {
		return nil, errors.New("ldap user sync service is unavailable")
	}

	users, _, err := s.listUsers(ctx)
	if err != nil {
		return nil, err
	}
	result.Total = len(users)

	jobs := make(chan User)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := 0; i < ldapUserSyncWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for user := range jobs {
				s.syncOne(ctx, user, result, &mu)
			}
		}()
	}
	for _, user := range users {
		select {
		case jobs <- user:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, ctx.Err()
		}
	}
	close(jobs)
	wg.Wait()

	result.FinishedAt = time.Now()
	result.DurationMS = result.FinishedAt.Sub(started).Milliseconds()
	return result, nil
}

func (s *LDAPUserSyncService) listUsers(ctx context.Context) ([]User, *pagination.PaginationResult, error) {
	const pageSize = 1000
	var all []User
	page := 1
	var last *pagination.PaginationResult
	for {
		users, pageResult, err := s.users.List(ctx, pagination.PaginationParams{
			Page: page, PageSize: pageSize, SortBy: "id", SortOrder: pagination.SortOrderAsc,
		})
		if err != nil {
			return nil, nil, err
		}
		all = append(all, users...)
		last = pageResult
		if len(users) < pageSize || pageResult == nil || int64(len(all)) >= pageResult.Total {
			return all, last, nil
		}
		page++
	}
}

func (s *LDAPUserSyncService) syncOne(ctx context.Context, user User, result *LDAPUserSyncResult, mu *sync.Mutex) {
	account := normalizeLDAPSyncAccount(user.Email)
	if _, local := s.localLoginAccount[account]; local {
		if user.Department == "" {
			return
		}
		user.Department = ""
		if err := s.users.Update(ctx, &user); err != nil {
			s.recordError(result, mu, user.Email, "local_clear", err)
			return
		}
		if s.cacheInvalidator != nil {
			s.cacheInvalidator.InvalidateAuthCacheByUserID(ctx, user.ID)
		}
		mu.Lock()
		result.LocalCleared++
		mu.Unlock()
		return
	}

	mu.Lock()
	result.LDAPCandidates++
	mu.Unlock()
	identity, err := s.lookup.LookupUser(ctx, user.Email)
	if err != nil {
		kind := "failed"
		if errors.Is(err, ldapauth.ErrDirectoryNotFound) {
			kind = "not_found"
		}
		if kind == "not_found" {
			mu.Lock()
			result.NotFound++
			mu.Unlock()
		} else {
			s.recordError(result, mu, user.Email, kind, err)
		}
		return
	}
	if identity == nil {
		s.recordError(result, mu, user.Email, "failed", errors.New("empty LDAP identity"))
		return
	}

	username := strings.TrimSpace(identity.DisplayName)
	if username == "" {
		username = strings.TrimSpace(identity.Username)
	}
	department := strings.TrimSpace(identity.Department)
	changedDepartment := user.Department != department
	if user.Username != username || changedDepartment {
		user.Username = username
		user.Department = department
		if err := s.users.Update(ctx, &user); err != nil {
			s.recordError(result, mu, user.Email, "update", err)
			return
		}
		if changedDepartment && s.cacheInvalidator != nil {
			s.cacheInvalidator.InvalidateAuthCacheByUserID(ctx, user.ID)
		}
	}
	mu.Lock()
	result.Synced++
	mu.Unlock()
}

func (s *LDAPUserSyncService) recordError(result *LDAPUserSyncResult, mu *sync.Mutex, email, kind string, err error) {
	mu.Lock()
	defer mu.Unlock()
	result.Failed++
	if len(result.Errors) < 50 {
		result.Errors = append(result.Errors, LDAPSyncError{Email: email, Kind: kind, Error: err.Error()})
	}
}

func normalizeLDAPSyncAccount(account string) string {
	return strings.ToLower(strings.TrimSpace(account))
}
