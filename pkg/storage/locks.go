package storage

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var (
	// ErrLockTimeout is returned when a lock cannot be acquired within the timeout period.
	ErrLockTimeout = errors.New("lock acquisition timed out")
	// ErrDeadlock is returned when a potential deadlock is detected.
	ErrDeadlock = errors.New("potential deadlock detected")
)

// DefaultLockTimeout is the default maximum time to wait for lock acquisition.
const DefaultLockTimeout = 30 * time.Second

// DeadlockDetectionInterval is how often the deadlock detector scans for stuck locks.
const DeadlockDetectionInterval = 10 * time.Second

// StaleLockerThreshold is how long a lock can be held before it's considered stale.
const StaleLockerThreshold = 5 * time.Minute

// lockEntry tracks an individual read/write lock for a single object key.
type lockEntry struct {
	mu        sync.RWMutex
	refCount  int       // number of goroutines referencing this entry
	writeLock bool      // whether a write lock is currently held
	readers   int       // number of current readers
	lockedAt  time.Time // when the current write lock was acquired (zero if none)
	key       string    // the object key (for diagnostics)
}

// LockManager provides per-object read/write locking with timeout and deadlock detection.
type LockManager struct {
	mu      sync.Mutex            // protects the locks map
	locks   map[string]*lockEntry // key -> lock entry
	timeout time.Duration         // lock acquisition timeout
	stopCh  chan struct{}         // signals the deadlock detector to stop
	stopped bool                  // prevents double-close
}

// NewLockManager creates a new LockManager with the given lock acquisition timeout.
// Pass 0 to use DefaultLockTimeout.
func NewLockManager(timeout time.Duration) *LockManager {
	if timeout <= 0 {
		timeout = DefaultLockTimeout
	}
	lm := &LockManager{
		locks:   make(map[string]*lockEntry),
		timeout: timeout,
		stopCh:  make(chan struct{}),
	}
	go lm.deadlockDetector()
	return lm
}

// objectKey builds a canonical lock key from bucket + object key.
func objectKey(bucket, key string) string {
	return bucket + "/" + key
}

// getOrCreateEntry returns the lockEntry for a key, creating one if needed.
// Caller must NOT hold lm.mu.
func (lm *LockManager) getOrCreateEntry(key string) *lockEntry {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	entry, ok := lm.locks[key]
	if !ok {
		entry = &lockEntry{key: key}
		lm.locks[key] = entry
	}
	entry.refCount++
	return entry
}

// releaseEntry decrements the refCount and removes the entry if no longer needed.
func (lm *LockManager) releaseEntry(key string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	entry, ok := lm.locks[key]
	if !ok {
		return
	}
	entry.refCount--
	if entry.refCount <= 0 {
		delete(lm.locks, key)
	}
}

// RLock acquires a read lock for the given bucket/key with timeout.
// Returns an unlock function that MUST be called when the read is done.
func (lm *LockManager) RLock(bucket, key string) (unlock func(), err error) {
	k := objectKey(bucket, key)
	entry := lm.getOrCreateEntry(k)

	locked := make(chan struct{}, 1)
	go func() {
		entry.mu.RLock()
		locked <- struct{}{}
	}()

	timer := time.NewTimer(lm.timeout)
	defer timer.Stop()

	select {
	case <-locked:
		lm.mu.Lock()
		entry.readers++
		lm.mu.Unlock()

		return func() {
			lm.mu.Lock()
			entry.readers--
			lm.mu.Unlock()
			entry.mu.RUnlock()
			lm.releaseEntry(k)
		}, nil

	case <-timer.C:
		// Timeout — the locking goroutine is still blocked waiting for
		// the RWMutex. We must let it complete so it doesn't leak.
		// But we can avoid spawning a SECOND goroutine by doing the
		// cleanup inline after the lock is released.
		go func() {
			<-locked
			entry.mu.RUnlock()
			lm.releaseEntry(k)
		}()
		return nil, fmt.Errorf("%w: read lock for %s after %v", ErrLockTimeout, k, lm.timeout)
	}
}

// Lock acquires a write lock for the given bucket/key with timeout.
// Returns an unlock function that MUST be called when the write is done.
func (lm *LockManager) Lock(bucket, key string) (unlock func(), err error) {
	k := objectKey(bucket, key)
	entry := lm.getOrCreateEntry(k)

	locked := make(chan struct{}, 1)
	go func() {
		entry.mu.Lock()
		locked <- struct{}{}
	}()

	timer := time.NewTimer(lm.timeout)
	defer timer.Stop()

	select {
	case <-locked:
		lm.mu.Lock()
		entry.writeLock = true
		entry.lockedAt = time.Now()
		lm.mu.Unlock()

		return func() {
			lm.mu.Lock()
			entry.writeLock = false
			entry.lockedAt = time.Time{}
			lm.mu.Unlock()
			entry.mu.Unlock()
			lm.releaseEntry(k)
		}, nil

	case <-timer.C:
		go func() {
			<-locked
			entry.mu.Unlock()
			lm.releaseEntry(k)
		}()
		return nil, fmt.Errorf("%w: write lock for %s after %v", ErrLockTimeout, k, lm.timeout)
	}
}

// deadlockDetector periodically scans for locks held longer than StaleLockerThreshold
// and logs warnings. This helps operators identify stuck operations.
func (lm *LockManager) deadlockDetector() {
	ticker := time.NewTicker(DeadlockDetectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-lm.stopCh:
			return
		case <-ticker.C:
			lm.detectStale()
		}
	}
}

// detectStale checks for write locks held longer than the stale threshold.
func (lm *LockManager) detectStale() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	now := time.Now()
	for key, entry := range lm.locks {
		if entry.writeLock && !entry.lockedAt.IsZero() {
			held := now.Sub(entry.lockedAt)
			if held > StaleLockerThreshold {
				slog.Warn("Potential deadlock detected", "key", key, "held", held.Round(time.Second), "threshold", StaleLockerThreshold)
			}
		}
	}
}

// Stats returns a snapshot of current lock statistics.
func (lm *LockManager) Stats() LockStats {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	stats := LockStats{
		ActiveLocks: len(lm.locks),
	}
	for _, entry := range lm.locks {
		if entry.writeLock {
			stats.WriteLocks++
		}
		stats.ReadLocks += entry.readers
	}
	return stats
}

// LockStats contains a snapshot of lock manager state.
type LockStats struct {
	ActiveLocks int `json:"activeLocks"`
	WriteLocks  int `json:"writeLocks"`
	ReadLocks   int `json:"readLocks"`
}

// Close shuts down the deadlock detector goroutine.
func (lm *LockManager) Close() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if !lm.stopped {
		close(lm.stopCh)
		lm.stopped = true
	}
}
