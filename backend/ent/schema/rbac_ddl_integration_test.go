package schema

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/go-sql-driver/mysql"
)

func TestRBACDDLOnEmptyDatabase(t *testing.T) {
	if os.Getenv("RBAC_VERIFY_DDL") != "1" {
		t.Skip("set RBAC_VERIFY_DDL=1 to verify against the configured MySQL/GoldenDB server")
	}
	cfg, err := config.LoadForBootstrap()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	serverConfig := mysql.NewConfig()
	serverConfig.User = cfg.Database.User
	serverConfig.Passwd = cfg.Database.Password
	serverConfig.Net = "tcp"
	serverConfig.Addr = fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port)
	serverConfig.ParseTime = true
	server, err := sql.Open("mysql", serverConfig.FormatDSN())
	if err != nil {
		t.Fatalf("open database server: %v", err)
	}
	defer server.Close()

	databaseName := fmt.Sprintf("rbac_ddl_verify_%d", time.Now().UnixNano())
	if _, err := server.Exec("CREATE DATABASE `" + databaseName + "`"); err != nil {
		t.Fatalf("create isolated verification database: %v", err)
	}
	defer func() {
		if _, dropErr := server.Exec("DROP DATABASE IF EXISTS `" + databaseName + "`"); dropErr != nil {
			t.Errorf("drop isolated verification database: %v", dropErr)
		}
	}()

	databaseConfig := *serverConfig
	databaseConfig.DBName = databaseName
	db, err := sql.Open("mysql", databaseConfig.FormatDSN())
	if err != nil {
		t.Fatalf("open isolated database: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE users (id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, role VARCHAR(20) NOT NULL)"); err != nil {
		t.Fatalf("create referenced users table: %v", err)
	}
	if _, err := db.Exec("INSERT INTO users (role) VALUES ('admin'), ('user')"); err != nil {
		t.Fatalf("insert compatibility fixtures: %v", err)
	}

	for pass := 1; pass <= 2; pass++ {
		if err := executeSQLFile(db, filepath.Join("..", "..", "sqlArchiving", "162_create_rbac_schema.sql")); err != nil {
			t.Fatalf("execute DDL pass %d: %v", pass, err)
		}
	}

	var tableCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = ? AND table_name LIKE 'rbac_%'
	`, databaseName).Scan(&tableCount)
	if err != nil {
		t.Fatalf("inspect tables: %v", err)
	}
	if tableCount != 7 {
		t.Fatalf("expected 7 RBAC tables, got %d", tableCount)
	}

	for pass := 1; pass <= 2; pass++ {
		if err := executeSQLFile(db, filepath.Join("..", "..", "sqlArchiving", "163_seed_rbac_compatibility.sql")); err != nil {
			t.Fatalf("execute compatibility seed pass %d: %v", pass, err)
		}
	}
	assertCount(t, db, "SELECT COUNT(*) FROM rbac_permissions", 101)
	assertCount(t, db, "SELECT COUNT(*) FROM rbac_roles", 2)
	assertCount(t, db, "SELECT COUNT(*) FROM rbac_user_roles", 2)
	assertCount(t, db, "SELECT COUNT(*) FROM rbac_user_versions", 2)
	assertCount(t, db, `
		SELECT COUNT(*) FROM rbac_role_permissions rp
		JOIN rbac_roles r ON r.id = rp.role_id AND r.code = 'admin'
		JOIN rbac_permissions p ON p.id = rp.permission_id AND p.code = '*'
	`, 1)
	assertCount(t, db, `
		SELECT COUNT(*) FROM rbac_role_permissions rp
		JOIN rbac_roles r ON r.id = rp.role_id AND r.code = 'admin'
	`, 1)
	assertCount(t, db, `
		SELECT COUNT(*)
		FROM rbac_permissions p
		WHERE p.module = 'self'
		  AND NOT EXISTS (
		      SELECT 1
		      FROM rbac_role_permissions rp
		      JOIN rbac_roles r ON r.id = rp.role_id AND r.code = 'user'
		      WHERE rp.permission_id = p.id
		  )
	`, 0)

	if _, err := db.Exec("INSERT INTO users (role) VALUES ('legacy_role')"); err != nil {
		t.Fatalf("insert unknown-role fixture: %v", err)
	}
	if err := executeSQLFile(db, filepath.Join("..", "..", "sqlArchiving", "163_seed_rbac_compatibility.sql")); err == nil {
		t.Fatal("compatibility seed accepted an unknown historical role")
	}
	_, _ = db.Exec("ROLLBACK")
}

func executeSQLFile(db *sql.DB, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	for _, statement := range strings.Split(string(data), ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("%w\nstatement: %s", err, statement)
		}
	}
	return nil
}

func assertCount(t *testing.T, db *sql.DB, query string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if got != want {
		t.Fatalf("count query got %d, want %d", got, want)
	}
}
