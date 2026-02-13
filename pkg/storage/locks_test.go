package storage

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLockManager_WriteLockSerializes(t *testing.T) {
	lm := NewLockManager(5 * time.Second)
	defer lm.Close()

	var (
		mu      sync.Mutex
		order   []int
		wg      sync.WaitGroup
		started = make(chan struct{})
	)

	// First goroutine grabs the write lock
	unlock1, err := lm.Lock("bucket", "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second goroutine waits for the lock
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(started)
		unlock2, err := lm.Lock("bucket", "key")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
		unlock2()
	}()

	<-started
	// Give goroutine time to block
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	order = append(order, 1)
	mu.Unlock()
	unlock1()

	wg.Wait()

	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("expected order [1, 2], got %v", order)
	}
}

func TestLockManager_ReadLocksAreShared(t *testing.T) {
	lm := NewLockManager(5 * time.Second)
	defer lm.Close()

	var (
		readersInside atomic.Int32
		maxReaders    atomic.Int32
		wg            sync.WaitGroup
	)

	const numReaders = 10

	wg.Add(numReaders)
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			unlock, err := lm.RLock("bucket", "key")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			defer unlock()

			cur := readersInside.Add(1)
			// Track maximum concurrent readers
			for {
				old := maxReaders.Load()
				if cur <= old || maxReaders.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			readersInside.Add(-1)
		}()
	}

	wg.Wait()

	if maxReaders.Load() < 2 {
		t.Errorf("expected multiple concurrent readers, got max %d", maxReaders.Load())
	}
}

func TestLockManager_WriteLockBlocksReaders(t *testing.T) {
	lm := NewLockManager(5 * time.Second)
	defer lm.Close()

	unlock, err := lm.Lock("bucket", "key")
	if err != nil {
		t.Fatal(err)
	}

	readDone := make(chan error, 1)
	go func() {
		_, err := lm.RLock("bucket", "key")
		readDone <- err
	}()

	// Read should be blocked while write lock is held
	select {
	case <-readDone:
		t.Fatal("read lock should be blocked by write lock")
	case <-time.After(100 * time.Millisecond):
		// Expected: read is blocked
	}

	unlock()

	// Now read should succeed
	select {
	case err := <-readDone:
		if err != nil {
			t.Fatalf("read lock failed after write unlock: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("read lock didn't complete after write unlock")
	}
}

func TestLockManager_Timeout(t *testing.T) {
	lm := NewLockManager(100 * time.Millisecond)
	defer lm.Close()

	// Hold a write lock
	unlock, err := lm.Lock("bucket", "key")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	// Try to acquire another write lock — should timeout
	_, err = lm.Lock("bucket", "key")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("expected ErrLockTimeout, got: %v", err)
	}
}

func TestLockManager_ReadTimeout(t *testing.T) {
	lm := NewLockManager(100 * time.Millisecond)
	defer lm.Close()

	// Hold a write lock
	unlock, err := lm.Lock("bucket", "key")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	// Try to read lock — should timeout since write is held
	_, err = lm.RLock("bucket", "key")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("expected ErrLockTimeout, got: %v", err)
	}
}

func TestLockManager_DifferentKeysIndependent(t *testing.T) {
	lm := NewLockManager(5 * time.Second)
	defer lm.Close()

	unlock1, err := lm.Lock("bucket", "key1")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock1()

	// Different key should not be blocked
	unlock2, err := lm.Lock("bucket", "key2")
	if err != nil {
		t.Fatalf("lock for different key should not block: %v", err)
	}
	defer unlock2()

	// Different bucket same key should not be blocked
	unlock3, err := lm.Lock("other-bucket", "key1")
	if err != nil {
		t.Fatalf("lock for different bucket should not block: %v", err)
	}
	defer unlock3()
}

func TestLockManager_Stats(t *testing.T) {
	lm := NewLockManager(5 * time.Second)
	defer lm.Close()

	stats := lm.Stats()
	if stats.ActiveLocks != 0 {
		t.Fatalf("expected 0 active locks, got %d", stats.ActiveLocks)
	}

	unlock1, _ := lm.Lock("bucket", "key1")
	unlock2, _ := lm.RLock("bucket", "key2")
	unlock3, _ := lm.RLock("bucket", "key2")

	stats = lm.Stats()
	if stats.ActiveLocks != 2 {
		t.Errorf("expected 2 active locks, got %d", stats.ActiveLocks)
	}
	if stats.WriteLocks != 1 {
		t.Errorf("expected 1 write lock, got %d", stats.WriteLocks)
	}
	if stats.ReadLocks != 2 {
		t.Errorf("expected 2 read locks, got %d", stats.ReadLocks)
	}

	unlock1()
	unlock2()
	unlock3()

	stats = lm.Stats()
	if stats.ActiveLocks != 0 {
		t.Errorf("expected 0 active locks after unlock, got %d", stats.ActiveLocks)
	}
}

func TestLockManager_ConcurrentWritesSameKey(t *testing.T) {
	lm := NewLockManager(5 * time.Second)
	defer lm.Close()

	var (
		counter int
		wg      sync.WaitGroup
	)

	const numWriters = 50

	wg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func() {
			defer wg.Done()
			unlock, err := lm.Lock("bucket", "key")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			// Non-atomic increment — without locking this would race
			val := counter
			time.Sleep(time.Microsecond) // force context switch
			counter = val + 1
			unlock()
		}()
	}

	wg.Wait()

	if counter != numWriters {
		t.Errorf("expected counter=%d, got %d (data race detected)", numWriters, counter)
	}
}

func TestLockManager_CleanupAfterUnlock(t *testing.T) {
	lm := NewLockManager(5 * time.Second)
	defer lm.Close()

	unlock, err := lm.Lock("bucket", "key")
	if err != nil {
		t.Fatal(err)
	}

	lm.mu.Lock()
	if len(lm.locks) != 1 {
		t.Errorf("expected 1 lock entry, got %d", len(lm.locks))
	}
	lm.mu.Unlock()

	unlock()

	// Give cleanup a moment
	time.Sleep(10 * time.Millisecond)

	lm.mu.Lock()
	if len(lm.locks) != 0 {
		t.Errorf("expected 0 lock entries after unlock, got %d", len(lm.locks))
	}
	lm.mu.Unlock()
}
