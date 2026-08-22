package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/router-for-me/cpa-quota-credit/pkg/billing"
)

func TestStore_RecordAndAggregate(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer func() {
		_ = store.Close()
		_ = os.Remove(dbPath)
	}()

	// 1. Record 2 requests
	rec1 := Record{
		APIKey:   "sk-user-1",
		Provider: "anthropic",
		Model:    "claude-3-5-sonnet",
		Cost: &billing.CostBreakdown{
			TotalTokens: 1000,
			ActualCost:  0.01,
			UserCost:    0.015,
		},
		Timestamp: time.Now(),
	}
	if err := store.RecordUsage(rec1); err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}

	rec2 := Record{
		APIKey:   "sk-user-2",
		Provider: "openai",
		Model:    "gpt-4o",
		Cost: &billing.CostBreakdown{
			TotalTokens: 2000,
			ActualCost:  0.02,
			UserCost:    0.025,
		},
		Timestamp: time.Now(),
	}
	if err := store.RecordUsage(rec2); err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}

	// 2. Fetch full stats
	stats, err := store.GetFullStats(10)
	if err != nil {
		t.Fatalf("GetFullStats failed: %v", err)
	}

	if stats.Summary.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2", stats.Summary.TotalRequests)
	}
	if stats.Summary.TotalTokens != 3000 {
		t.Errorf("TotalTokens = %d, want 3000", stats.Summary.TotalTokens)
	}
	if stats.Summary.ActualCost != 0.03 {
		t.Errorf("ActualCost = %v, want 0.03", stats.Summary.ActualCost)
	}
	if stats.Summary.UserCost != 0.04 {
		t.Errorf("UserCost = %v, want 0.04", stats.Summary.UserCost)
	}
	if len(stats.Keys) != 2 {
		t.Errorf("len(Keys) = %d, want 2", len(stats.Keys))
	}
}
