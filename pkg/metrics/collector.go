package metrics

import (
	"io/fs"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ekilie/beamdrop/pkg/storage"
)

// Collector periodically gathers storage and runtime gauges and
// pushes them into the Prometheus metrics.  It runs in a background
// goroutine and can be stopped via Stop().
type Collector struct {
	sharedDir string
	interval  time.Duration
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// NewCollector creates a new background metrics collector.
// Call Start() to begin collection and Stop() on shutdown.
func NewCollector(sharedDir string, interval time.Duration) *Collector {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	return &Collector{
		sharedDir: sharedDir,
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the periodic collection loop.
func (c *Collector) Start() {
	c.wg.Go(func() {
		// Collect once immediately on startup
		c.collect()

		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.collect()
			case <-c.stopCh:
				return
			}
		}
	})
	slog.Info("Metrics collector started", "interval", c.interval)
}

// Stop signals the collection loop to exit and waits for it to finish.
func (c *Collector) Stop() {
	close(c.stopCh)
	c.wg.Wait()
	slog.Info("Metrics collector stopped")
}

// collect gathers current values and sets the Prometheus gauges.
func (c *Collector) collect() {
	// --- Storage usage ---
	// Use the cached result from storage.GetDirStorageUsage to avoid
	// duplicating the full filesystem walk that the storage layer already does.
	usage, err := storage.GetDirStorageUsage(c.sharedDir)
	if err == nil {
		StorageBytes.Set(float64(usage.UsedBytes))
		StorageTotalBytes.Set(float64(usage.TotalBytes))
		StorageFreeBytes.Set(float64(usage.FreeBytes))
	}

	// --- File count is not available from GetDirStorageUsage, so we need
	// a lighter-weight count. We refetch from the cache if it's recent.
	var fileCount int64
	filepath.WalkDir(c.sharedDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != c.sharedDir {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			fileCount++
		}
		return nil
	})
	ObjectsTotal.Set(float64(fileCount))

	// --- Runtime ---
	GoroutinesCount.Set(float64(runtime.NumGoroutine()))
}
