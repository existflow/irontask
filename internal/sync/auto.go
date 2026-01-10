package sync

import (
	"sync"
	"time"

	"github.com/existflow/irontask/internal/db"
)

// AutoSync manages automatic background syncing
type AutoSync struct {
	client       *Client
	db           *db.DB
	debounceTime time.Duration
	pollInterval time.Duration
	pending      bool
	mu           sync.Mutex
	stopCh       chan struct{}
	onPull       func() // Callback when remote changes are pulled
}

// NewAutoSync creates a new auto-sync manager
func NewAutoSync(client *Client, database *db.DB) *AutoSync {
	a := &AutoSync{
		client:       client,
		db:           database,
		debounceTime: 5 * time.Second,  // Wait 5s after last change before syncing
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

// pollLoop periodically checks for remote changes
func (a *AutoSync) pollLoop() {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.client.CanAutoSync() {
				a.pullRemoteChanges()
			}
		case <-a.stopCh:
			return
		}
	}
}

// pullRemoteChanges pulls changes from the server
func (a *AutoSync) pullRemoteChanges() {
	result, err := a.client.Sync(a.db, SyncModeMerge)
	if err != nil {
		return
	}

	// If we pulled changes, notify the callback
	if result.Pulled > 0 {
		a.mu.Lock()
		callback := a.onPull
		a.mu.Unlock()

		if callback != nil {
			callback()
		}
	}
}

// TriggerSync marks that a sync is needed (debounced)
func (a *AutoSync) TriggerSync() {
	if !a.client.CanAutoSync() {
		return
	}

	a.mu.Lock()
	if !a.pending {
		a.pending = true
		go a.debouncedSync()
	}
	a.mu.Unlock()
}

func (a *AutoSync) debouncedSync() {
	// Wait for debounce period
	timer := time.NewTimer(a.debounceTime)
	defer timer.Stop()

	select {
	case <-timer.C:
		a.performSync()
	case <-a.stopCh:
		return
	}
}

func (a *AutoSync) performSync() {
	a.mu.Lock()
	a.pending = false
	a.mu.Unlock()

	_, err := a.client.Sync(a.db, SyncModeMerge)
	if err != nil {
		return
	}

	// keeping silent for TUI
}

// Stop stops the auto-sync manager
func (a *AutoSync) Stop() {
	close(a.stopCh)
}

// SyncNowIfPending performs immediate sync if there are pending changes
func (a *AutoSync) SyncNowIfPending() error {
	a.mu.Lock()
	isPending := a.pending
	a.pending = false
	a.mu.Unlock()

	if !isPending {
		return nil
	}

	_, err := a.client.Sync(a.db, SyncModeMerge)
	return err
}

// IsPending returns true if a sync is scheduled or running
func (a *AutoSync) IsPending() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.pending
}
