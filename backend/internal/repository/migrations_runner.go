package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
)

// schemaMigrationsTableDDL 定义迁移记录表的 DDL。
// 该表用于跟踪已应用的迁移文件及其校验和。
// - filename: 迁移文件名，作为主键唯一标识每个迁移
// - checksum: 文件内容的 SHA256 哈希值，用于检测迁移文件是否被篡改
// - applied_at: 迁移应用时间戳
const schemaMigrationsTableDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	filename   VARCHAR(255) PRIMARY KEY,
	checksum   CHAR(64) NOT NULL,
	applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);
`

const atlasSchemaRevisionsTableDDL = `
CREATE TABLE IF NOT EXISTS atlas_schema_revisions (
version VARCHAR(255) PRIMARY KEY,
description TEXT NOT NULL,
type INTEGER NOT NULL,
applied INTEGER NOT NULL DEFAULT 0,
total INTEGER NOT NULL DEFAULT 0,
executed_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
execution_time BIGINT NOT NULL DEFAULT 0,
error TEXT NULL,
error_stmt TEXT NULL,
hash VARCHAR(64) NOT NULL DEFAULT '',
partial_hashes TEXT NULL,
operator_version TEXT NULL
);
`

// migrationsAdvisoryLockID is kept for legacy tests while the runtime uses the
// named GoldenDB/MySQL GET_LOCK key below.
const migrationsAdvisoryLockID int64 = 82931720485713631

const migrationsAdvisoryLockName = "sub2api:migrations"
const migrationsLockRetryInterval = 500 * time.Millisecond
const nonTransactionalMigrationSuffix = "_notx.sql"
const paymentOrdersOutTradeNoUniqueMigration = "120_enforce_payment_orders_out_trade_no_unique_notx.sql"
const paymentOrdersOutTradeNoUniqueIndex = "paymentorder_out_trade_no_unique"
const schedulerOutboxPendingDedupKeyMigration = "153_scheduler_outbox_pending_dedup_key_index_notx.sql"
const schedulerOutboxPendingDedupKeyIndex = "idx_scheduler_outbox_pending_dedup_key"

var (
	createIndexIfNotExistsPattern = regexp.MustCompile(`(?is)^\s*CREATE\s+(UNIQUE\s+)?INDEX\s+IF\s+NOT\s+EXISTS\s+` + identifierPattern("index") + `\s+ON\s+` + identifierPattern("table"))
	dropIndexIfExistsPattern      = regexp.MustCompile(`(?is)^\s*DROP\s+INDEX\s+IF\s+EXISTS\s+` + identifierPattern("index") + `(?:\s+ON\s+` + identifierPattern("table") + `)?\s*$`)
	alterTablePattern             = regexp.MustCompile(`(?is)^\s*ALTER\s+TABLE\s+` + identifierPattern("table") + `\s+(?P<clauses>.+)$`)
	addColumnIfNotExistsPattern   = regexp.MustCompile(`(?is)^\s*ADD\s+COLUMN\s+IF\s+NOT\s+EXISTS\s+` + identifierPattern("column") + `\s+`)
	dropColumnIfExistsPattern     = regexp.MustCompile(`(?is)^\s*DROP\s+COLUMN\s+IF\s+EXISTS\s+` + identifierPattern("column") + `\s*$`)
	dropObjectCascadePattern      = regexp.MustCompile(`(?is)^(\s*DROP\s+(?:TABLE|VIEW)\s+IF\s+EXISTS\s+.+?)\s+CASCADE\s*$`)
	setLocalPattern               = regexp.MustCompile(`(?is)^\s*SET\s+LOCAL\s+`)
)

type migrationExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type migrationChecksumCompatibilityRule struct {
	fileChecksum       string
	acceptedDBChecksum map[string]struct{}
	acceptedChecksums  map[string]struct{}
}

// migrationChecksumCompatibilityRules 仅用于兼容历史上误修改过的迁移文件 checksum。
// 规则必须同时匹配「迁移名 + 数据库 checksum + 当前文件 checksum」且两者都落在该迁移的已知版本集合内才会放行，
// 避免放宽全局校验，也允许将误改的历史 migration 回滚为已发布版本而不要求人工修 checksum。
var migrationChecksumCompatibilityRules = map[string]migrationChecksumCompatibilityRule{
	"054_drop_legacy_cache_columns.sql":                       newMigrationChecksumCompatibilityRule("82de761156e03876653e7a6a4eee883cd927847036f779b0b9f34c42a8af7a7d", "182c193f3359946cf094090cd9e57d5c3fd9abaffbc1e8fc378646b8a6fa12b4"),
	"061_add_usage_log_request_type.sql":                      newMigrationChecksumCompatibilityRule("66207e7aa5dd0429c2e2c0fabdaf79783ff157fa0af2e81adff2ee03790ec65c", "08a248652cbab7cfde147fc6ef8cda464f2477674e20b718312faa252e0481c0", "222b4a09c797c22e5922b6b172327c824f5463aaa8760e4f621bc5c22e2be0f3"),
	"109_auth_identity_compat_backfill.sql":                   newMigrationChecksumCompatibilityRule("0580b4602d85435edf9aca1633db580bb3932f26517f75134106f80275ec2ace", "551e498aa5616d2d91096e9d72cf9fb36e418ee22eacc557f8811cadbc9e20ee"),
	"110_pending_auth_and_provider_default_grants.sql":        newMigrationChecksumCompatibilityRule("32cf87ee787b1bb36b5c691367c96eee37518fa3eed6f3322cf68795e3745279", "e3d1f433be2b564cfbdc549adf98fce13c5c7b363ebc20fd05b765d0563b0925"),
	"112_add_payment_order_provider_key_snapshot.sql":         newMigrationChecksumCompatibilityRule("b75f8f56d39455682787696a3d92ad25b055444ca328fb7fca9a460a15d68d99", "ffd3e8a2c9295fa9cbefefd629a78268877e5b51bc970a82d9b3f46ec4ebd15e"),
	"115_auth_identity_legacy_external_backfill.sql":          newMigrationChecksumCompatibilityRule("022aadd97bb53e755f0cf7a3a957e0cb1a1353b0c39ec4de3234acd2871fd04f", "4cf39e508be9fd1a5aa41610cbbebeb80385c9adda45bf78a706de9db4f1385f"),
	"116_auth_identity_legacy_external_safety_reports.sql":    newMigrationChecksumCompatibilityRule("07edb09fa8d04ffb172b0621e3c22f4d1757d20a24ae267b3b36b087ab72d488", "f7757bd929ac67ffb08ce69fa4cf20fad39dbff9d5a5085fb2adabb7607e5877"),
	"118_wechat_dual_mode_and_auth_source_defaults.sql":       newMigrationChecksumCompatibilityRule("b54194d7a3e4fbf710e0a3590d22a2fe7966804c487052a356e0b55f53ef96b0", "e0cdf835d6c688d64100f483d31bc02ac9ebad414bf1837af239a84bf75b8227", "a38243ca0a72c3a01c0a92b7986423054d6133c0399441f853b99802852720fb"),
	"119_enforce_payment_orders_out_trade_no_unique.sql":      newMigrationChecksumCompatibilityRule("0bbe809ae48a9d811dabda1ba1c74955bd71c4a9cc610f9128816818dfa6c11e", "ebd2c67cce0116393fb4f1b5d5116a67c6aceb73820dfb5133d1ff6f36d72d34"),
	"120_enforce_payment_orders_out_trade_no_unique_notx.sql": newMigrationChecksumCompatibilityRule("34aadc0db59a4e390f92a12b73bd74642d9724f33124f73638ae00089ea5e074", "e77921f79d539bc24575cb9c16cbe566d2b23ce816190343d0a7568f6a3fcf61", "707431450603e70a43ce9fbd61e0c12fa67da4875158ccefabacea069587ab22", "04b082b5a239c525154fe9185d324ee2b05ff90da9297e10dba19f9be79aa59a"),
	"123_fix_legacy_auth_source_grant_on_signup_defaults.sql": newMigrationChecksumCompatibilityRule("2ce43c2cd89e9f9e1febd34a407ed9e84d177386c5544b6f02c1f58a21129f57", "6cd33422f215dcd1f486ab6f35c0ea5805d9ca69bb25906d94bc649156657145"),
}

// ApplyMigrations 将嵌入的 SQL 迁移文件应用到指定的数据库。
//
// 该函数可以在每次应用启动时安全调用：
// - 已应用的迁移会被自动跳过（通过校验 filename 判断）
// - 如果迁移文件内容被修改（checksum 不匹配），会返回错误
// - 使用 PostgreSQL Advisory Lock 确保多实例并发安全
//
// 参数：
//   - ctx: 上下文，用于超时控制和取消
//   - db: 数据库连接
//
// 返回：
//   - error: 迁移过程中的任何错误
func ApplyMigrations(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("nil sql db")
	}
	return applyMigrationsFS(ctx, db, migrations.FS)
}

// applyMigrationsFS 是迁移执行的核心实现。
// 它从指定的文件系统读取 SQL 迁移文件并按顺序应用。
//
// 迁移执行流程：
//  1. 获取 PostgreSQL Advisory Lock，防止多实例并发迁移
//  2. 确保 schema_migrations 表存在
//  3. 按文件名排序读取所有 .sql 文件
//  4. 对于每个迁移文件：
//     - 计算文件内容的 SHA256 校验和
//     - 检查该迁移是否已应用（通过 filename 查询）
//     - 如果已应用，验证校验和是否匹配
//     - 如果未应用，在事务中执行迁移并记录
//  5. 释放 Advisory Lock
//
// 参数：
//   - ctx: 上下文
//   - db: 数据库连接
//   - fsys: 包含迁移文件的文件系统（通常是 embed.FS）
func applyMigrationsFS(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	if db == nil {
		return errors.New("nil sql db")
	}

	// 获取分布式锁，确保多实例部署时只有一个实例执行迁移。
	// 这是 PostgreSQL 特有的 Advisory Lock 机制。
	if err := pgAdvisoryLock(ctx, db); err != nil {
		return err
	}
	defer func() {
		// 无论迁移是否成功，都要释放锁。
		// 使用 context.Background() 确保即使原 ctx 已取消也能释放锁。
		_ = pgAdvisoryUnlock(context.Background(), db)
	}()

	// 创建迁移记录表（如果不存在）。
	// 该表记录所有已应用的迁移及其校验和。
	if _, err := db.ExecContext(ctx, schemaMigrationsTableDDL); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	// 自动对齐 Atlas 基线（如果检测到 legacy schema_migrations 且缺失 atlas_schema_revisions）。
	if err := ensureAtlasBaselineAligned(ctx, db, fsys); err != nil {
		return err
	}

	// 获取所有 .sql 迁移文件并按文件名排序。
	// 命名规范：使用零填充数字前缀（如 001_init.sql, 002_add_users.sql）。
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files) // 确保按文件名顺序执行迁移

	for _, name := range files {
		// 读取迁移文件内容
		contentBytes, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		content := strings.TrimSpace(string(contentBytes))
		if content == "" {
			continue // 跳过空文件
		}

		// 计算文件内容的 SHA256 校验和，用于检测文件是否被修改。
		// 这是一种防篡改机制：如果有人修改了已应用的迁移文件，系统会拒绝启动。
		sum := sha256.Sum256([]byte(content))
		checksum := hex.EncodeToString(sum[:])

		// 检查该迁移是否已经应用
		var existing string
		rowErr := db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE filename = ?", name).Scan(&existing)
		if rowErr == nil {
			// 迁移已应用，验证校验和是否匹配
			if existing != checksum {
				// 兼容特定历史误改场景（仅白名单规则），其余仍保持严格不可变约束。
				if isMigrationChecksumCompatible(name, existing, checksum) {
					continue
				}
				// 校验和不匹配意味着迁移文件在应用后被修改，这是危险的。
				// 正确的做法是创建新的迁移文件来进行变更。
				return fmt.Errorf(
					"migration %s checksum mismatch (db=%s file=%s)\n"+
						"This means the migration file was modified after being applied to the database.\n"+
						"Solutions:\n"+
						"  1. Revert to original: git log --oneline -- migrations/%s && git checkout <commit> -- migrations/%s\n"+
						"  2. For new changes, create a new migration file instead of modifying existing ones\n"+
						"Note: Modifying applied migrations breaks the immutability principle and can cause inconsistencies across environments",
					name, existing, checksum, name, name,
				)
			}
			continue // 迁移已应用且校验和匹配，跳过
		}
		if !errors.Is(rowErr, sql.ErrNoRows) {
			return fmt.Errorf("check migration %s: %w", name, rowErr)
		}

		nonTx, err := validateMigrationExecutionMode(name, content)
		if err != nil {
			return fmt.Errorf("validate migration %s: %w", name, err)
		}

		if nonTx {
			if err := prepareNonTransactionalMigration(ctx, db, name); err != nil {
				return fmt.Errorf("prepare migration %s: %w", name, err)
			}

			// *_notx.sql：用于 CREATE/DROP INDEX CONCURRENTLY 场景，必须非事务执行。
			// 逐条语句执行，避免将多条 CONCURRENTLY 语句放入同一个隐式事务块。
			statements := splitSQLStatements(content)
			for i, stmt := range statements {
				trimmed := strings.TrimSpace(stmt)
				if trimmed == "" {
					continue
				}
				if stripSQLLineComment(trimmed) == "" {
					continue
				}
				if err := executeMigrationStatement(ctx, db, trimmed); err != nil {
					return fmt.Errorf("apply migration %s (non-tx statement %d): %w", name, i+1, err)
				}
			}
			if _, err := db.ExecContext(ctx, "INSERT INTO schema_migrations (filename, checksum) VALUES (?, ?)", name, checksum); err != nil {
				return fmt.Errorf("record migration %s (non-tx): %w", name, err)
			}
			continue
		}

		// 默认迁移在事务中执行，确保原子性：要么完全成功，要么完全回滚。
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}

		// 执行迁移 SQL
		if err := executeMigrationStatements(ctx, tx, content, name, "tx"); err != nil {
			_ = tx.Rollback()
			return err
		}

		// 记录迁移已完成，保存文件名和校验和
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (filename, checksum) VALUES (?, ?)", name, checksum); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		// 提交事务
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}

	return nil
}

func prepareNonTransactionalMigration(ctx context.Context, db *sql.DB, name string) error {
	switch name {
	case paymentOrdersOutTradeNoUniqueMigration:
		return preparePaymentOrdersOutTradeNoUniqueMigration(ctx, db)
	case schedulerOutboxPendingDedupKeyMigration:
		return dropInvalidIndexIfPresent(ctx, db, schedulerOutboxPendingDedupKeyIndex)
	default:
		return nil
	}
}

func preparePaymentOrdersOutTradeNoUniqueMigration(ctx context.Context, db *sql.DB) error {
	duplicates, err := findDuplicatePaymentOrderOutTradeNos(ctx, db)
	if err != nil {
		return fmt.Errorf("precheck duplicate out_trade_no: %w", err)
	}
	if len(duplicates) > 0 {
		return fmt.Errorf(
			"duplicate out_trade_no values block %s; remediate duplicates before retrying: %s",
			paymentOrdersOutTradeNoUniqueMigration,
			strings.Join(duplicates, ", "),
		)
	}

	return dropInvalidIndexIfPresent(ctx, db, paymentOrdersOutTradeNoUniqueIndex)
}

func dropInvalidIndexIfPresent(ctx context.Context, db *sql.DB, indexName string) error {
	invalid, err := indexIsInvalid(ctx, db, indexName)
	if err != nil {
		return fmt.Errorf("check invalid index %s: %w", indexName, err)
	}
	if !invalid {
		return nil
	}

	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP INDEX %s ON payment_orders", indexName)); err != nil {
		return fmt.Errorf("drop invalid index %s: %w", indexName, err)
	}
	return nil
}

func findDuplicatePaymentOrderOutTradeNos(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT out_trade_no, COUNT(*) AS duplicate_count
		FROM payment_orders
		WHERE out_trade_no <> ''
		GROUP BY out_trade_no
		HAVING COUNT(*) > 1
		ORDER BY duplicate_count DESC, out_trade_no
		LIMIT 5
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	duplicates := make([]string, 0, 5)
	for rows.Next() {
		var outTradeNo string
		var duplicateCount int
		if err := rows.Scan(&outTradeNo, &duplicateCount); err != nil {
			return nil, err
		}
		duplicates = append(duplicates, fmt.Sprintf("%s (count=%d)", outTradeNo, duplicateCount))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return duplicates, nil
}

func indexIsInvalid(ctx context.Context, db *sql.DB, indexName string) (bool, error) {
	_ = ctx
	_ = db
	_ = indexName
	return false, nil
}

func ensureAtlasBaselineAligned(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	hasLegacy, err := tableExists(ctx, db, "schema_migrations")
	if err != nil {
		return fmt.Errorf("check schema_migrations: %w", err)
	}
	if !hasLegacy {
		return nil
	}

	hasAtlas, err := tableExists(ctx, db, "atlas_schema_revisions")
	if err != nil {
		return fmt.Errorf("check atlas_schema_revisions: %w", err)
	}
	if !hasAtlas {
		if _, err := db.ExecContext(ctx, atlasSchemaRevisionsTableDDL); err != nil {
			return fmt.Errorf("create atlas_schema_revisions: %w", err)
		}
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM atlas_schema_revisions").Scan(&count); err != nil {
		return fmt.Errorf("count atlas_schema_revisions: %w", err)
	}
	if count > 0 {
		return nil
	}

	version, description, hash, err := latestMigrationBaseline(fsys)
	if err != nil {
		return fmt.Errorf("atlas baseline version: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO atlas_schema_revisions (version, description, type, applied, total, executed_at, execution_time, hash)
		VALUES (?, ?, ?, 0, 0, CURRENT_TIMESTAMP(6), 0, ?)
	`, version, description, 1, hash); err != nil {
		return fmt.Errorf("insert atlas baseline: %w", err)
	}
	return nil
}

func tableExists(ctx context.Context, db *sql.DB, tableName string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = DATABASE() AND table_name = ?
		)
	`, tableName).Scan(&exists)
	return exists, err
}

func executeMigrationStatements(ctx context.Context, exec migrationExecutor, content, name, mode string) error {
	statements := splitSQLStatements(content)
	for i, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		if stripSQLLineComment(trimmed) == "" {
			continue
		}
		if err := executeMigrationStatement(ctx, exec, trimmed); err != nil {
			return fmt.Errorf("apply migration %s (%s statement %d): %w", name, mode, i+1, err)
		}
	}
	return nil
}

func executeMigrationStatement(ctx context.Context, exec migrationExecutor, stmt string) error {
	normalized := stripSQLLineComment(strings.TrimSpace(stmt))
	if normalized == "" {
		return nil
	}
	if setLocalPattern.MatchString(normalized) {
		return nil
	}

	if rewritten, skip, err := rewriteCreateIndexIfNotExists(ctx, exec, normalized); err != nil {
		return err
	} else if skip {
		return nil
	} else if rewritten != "" {
		_, err := exec.ExecContext(ctx, rewritten)
		return err
	}

	if rewritten, skip, err := rewriteDropIndexIfExists(ctx, exec, normalized); err != nil {
		return err
	} else if skip {
		return nil
	} else if rewritten != "" {
		_, err := exec.ExecContext(ctx, rewritten)
		return err
	}

	if rewritten, skip, err := rewriteAlterTableColumnIfExists(ctx, exec, normalized); err != nil {
		return err
	} else if skip {
		return nil
	} else if rewritten != "" {
		_, err := exec.ExecContext(ctx, rewritten)
		return err
	}

	if rewritten := rewriteDropObjectCascade(normalized); rewritten != "" {
		_, err := exec.ExecContext(ctx, rewritten)
		return err
	}

	_, err := exec.ExecContext(ctx, normalized)
	return err
}

func rewriteCreateIndexIfNotExists(ctx context.Context, exec migrationExecutor, stmt string) (string, bool, error) {
	matches := createIndexIfNotExistsPattern.FindStringSubmatch(stmt)
	if matches == nil {
		return "", false, nil
	}

	indexName := regexpNamedMatch(createIndexIfNotExistsPattern, matches, "index_quoted")
	if indexName == "" {
		indexName = regexpNamedMatch(createIndexIfNotExistsPattern, matches, "index_plain")
	}
	tableName := regexpNamedMatch(createIndexIfNotExistsPattern, matches, "table_quoted")
	if tableName == "" {
		tableName = regexpNamedMatch(createIndexIfNotExistsPattern, matches, "table_plain")
	}
	exists, err := indexExists(ctx, exec, tableName, indexName)
	if err != nil {
		return "", false, fmt.Errorf("check index %s on %s: %w", indexName, tableName, err)
	}
	if exists {
		return "", true, nil
	}
	return createIndexIfNotExistsPattern.ReplaceAllString(stmt, "CREATE ${1}INDEX "+quoteIdentifier(indexName)+" ON "+quoteIdentifier(tableName)), false, nil
}

func rewriteDropIndexIfExists(ctx context.Context, exec migrationExecutor, stmt string) (string, bool, error) {
	matches := dropIndexIfExistsPattern.FindStringSubmatch(stmt)
	if matches == nil {
		return "", false, nil
	}

	indexName := regexpNamedMatch(dropIndexIfExistsPattern, matches, "index_quoted")
	if indexName == "" {
		indexName = regexpNamedMatch(dropIndexIfExistsPattern, matches, "index_plain")
	}
	tableName := regexpNamedMatch(dropIndexIfExistsPattern, matches, "table_quoted")
	if tableName == "" {
		tableName = regexpNamedMatch(dropIndexIfExistsPattern, matches, "table_plain")
	}

	if tableName != "" {
		exists, err := indexExists(ctx, exec, tableName, indexName)
		if err != nil {
			return "", false, fmt.Errorf("check index %s on %s: %w", indexName, tableName, err)
		}
		if !exists {
			return "", true, nil
		}
		return "DROP INDEX " + quoteIdentifier(indexName) + " ON " + quoteIdentifier(tableName), false, nil
	}

	tables, err := indexTables(ctx, exec, indexName)
	if err != nil {
		return "", false, fmt.Errorf("lookup index %s: %w", indexName, err)
	}
	if len(tables) == 0 {
		return "", true, nil
	}
	if len(tables) > 1 {
		return "", false, fmt.Errorf("index %s exists on multiple tables: %s", indexName, strings.Join(tables, ", "))
	}
	return "DROP INDEX " + quoteIdentifier(indexName) + " ON " + quoteIdentifier(tables[0]), false, nil
}

func rewriteAlterTableColumnIfExists(ctx context.Context, exec migrationExecutor, stmt string) (string, bool, error) {
	matches := alterTablePattern.FindStringSubmatch(stmt)
	if matches == nil {
		return "", false, nil
	}

	tableName := regexpNamedMatch(alterTablePattern, matches, "table_quoted")
	if tableName == "" {
		tableName = regexpNamedMatch(alterTablePattern, matches, "table_plain")
	}
	clauses := regexpNamedMatch(alterTablePattern, matches, "clauses")
	if !strings.Contains(strings.ToUpper(clauses), "IF") {
		return "", false, nil
	}

	parts := splitTopLevelComma(clauses)
	rewrittenParts := make([]string, 0, len(parts))
	changed := false
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		if addMatches := addColumnIfNotExistsPattern.FindStringSubmatch(trimmed); addMatches != nil {
			columnName := regexpNamedMatch(addColumnIfNotExistsPattern, addMatches, "column_quoted")
			if columnName == "" {
				columnName = regexpNamedMatch(addColumnIfNotExistsPattern, addMatches, "column_plain")
			}
			exists, err := columnExists(ctx, exec, tableName, columnName)
			if err != nil {
				return "", false, fmt.Errorf("check column %s on %s: %w", columnName, tableName, err)
			}
			changed = true
			if exists {
				continue
			}
			rewrittenParts = append(rewrittenParts, addColumnIfNotExistsPattern.ReplaceAllString(trimmed, "ADD COLUMN "+quoteIdentifier(columnName)+" "))
			continue
		}

		if dropMatches := dropColumnIfExistsPattern.FindStringSubmatch(trimmed); dropMatches != nil {
			columnName := regexpNamedMatch(dropColumnIfExistsPattern, dropMatches, "column_quoted")
			if columnName == "" {
				columnName = regexpNamedMatch(dropColumnIfExistsPattern, dropMatches, "column_plain")
			}
			exists, err := columnExists(ctx, exec, tableName, columnName)
			if err != nil {
				return "", false, fmt.Errorf("check column %s on %s: %w", columnName, tableName, err)
			}
			changed = true
			if !exists {
				continue
			}
			rewrittenParts = append(rewrittenParts, "DROP COLUMN "+quoteIdentifier(columnName))
			continue
		}

		rewrittenParts = append(rewrittenParts, trimmed)
	}

	if !changed {
		return "", false, nil
	}
	if len(rewrittenParts) == 0 {
		return "", true, nil
	}
	return "ALTER TABLE " + quoteIdentifier(tableName) + " " + strings.Join(rewrittenParts, ", "), false, nil
}

func rewriteDropObjectCascade(stmt string) string {
	matches := dropObjectCascadePattern.FindStringSubmatch(stmt)
	if matches == nil {
		return ""
	}
	return matches[1]
}

func indexExists(ctx context.Context, exec migrationExecutor, tableName, indexName string) (bool, error) {
	var exists bool
	err := exec.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.statistics
			WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?
		)
	`, tableName, indexName).Scan(&exists)
	return exists, err
}

func indexTables(ctx context.Context, exec migrationExecutor, indexName string) ([]string, error) {
	rows, err := queryContext(ctx, exec, `
		SELECT DISTINCT table_name
		FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND index_name = ?
		ORDER BY table_name
	`, indexName)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}
	return tables, rows.Err()
}

func columnExists(ctx context.Context, exec migrationExecutor, tableName, columnName string) (bool, error) {
	var exists bool
	err := exec.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
		)
	`, tableName, columnName).Scan(&exists)
	return exists, err
}

func splitTopLevelComma(s string) []string {
	var parts []string
	start := 0
	depth := 0
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	for i, r := range s {
		switch r {
		case '\'':
			if !inDoubleQuote && !inBacktick {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote && !inBacktick {
				inDoubleQuote = !inDoubleQuote
			}
		case '`':
			if !inSingleQuote && !inDoubleQuote {
				inBacktick = !inBacktick
			}
		case '(':
			if !inSingleQuote && !inDoubleQuote && !inBacktick {
				depth++
			}
		case ')':
			if !inSingleQuote && !inDoubleQuote && !inBacktick && depth > 0 {
				depth--
			}
		case ',':
			if !inSingleQuote && !inDoubleQuote && !inBacktick && depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func queryContext(ctx context.Context, exec migrationExecutor, query string, args ...any) (*sql.Rows, error) {
	q, ok := exec.(queryer)
	if !ok {
		return nil, errors.New("migration executor does not support QueryContext")
	}
	return q.QueryContext(ctx, query, args...)
}

func identifierPattern(name string) string {
	return `(?:` + "`" + `(?P<` + name + `_quoted>[^` + "`" + `]+)` + "`" + `|(?P<` + name + `_plain>[A-Za-z0-9_]+))`
}

func regexpNamedMatch(re *regexp.Regexp, matches []string, name string) string {
	for i, groupName := range re.SubexpNames() {
		if groupName == name && i < len(matches) {
			return matches[i]
		}
	}
	return ""
}

func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func latestMigrationBaseline(fsys fs.FS) (string, string, string, error) {
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return "", "", "", err
	}
	if len(files) == 0 {
		return "baseline", "baseline", "", nil
	}
	sort.Strings(files)
	name := files[len(files)-1]
	contentBytes, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "", "", "", err
	}
	content := strings.TrimSpace(string(contentBytes))
	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])
	version := strings.TrimSuffix(name, ".sql")
	return version, version, hash, nil
}

func checksumSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func newMigrationChecksumCompatibilityRule(fileChecksum string, acceptedDBChecksums ...string) migrationChecksumCompatibilityRule {
	return migrationChecksumCompatibilityRule{
		fileChecksum:       fileChecksum,
		acceptedDBChecksum: checksumSet(acceptedDBChecksums...),
		acceptedChecksums:  checksumSet(append([]string{fileChecksum}, acceptedDBChecksums...)...),
	}
}

func isMigrationChecksumCompatible(name, dbChecksum, fileChecksum string) bool {
	rule, ok := migrationChecksumCompatibilityRules[name]
	if !ok {
		return false
	}
	_, dbOK := rule.acceptedChecksums[dbChecksum]
	if !dbOK {
		return false
	}
	_, fileOK := rule.acceptedChecksums[fileChecksum]
	return fileOK
}

func validateMigrationExecutionMode(name, content string) (bool, error) {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	upperContent := strings.ToUpper(content)
	nonTx := strings.HasSuffix(normalizedName, nonTransactionalMigrationSuffix)

	if !nonTx {
		return false, nil
	}

	if strings.Contains(upperContent, "BEGIN") || strings.Contains(upperContent, "COMMIT") || strings.Contains(upperContent, "ROLLBACK") {
		return false, errors.New("*_notx.sql must not contain transaction control statements (BEGIN/COMMIT/ROLLBACK)")
	}

	statements := splitSQLStatements(content)
	for _, stmt := range statements {
		normalizedStmt := strings.ToUpper(stripSQLLineComment(strings.TrimSpace(stmt)))
		if normalizedStmt == "" {
			continue
		}

		_ = upperContent
	}

	return true, nil
}

func splitSQLStatements(content string) []string {
	var out []string
	start := 0
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false
	for i := 0; i < len(content); i++ {
		ch := content[i]
		var next byte
		if i+1 < len(content) {
			next = content[i+1]
		}

		if inLineComment {
			if ch == '\n' || ch == '\r' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if !inSingleQuote && !inDoubleQuote && !inBacktick {
			if ch == '-' && next == '-' {
				inLineComment = true
				i++
				continue
			}
			if ch == '/' && next == '*' {
				inBlockComment = true
				i++
				continue
			}
		}

		switch ch {
		case '\'':
			if !inDoubleQuote && !inBacktick {
				if inSingleQuote && next == '\'' {
					i++
				} else {
					inSingleQuote = !inSingleQuote
				}
			}
		case '"':
			if !inSingleQuote && !inBacktick {
				inDoubleQuote = !inDoubleQuote
			}
		case '`':
			if !inSingleQuote && !inDoubleQuote {
				inBacktick = !inBacktick
			}
		case ';':
			if !inSingleQuote && !inDoubleQuote && !inBacktick {
				part := content[start:i]
				if strings.TrimSpace(part) != "" {
					out = append(out, part)
				}
				start = i + 1
			}
		}
	}
	if tail := content[start:]; strings.TrimSpace(tail) != "" {
		out = append(out, tail)
	}
	return out
}

func stripSQLLineComment(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// pgAdvisoryLock 获取 PostgreSQL Advisory Lock。
// Advisory Lock 是一种轻量级的锁机制，不与任何特定的数据库对象关联。
// 它非常适合用于应用层面的分布式锁场景，如迁移序列化。
func pgAdvisoryLock(ctx context.Context, db *sql.DB) error {
	ticker := time.NewTicker(migrationsLockRetryInterval)
	defer ticker.Stop()

	for {
		var locked bool
		var lockResult int
		if err := db.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", migrationsAdvisoryLockName).Scan(&lockResult); err != nil {
			return fmt.Errorf("acquire migrations lock: %w", err)
		}
		locked = lockResult == 1
		if locked {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("acquire migrations lock: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

// pgAdvisoryUnlock 释放 PostgreSQL Advisory Lock。
// 必须在获取锁后确保释放，否则会阻塞其他实例的迁移操作。
func pgAdvisoryUnlock(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "SELECT RELEASE_LOCK(?)", migrationsAdvisoryLockName)
	if err != nil {
		return fmt.Errorf("release migrations lock: %w", err)
	}
	return nil
}
