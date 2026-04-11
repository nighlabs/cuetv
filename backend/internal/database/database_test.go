package database

import (
	"testing"
)

func TestNewInMemory(t *testing.T) {
	db, err := NewInMemory()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	// Verify tables exist
	tables := []string{"sessions", "room_config", "queue_items"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}
