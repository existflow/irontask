package cli

import (
	"fmt"

	"github.com/existflow/irontask/internal/db"
	"github.com/existflow/irontask/internal/sync"
)

// MaybeSyncCLI performs sync if --sync flag is set or if auto-sync is due (12 hours)
// Returns the sync client for further operations, or nil if not logged in
func MaybeSyncCLI(dbConn *db.DB, forceSync bool) *sync.Client {
	client, err := sync.NewClient()
	if err != nil || !client.IsLoggedIn() {
		return nil
	}

	shouldSync := forceSync || client.ShouldAutoSync()

	if shouldSync {
		fmt.Println("Syncing...")
		result, err := client.Sync(dbConn, sync.SyncModeMerge)
		if err != nil {
			fmt.Printf("Sync failed: %v\n", err)
		} else {
			_ = client.UpdateSyncTime()
			if result.Pushed > 0 || result.Pulled > 0 {
				fmt.Printf("[OK] Synced (↑%d ↓%d)\n", result.Pushed, result.Pulled)
			} else {
				fmt.Println("[OK] Already up to date")
			}
		}
	}

	return client
}

// MaybeSyncAfterChange syncs after a write operation if --sync flag is set or auto-sync is due
func MaybeSyncAfterChange(dbConn *db.DB, forceSync bool) {
	client, err := sync.NewClient()
	if err != nil || !client.IsLoggedIn() {
		return
	}

	shouldSync := forceSync || client.ShouldAutoSync()

	if shouldSync {
		fmt.Println("Syncing changes...")
		result, err := client.Sync(dbConn, sync.SyncModeMerge)
		if err != nil {
			fmt.Printf("Sync failed: %v\n", err)
		} else {
			_ = client.UpdateSyncTime()
			fmt.Printf("[OK] Synced (↑%d ↓%d)\n", result.Pushed, result.Pulled)
		}
	}
}
