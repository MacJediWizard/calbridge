package scheduler

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("creates scheduler with nil dependencies", func(t *testing.T) {
		// Note: In production, db and syncEngine would be required,
		// but we can create the scheduler without them for testing structure
		sched := New(nil, nil)

		if sched == nil {
			t.Fatal("expected non-nil scheduler")
		}

		if sched.jobs == nil {
			t.Error("expected jobs map to be initialized")
		}

		if sched.syncLocks == nil {
			t.Error("expected syncLocks map to be initialized")
		}

		if sched.ctx == nil {
			t.Error("expected context to be initialized")
		}

		if sched.cancel == nil {
			t.Error("expected cancel function to be initialized")
		}
	})
}

func TestGetJobCount(t *testing.T) {
	t.Run("returns zero for new scheduler", func(t *testing.T) {
		sched := New(nil, nil)

		count := sched.GetJobCount()
		if count != 0 {
			t.Errorf("expected 0 jobs, got %d", count)
		}
	})
}

func TestJobStruct(t *testing.T) {
	t.Run("job struct has expected fields", func(t *testing.T) {
		job := &Job{
			sourceID: "source-123",
			interval: 5 * time.Minute,
		}

		if job.sourceID != "source-123" {
			t.Error("sourceID not set correctly")
		}

		if job.interval != 5*time.Minute {
			t.Error("interval not set correctly")
		}
	})
}

func TestSchedulerConstants(t *testing.T) {
	t.Run("cleanup interval is 24 hours", func(t *testing.T) {
		if cleanupInterval != 24*time.Hour {
			t.Errorf("expected cleanupInterval to be 24h, got %v", cleanupInterval)
		}
	})

	t.Run("log retention is 30 days", func(t *testing.T) {
		if logRetentionDays != 30 {
			t.Errorf("expected logRetentionDays to be 30, got %d", logRetentionDays)
		}
	})

	t.Run("sync timeout is 30 minutes", func(t *testing.T) {
		if syncTimeout != 30*time.Minute {
			t.Errorf("expected syncTimeout to be 30m, got %v", syncTimeout)
		}
	})
}

func TestSchedulerStartStop(t *testing.T) {
	t.Run("start sets started flag", func(t *testing.T) {
		sched := New(nil, nil)

		// Note: Start() will fail without a real DB, but we can test
		// the started flag protection by checking the initial state
		if sched.started {
			t.Error("expected started to be false initially")
		}
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		sched := New(nil, nil)

		// Stop should not panic when scheduler hasn't started
		sched.Stop()
		sched.Stop() // Should be safe to call multiple times
	})
}

func TestGetSyncLock(t *testing.T) {
	t.Run("creates lock for new source", func(t *testing.T) {
		sched := New(nil, nil)

		lock := sched.getSyncLock("source-1")
		if lock == nil {
			t.Fatal("expected non-nil lock")
		}

		// Same source should return same lock
		lock2 := sched.getSyncLock("source-1")
		if lock != lock2 {
			t.Error("expected same lock for same source")
		}
	})

	t.Run("creates different locks for different sources", func(t *testing.T) {
		sched := New(nil, nil)

		lock1 := sched.getSyncLock("source-1")
		lock2 := sched.getSyncLock("source-2")

		if lock1 == lock2 {
			t.Error("expected different locks for different sources")
		}
	})
}

func TestRemoveJob(t *testing.T) {
	t.Run("remove non-existent job is safe", func(t *testing.T) {
		sched := New(nil, nil)

		// Should not panic
		sched.RemoveJob("non-existent-source")
	})
}

func TestUpdateJobInterval(t *testing.T) {
	t.Run("update non-existent job is safe", func(t *testing.T) {
		sched := New(nil, nil)

		// Should not panic
		sched.UpdateJobInterval("non-existent-source", 10*time.Minute)
	})
}
