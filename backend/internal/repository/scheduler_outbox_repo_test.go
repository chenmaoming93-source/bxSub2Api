package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSchedulerOutboxRepositoryListAfterAndReleaseDedupUsesMySQLTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	createdAt := time.Unix(1710000000, 0)
	const selectSQL = `
		SELECT id, event_type, account_id, group_id, payload, created_at
		FROM scheduler_outbox
		WHERE id > ?
		ORDER BY id ASC
		LIMIT ?
		FOR UPDATE
	`
	const updateSQL = "UPDATE scheduler_outbox SET dedup_key = NULL WHERE dedup_key IS NOT NULL AND id IN (?)"
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(selectSQL)).
		WithArgs(int64(10), 2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "account_id", "group_id", "payload", "created_at"}).
			AddRow(int64(11), service.SchedulerOutboxEventAccountChanged, int64(42), nil, []byte(`{"group_ids":[7]}`), createdAt))
	mock.ExpectExec(regexp.QuoteMeta(updateSQL)).
		WithArgs(int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	events, err := repo.ListAfterAndReleaseDedup(context.Background(), 10, 2)

	require.NoError(t, err)
	require.Len(t, events, 1)
	require.EqualValues(t, 11, events[0].ID)
	require.Equal(t, service.SchedulerOutboxEventAccountChanged, events[0].EventType)
	require.NotNil(t, events[0].AccountID)
	require.EqualValues(t, 42, *events[0].AccountID)
	require.Equal(t, createdAt, events[0].CreatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryDeleteConsumedUpToUsesBoundedSubquery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	const expectedSQL = `
		DELETE FROM scheduler_outbox
		WHERE id IN (
			SELECT id FROM (
				SELECT id
				FROM scheduler_outbox
				WHERE id <= ?
					AND created_at < NOW() - INTERVAL 10 SECOND
				ORDER BY id ASC
				LIMIT ?
			) AS doomed
		)
	`
	mock.ExpectExec(regexp.QuoteMeta(expectedSQL)).
		WithArgs(int64(42), 5000).
		WillReturnResult(sqlmock.NewResult(0, 17))

	deleted, err := repo.DeleteConsumedUpTo(context.Background(), 42, 5000)

	require.NoError(t, err)
	require.EqualValues(t, 17, deleted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryDeleteConsumedUpToSkipsNonPositiveWatermark(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}

	deleted, err := repo.DeleteConsumedUpTo(context.Background(), 0, 5000)

	require.NoError(t, err)
	require.EqualValues(t, 0, deleted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryTryAcquireCleanupLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT GET_LOCK(?, 0)")).
		WithArgs("sub2api:scheduler_outbox_cleanup").
		WillReturnRows(sqlmock.NewRows([]string{"get_lock"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta("SELECT RELEASE_LOCK(?)")).
		WithArgs("sub2api:scheduler_outbox_cleanup").
		WillReturnResult(sqlmock.NewResult(0, 1))

	lease, acquired, err := repo.TryAcquireCleanupLock(context.Background())
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, lease)

	lease.Release()

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSchedulerOutboxRepositoryTryAcquireCleanupLockUnavailable(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &schedulerOutboxRepository{db: db}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT GET_LOCK(?, 0)")).
		WithArgs("sub2api:scheduler_outbox_cleanup").
		WillReturnRows(sqlmock.NewRows([]string{"get_lock"}).AddRow(0))

	lease, acquired, err := repo.TryAcquireCleanupLock(context.Background())
	require.NoError(t, err)
	require.False(t, acquired)
	require.Nil(t, lease)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEnqueueSchedulerOutboxDedupUsesMySQLInsertIgnore(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	accountID := int64(42)
	const expectedSQL = `
			INSERT IGNORE INTO scheduler_outbox (event_type, account_id, group_id, payload, dedup_key)
			VALUES (?, ?, ?, ?, ?)
		`
	mock.ExpectExec(regexp.QuoteMeta(expectedSQL)).
		WithArgs(service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = enqueueSchedulerOutbox(context.Background(), db, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// buildSchedulerGroupPayload 在 groupIDs 为空时必须返回 untyped nil（any），
// 否则 enqueueSchedulerOutbox 的 "payload != nil" 接口判空会被 typed-nil 欺骗，
// 把 payload marshal 成 "null" 写入 dedup_key 哈希，破坏与其他 nil-payload
// 调用的去重一致性。本测试用 ungrouped 账号场景验证两条路径的 dedup_key 一致。
func TestEnqueueSchedulerOutbox_UngroupedAccountDedupesWithLiteralNilPayload(t *testing.T) {
	accountID := int64(42)

	// Path A: 显式 nil payload（如 SetError、SetStatus 等调用模式）
	keyLiteralNil := schedulerOutboxDedupKey("account_changed", &accountID, nil, nil)

	// Path B: buildSchedulerGroupPayload(account.GroupIDs) 当账号没有任何分组
	emptyGroupsPayload := buildSchedulerGroupPayload(nil)
	require.Nil(t, emptyGroupsPayload,
		"buildSchedulerGroupPayload(empty) must return untyped-nil any to avoid typed-nil marshal")

	// 模拟 enqueueSchedulerOutbox 内部的判空逻辑
	var payloadJSON []byte
	if emptyGroupsPayload != nil {
		t.Fatalf("typed-nil regression: buildSchedulerGroupPayload(empty) interface should be nil")
	}
	keyEmptyGroups := schedulerOutboxDedupKey("account_changed", &accountID, nil, payloadJSON)

	require.Equal(t, keyLiteralNil, keyEmptyGroups,
		"ungrouped-account account_changed must share dedup_key with other nil-payload variants")
}
