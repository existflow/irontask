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
	pending      bool
	mu           sync.Mutex
	stopCh       chan struct{}
}

// NewAutoSync creates a new auto-sync manager
func NewAutoSync(client *Client, database *db.DB) *AutoSync {
	return &AutoSync{
		client:       client,
		db:           database,
		debounceTime: 5 * time.Second, // Wait 5s after last change before syncing
		stopCh:       make(chan struct{}),
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
