package domain_test

import (
	"testing"
	"time"

	"github.com/5000K/5000mails/domain"
)

func TestUser_IsConfirmed(t *testing.T) {
	t.Run("nil ConfirmedAt returns false", func(t *testing.T) {
		u := domain.User{ConfirmedAt: nil}
		if u.IsConfirmed() {
			t.Error("expected IsConfirmed() = false for nil ConfirmedAt")
		}
	})

	t.Run("non-nil ConfirmedAt returns true", func(t *testing.T) {
		now := time.Now()
		u := domain.User{ConfirmedAt: &now}
		if !u.IsConfirmed() {
			t.Error("expected IsConfirmed() = true for non-nil ConfirmedAt")
		}
	})
}
