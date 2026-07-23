package routes

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestRBACUserRouteDeclarationCount(t *testing.T) {
	files := []string{
		"user.go",
		"auth.go",
		filepath.Join("..", "..", "handler", "page_handler.go"),
	}
	pattern := regexp.MustCompile(`rbacRoutes\.(GET|POST|PUT|PATCH|DELETE)\(`)
	count := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		count += len(pattern.FindAll(data, -1))
	}
	paymentData, err := os.ReadFile("payment.go")
	if err != nil {
		t.Fatalf("read payment.go: %v", err)
	}
	paymentSelfPattern := regexp.MustCompile(`rbacRoutes\.(GET|POST|PUT|PATCH|DELETE)\([^\r\n]+PermissionPaymentsSelf`)
	count += len(paymentSelfPattern.FindAll(paymentData, -1))
	if count != 64 {
		t.Fatalf("declared page/auth/user RBAC routes = %d, want current source total 64", count)
	}
}
