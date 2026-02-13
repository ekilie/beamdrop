package db

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
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
	logger.Info("Orphan record cleaner started (interval: %v)", CleanupInterval)
}

// Stop signals the background goroutine to exit.
func (oc *OrphanCleaner) Stop() {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	if !oc.stopped {
		close(oc.stopCh)
		oc.stopped = true
		logger.Info("Orphan record cleaner stopped")
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
		logger.Info("Orphan cleanup: removed %d starred files, %d shareable links, %d expired links",
			orphanedStars, orphanedLinks, expiredLinks)
	}
}

// cleanupStarredFiles removes starred records whose file no longer exists.
func (oc *OrphanCleaner) cleanupStarredFiles() int {
	db := GetDB()
	var starredFiles []StarredFile
	if err := db.Find(&starredFiles).Error; err != nil {
		logger.Error("Orphan cleanup: failed to list starred files: %v", err)
		return 0
	}

	removed := 0
	for _, sf := range starredFiles {
		fullPath := filepath.Join(oc.sharedDir, sf.FilePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			if err := db.Delete(&sf).Error; err != nil {
				logger.Error("Orphan cleanup: failed to delete orphaned starred file %q: %v", sf.FilePath, err)
			} else {
				logger.Debug("Orphan cleanup: removed starred file record for missing path %q", sf.FilePath)
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
		logger.Error("Orphan cleanup: failed to list shareable links: %v", err)
		return 0
	}

	removed := 0
	for _, link := range links {
		fullPath := filepath.Join(oc.sharedDir, link.Path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			if err := db.Delete(&link).Error; err != nil {
				logger.Error("Orphan cleanup: failed to delete orphaned shareable link %q (path %q): %v",
					link.Token, link.Path, err)
			} else {
				logger.Debug("Orphan cleanup: removed shareable link %q for missing path %q", link.Token, link.Path)
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
		logger.Error("Orphan cleanup: failed to delete expired links: %v", result.Error)
		return 0
	}
	return int(result.RowsAffected)
}
