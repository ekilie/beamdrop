package db

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CleanupInterval is how often the orphan cleanup job runs.
const CleanupInterval = 1 * time.Hour

// OrphanCleaner periodically removes DB records that reference
// filesystem paths that no longer exist.
type OrphanCleaner struct {
	sharedDir string
	stopCh    chan struct{}
	stopped   bool
	mu        sync.Mutex
}

// NewOrphanCleaner creates a new cleaner. Call Start() to begin.
func NewOrphanCleaner(sharedDir string) *OrphanCleaner {
	return &OrphanCleaner{
		sharedDir: sharedDir,
		stopCh:    make(chan struct{}),
	}
}

// Start launches the background cleanup goroutine.
func (oc *OrphanCleaner) Start() {
	go oc.loop()
	slog.Info("Orphan record cleaner started", "interval", CleanupInterval)
}

// Stop signals the background goroutine to exit.
func (oc *OrphanCleaner) Stop() {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	if !oc.stopped {
		close(oc.stopCh)
		oc.stopped = true
		slog.Info("Orphan record cleaner stopped")
	}
}

func (oc *OrphanCleaner) loop() {
	// Run once at startup, then on interval
	oc.RunOnce()

	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-oc.stopCh:
			return
		case <-ticker.C:
			oc.RunOnce()
		}
	}
}

// RunOnce performs a single cleanup pass. Can be called directly for testing.
func (oc *OrphanCleaner) RunOnce() {
	orphanedStars := oc.cleanupStarredFiles()
	orphanedLinks := oc.cleanupShareableLinks()
	expiredLinks := oc.cleanupExpiredLinks()

	total := orphanedStars + orphanedLinks + expiredLinks
	if total > 0 {
		slog.Info("Orphan cleanup completed",
			"starred_removed", orphanedStars,
			"links_removed", orphanedLinks,
			"expired_removed", expiredLinks)
	}
}

// cleanupStarredFiles removes starred records whose file no longer exists.
func (oc *OrphanCleaner) cleanupStarredFiles() int {
	db := GetDB()
	var starredFiles []StarredFile
	if err := db.Find(&starredFiles).Error; err != nil {
		slog.Error("Orphan cleanup: failed to list starred files", "error", err)
		return 0
	}

	removed := 0
	for _, sf := range starredFiles {
		fullPath := filepath.Join(oc.sharedDir, sf.FilePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			if err := db.Delete(&sf).Error; err != nil {
				slog.Error("Orphan cleanup: failed to delete orphaned starred file", "path", sf.FilePath, "error", err)
			} else {
				slog.Debug("Orphan cleanup: removed starred file record", "path", sf.FilePath)
				removed++
			}
		}
	}
	return removed
}

// cleanupShareableLinks removes links whose target path no longer exists.
func (oc *OrphanCleaner) cleanupShareableLinks() int {
	db := GetDB()
	var links []ShareableLink
	if err := db.Find(&links).Error; err != nil {
		slog.Error("Orphan cleanup: failed to list shareable links", "error", err)
		return 0
	}

	removed := 0
	for _, link := range links {
		fullPath := filepath.Join(oc.sharedDir, link.Path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			if err := db.Delete(&link).Error; err != nil {
				slog.Error("Orphan cleanup: failed to delete orphaned shareable link",
					"token", link.Token, "path", link.Path, "error", err)
			} else {
				slog.Debug("Orphan cleanup: removed shareable link", "token", link.Token, "path", link.Path)
				removed++
			}
		}
	}
	return removed
}

// cleanupExpiredLinks removes links that have passed their expiration time.
func (oc *OrphanCleaner) cleanupExpiredLinks() int {
	db := GetDB()
	now := time.Now()
	result := db.Where("expires_at IS NOT NULL AND expires_at < ?", now).Delete(&ShareableLink{})
	if result.Error != nil {
		slog.Error("Orphan cleanup: failed to delete expired links", "error", result.Error)
		return 0
	}
	return int(result.RowsAffected)
}
