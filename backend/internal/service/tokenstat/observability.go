package tokenstat

import "sync/atomic"

type ObservabilitySnapshot struct {
	Enqueued          uint64 `json:"enqueued"`
	DroppedQueueFull  uint64 `json:"dropped_queue_full"`
	InvalidEvents     uint64 `json:"invalid_events"`
	RedisWriteFailure uint64 `json:"redis_write_failures"`
	SyncedRows        uint64 `json:"synced_rows"`
	SyncFailures      uint64 `json:"sync_failures"`
	FinalizedPeriods  uint64 `json:"finalized_periods"`
	FinalizeFailures  uint64 `json:"finalize_failures"`
	QuotaChecks       uint64 `json:"quota_checks"`
	QuotaExceeded     uint64 `json:"quota_exceeded"`
	QuotaFailOpen     uint64 `json:"quota_fail_open"`
	ConfigVersion     uint64 `json:"config_version"`
}

var observability struct {
	enqueued, dropped, invalid, redisWriteFailure atomic.Uint64
	syncedRows, syncFailures                      atomic.Uint64
	finalizedPeriods, finalizeFailures            atomic.Uint64
	quotaChecks, quotaExceeded, quotaFailOpen     atomic.Uint64
	configVersion                                 atomic.Uint64
}

func MetricsSnapshot() ObservabilitySnapshot {
	return ObservabilitySnapshot{
		Enqueued: observability.enqueued.Load(), DroppedQueueFull: observability.dropped.Load(),
		InvalidEvents: observability.invalid.Load(), RedisWriteFailure: observability.redisWriteFailure.Load(),
		SyncedRows: observability.syncedRows.Load(), SyncFailures: observability.syncFailures.Load(),
		FinalizedPeriods: observability.finalizedPeriods.Load(), FinalizeFailures: observability.finalizeFailures.Load(),
		QuotaChecks: observability.quotaChecks.Load(), QuotaExceeded: observability.quotaExceeded.Load(),
		QuotaFailOpen: observability.quotaFailOpen.Load(), ConfigVersion: observability.configVersion.Load(),
	}
}

func RecordSync(rows uint64, failed bool) {
	observability.syncedRows.Add(rows)
	if failed {
		observability.syncFailures.Add(1)
	}
}

func RecordFinalization(failed bool) {
	if failed {
		observability.finalizeFailures.Add(1)
	} else {
		observability.finalizedPeriods.Add(1)
	}
}
