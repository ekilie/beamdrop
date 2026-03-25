package metrics

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ekilie/beamdrop/pkg/system"
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
	var totalBytes int64
	var fileCount int64

	err := filepath.WalkDir(c.sharedDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		// Skip hidden/internal directories
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != c.sharedDir {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				totalBytes += info.Size()
				fileCount++
			}
		}
		return nil
	})
	if err != nil {
		slog.Warn("Metrics collector: storage walk failed", "error", err)
	}

	StorageBytes.Set(float64(totalBytes))
	ObjectsTotal.Set(float64(fileCount))

	// --- Filesystem capacity ---
	if info, err := os.Stat(c.sharedDir); err == nil && info.IsDir() {
		du := system.GetDiskUsage(c.sharedDir)
		StorageTotalBytes.Set(float64(du.Total))
		StorageFreeBytes.Set(float64(du.Free))
	}

	// --- Runtime ---
	GoroutinesCount.Set(float64(runtime.NumGoroutine()))
}
