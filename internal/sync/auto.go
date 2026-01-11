package sync

import (
	"sync"
	"time"

	"github.com/existflow/irontask/internal/db"
	"github.com/existflow/irontask/internal/logger"
)

// AutoSync manages automatic background syncing
type AutoSync struct {
	client       *Client
	db           *db.DB
	debounceTime time.Duration
	pollInterval time.Duration
	pending      bool
	syncing      bool // Prevents concurrent sync operations
	mu           sync.Mutex
	stopCh       chan struct{}
	onPull       func()               // Callback when remote changes are pulled
	onConflict   func([]ConflictItem) // Callback when conflicts are detected
	lastError    error
}

// NewAutoSync creates a new auto-sync manager
func NewAutoSync(client *Client, database *db.DB) *AutoSync {
	logger.Info("Initializing auto-sync",
		logger.F("debounceTime", "2s"),
		logger.F("pollInterval", "30s"))

	a := &AutoSync{
		client:       client,
		db:           database,
		debounceTime: 2 * time.Second,  // Wait 2s after last change before syncing
		pollInterval: 30 * time.Second, // Poll for remote changes every 30s
		stopCh:       make(chan struct{}),
	}

	// Start background polling for remote changes
	go a.pollLoop()

	return a
}

// SetOnPull sets a callback function to be called when remote changes are pulled
func (a *AutoSync) SetOnPull(callback func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onPull = callback
}

// SetOnConflict sets a callback function to be called when conflicts are detected
func (a *AutoSync) SetOnConflict(callback func([]ConflictItem)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onConflict = callback
}

// pollLoop periodically checks for remote changes
func (a *AutoSync) pollLoop() {
	logger.Debug("Starting auto-sync poll loop")
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debug("Auto-sync poll tick")
			if a.client.CanAutoSync() {
				logger.Debug("Triggering auto-sync from poll")
				a.doSync()
			} else {
				logger.Debug("Auto-sync not available, skipping poll")
			}
		case <-a.stopCh:
			logger.Info("Auto-sync poll loop stopped")
			return
		}
	}
}

// doSync performs the actual sync, with locking to prevent concurrent syncs
func (a *AutoSync) doSync() {
	a.mu.Lock()
	if a.syncing {
		logger.Debug("Sync already in progress, skipping")
		a.mu.Unlock()
		return // Already syncing, skip
	}
	a.syncing = true
	a.mu.Unlock()

	logger.Info("Starting auto-sync")
	startTime := time.Now()

	defer func() {
		a.mu.Lock()
		a.syncing = false
		a.mu.Unlock()
		duration := time.Since(startTime)
		logger.Debug("Auto-sync completed", logger.F("duration", duration.String()))
	}()

	result, err := a.client.Sync(a.db, SyncModeMerge)
	a.mu.Lock()
	a.lastError = err
	a.mu.Unlock()

	if err != nil {
		logger.Error("Auto-sync failed", logger.F("error", err))
		return
	}

	logger.Info("Auto-sync successful",
		logger.F("pushed", result.Pushed),
		logger.F("pulled", result.Pulled))

	// If we pulled changes, notify the callback
	if result.Pulled > 0 {
		logger.Debug("Remote changes pulled, triggering callback")
		a.mu.Lock()
		pullCallback := a.onPull
		a.mu.Unlock()

		if pullCallback != nil {
			pullCallback()
		}
	}

	// If we have conflicts, notify the callback
	if len(result.Conflicts) > 0 {
		logger.Info("Sync conflicts detected, triggering callback", logger.F("count", len(result.Conflicts)))
		a.mu.Lock()
		conflictCallback := a.onConflict
		a.mu.Unlock()

		if conflictCallback != nil {
			conflictCallback(result.Conflicts)
		}
	}
}

// TriggerSync marks that a sync is needed (debounced)
func (a *AutoSync) TriggerSync() {
	if !a.client.CanAutoSync() {
		logger.Debug("Auto-sync not available, ignoring trigger")
		return
	}

	a.mu.Lock()
	if !a.pending {
		logger.Debug("Sync triggered, starting debounce timer")
		a.pending = true
		go a.debouncedSync()
	} else {
		logger.Debug("Sync already pending, ignoring trigger")
	}
	a.mu.Unlock()
}

func (a *AutoSync) debouncedSync() {
	// Wait for debounce period
	timer := time.NewTimer(a.debounceTime)
	defer timer.Stop()

	select {
	case <-timer.C:
		a.mu.Lock()
		a.pending = false
		a.mu.Unlock()
		a.doSync()
	case <-a.stopCh:
		return
	}
}

// Stop stops the auto-sync manager
func (a *AutoSync) Stop() {
	logger.Info("Stopping auto-sync")
	close(a.stopCh)
}

// SyncNowIfPending performs immediate sync if there are pending changes
func (a *AutoSync) SyncNowIfPending() error {
	a.mu.Lock()
	isPending := a.pending
	a.pending = false
	a.mu.Unlock()

	if !isPending {
		logger.Debug("No pending sync")
		return nil
	}

	logger.Info("Executing pending sync immediately")
	_, err := a.client.Sync(a.db, SyncModeMerge)
	if err != nil {
		logger.Error("Immediate sync failed", logger.F("error", err))
	} else {
		logger.Info("Immediate sync completed successfully")
	}
	return err
}

// IsPending returns true if a sync is scheduled or running
func (a *AutoSync) IsPending() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.pending
}

// GetLastError returns the last sync error
func (a *AutoSync) GetLastError() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastError
}
