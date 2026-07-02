package repository

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestDeletedUserLoginIdentifier(t *testing.T) {
	t.Run("preserves readable original", func(t *testing.T) {
		require.Equal(t, "__deleted_user_42__person@example.com", deletedUserLoginIdentifier(42, "person@example.com"))
	})

	t.Run("truncates to database character limit", func(t *testing.T) {
		got := deletedUserLoginIdentifier(42, strings.Repeat("界", 300))
		require.Equal(t, 255, utf8.RuneCountInString(got))
		require.True(t, strings.HasPrefix(got, "__deleted_user_42__"))
	})
}

func TestIsDeletedUserLoginIdentifier(t *testing.T) {
	require.True(t, isDeletedUserLoginIdentifier("  __DELETED_USER_42__old@example.com"))
	require.False(t, isDeletedUserLoginIdentifier("person@example.com"))
}
