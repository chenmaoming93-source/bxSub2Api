package ldapauth

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/require"
)

func TestUserFromEntryUsesConfiguredAttributes(t *testing.T) {
	client := New(config.LDAPConfig{
		UsernameAttribute:    "uid",
		EmailAttribute:       "mail",
		DisplayNameAttribute: "cn",
	})
	entry := &ldap.Entry{
		DN: "uid=zhangsan,ou=people,dc=example,dc=com",
		Attributes: []*ldap.EntryAttribute{
			{Name: "uid", Values: []string{"zhangsan"}},
			{Name: "mail", Values: []string{"ZhangSan@Example.com"}},
			{Name: "cn", Values: []string{"张三"}},
		},
	}

	user := client.userFromEntry(entry, "fallback")
	require.Equal(t, "zhangsan", user.Username)
	require.Equal(t, "zhangsan@example.com", user.Email)
	require.Equal(t, "张三", user.DisplayName)
	require.Equal(t, entry.DN, user.DN)
}
