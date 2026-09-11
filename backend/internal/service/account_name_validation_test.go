package service

import (
	"context"
	"errors"
	"testing"
)

type accountNameValidationRepoStub struct {
	AccountRepository
	exists    bool
	existsErr error
	account   *Account
	created   bool
	updated   bool
}

func (r *accountNameValidationRepoStub) ExistsByName(context.Context, string, int64) (bool, error) {
	return r.exists, r.existsErr
}
func (r *accountNameValidationRepoStub) Create(_ context.Context, account *Account) error {
	r.created = true
	account.ID = 1
	r.account = account
	return nil
}
func (r *accountNameValidationRepoStub) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}
func (r *accountNameValidationRepoStub) Update(_ context.Context, account *Account) error {
	r.updated = true
	r.account = account
	return nil
}

func TestAccountServiceCreateRejectsDuplicateName(t *testing.T) {
	repo := &accountNameValidationRepoStub{exists: true}
	service := NewAccountService(repo, nil)
	_, err := service.Create(context.Background(), CreateAccountRequest{Name: "duplicate"})
	if !errors.Is(err, ErrAccountNameExists) {
		t.Fatalf("expected duplicate name error, got %v", err)
	}
	if repo.created {
		t.Fatal("duplicate account must not be created")
	}
}

func TestAccountServiceCreateAndUpdateNormalizeAndCheckName(t *testing.T) {
	repo := &accountNameValidationRepoStub{}
	service := NewAccountService(repo, nil)
	created, err := service.Create(context.Background(), CreateAccountRequest{Name: "  new-account  "})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if created.Name != "new-account" {
		t.Fatalf("expected trimmed name, got %q", created.Name)
	}

	duplicate := "other-account"
	repo.exists = true
	if _, err := service.Update(context.Background(), created.ID, UpdateAccountRequest{Name: &duplicate}); !errors.Is(err, ErrAccountNameExists) {
		t.Fatalf("expected duplicate update error, got %v", err)
	}
	if repo.updated {
		t.Fatal("duplicate account name must not be updated")
	}

	repo.exists = false
	updatedName := "  renamed-account  "
	updated, err := service.Update(context.Background(), created.ID, UpdateAccountRequest{Name: &updatedName})
	if err != nil {
		t.Fatalf("update account: %v", err)
	}
	if updated.Name != "renamed-account" {
		t.Fatalf("expected trimmed updated name, got %q", updated.Name)
	}
}
