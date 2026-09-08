package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/router-for-me/cpa-quota-credit/pkg/billing"
	bolt "go.etcd.io/bbolt"
)

var (
	bucketUsageLogs  = []byte("usage_logs")
	bucketKeyStats   = []byte("key_stats")
	bucketAuthStats  = []byte("auth_stats")
	bucketModelStats = []byte("model_stats")
	bucketDailyStats = []byte("daily_stats")
	bucketGlobal     = []byte("global_stats")
	bucketQuota      = []byte("quota_snapshots")
)

type Record struct {
	ID        string                 `json:"id"`
	APIKey    string                 `json:"api_key"`
	AuthID    string                 `json:"auth_id"`
	Provider  string                 `json:"provider"`
	Model     string                 `json:"model"`
	Cost      *billing.CostBreakdown `json:"cost"`
	LatencyMs int64                  `json:"latency_ms"`
	Timestamp time.Time              `json:"timestamp"`
	Failed    bool                   `json:"failed"`
}

type WindowStat struct {
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	ActualCost    float64 `json:"actual_cost"`
	UserCost      float64 `json:"user_cost"`
}

type QuotaWindow struct {
	UsedPercent   float64    `json:"used_percent"`
	WindowMinutes int        `json:"window_minutes,omitempty"`
	ResetAt       *time.Time `json:"reset_at,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type QuotaSnapshot struct {
	FiveHour *QuotaWindow `json:"five_hour,omitempty"`
	SevenDay *QuotaWindow `json:"seven_day,omitempty"`
}

type GlobalSummary struct {
	TotalRequests int64       `json:"total_requests"`
	TotalTokens   int64       `json:"total_tokens"`
	ActualCost    float64     `json:"actual_cost"`
	UserCost      float64     `json:"user_cost"`
	FiveHour      *WindowStat `json:"five_hour,omitempty"`
	SevenDay      *WindowStat `json:"seven_day,omitempty"`
	LastUpdated   time.Time   `json:"last_updated"`
}

type KeyStat struct {
	APIKey        string      `json:"api_key"`
	TotalRequests int64       `json:"total_requests"`
	TotalTokens   int64       `json:"total_tokens"`
	ActualCost    float64     `json:"actual_cost"`
	UserCost      float64     `json:"user_cost"`
	FiveHour      *WindowStat `json:"five_hour,omitempty"`
	SevenDay      *WindowStat `json:"seven_day,omitempty"`
	LastActive    time.Time   `json:"last_active"`
}

type AuthStat struct {
	AuthID        string         `json:"auth_id"`
	Provider      string         `json:"provider"`
	TotalRequests int64          `json:"total_requests"`
	TotalTokens   int64          `json:"total_tokens"`
	ActualCost    float64        `json:"actual_cost"`
	UserCost      float64        `json:"user_cost"`
	FiveHour      *WindowStat    `json:"five_hour,omitempty"`
	SevenDay      *WindowStat    `json:"seven_day,omitempty"`
	Quota         *QuotaSnapshot `json:"quota,omitempty"`
	LastActive    time.Time      `json:"last_active"`
}

type ModelStat struct {
	Model         string      `json:"model"`
	Provider      string      `json:"provider"`
	TotalRequests int64       `json:"total_requests"`
	TotalTokens   int64       `json:"total_tokens"`
	ActualCost    float64     `json:"actual_cost"`
	UserCost      float64     `json:"user_cost"`
	FiveHour      *WindowStat `json:"five_hour,omitempty"`
	SevenDay      *WindowStat `json:"seven_day,omitempty"`
}

type DailyStat struct {
	Date          string  `json:"date"` // YYYY-MM-DD
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	ActualCost    float64 `json:"actual_cost"`
	UserCost      float64 `json:"user_cost"`
}

type Store struct {
	dbPath string
	db     *bolt.DB
	mu     sync.RWMutex
}

func NewStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		dbPath = "./data/quota_credit.db"
	}
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open bolt db: %w", err)
	}

	// Initialize buckets
	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketUsageLogs, bucketKeyStats, bucketAuthStats, bucketModelStats, bucketDailyStats, bucketGlobal, bucketQuota} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{
		dbPath: dbPath,
		db:     db,
	}, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) RecordUsage(record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}
	if record.ID == "" {
		record.ID = fmt.Sprintf("%d_%s", record.Timestamp.UnixNano(), record.Model)
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		// 1. Save individual log
		bLogs := tx.Bucket(bucketUsageLogs)
		rawLog, err := json.Marshal(record)
		if err != nil {
			return err
		}
		if err := bLogs.Put([]byte(record.ID), rawLog); err != nil {
			return err
		}

		tokens := record.Cost.TotalTokens
		actualCost := record.Cost.ActualCost
		userCost := record.Cost.UserCost

		// 2. Update Global Stats
		bGlobal := tx.Bucket(bucketGlobal)
		var global GlobalSummary
		if v := bGlobal.Get([]byte("summary")); v != nil {
			_ = json.Unmarshal(v, &global)
		}
		global.TotalRequests++
		global.TotalTokens += tokens
		global.ActualCost = billing.QuantizeAmount(global.ActualCost + actualCost)
		global.UserCost = billing.QuantizeAmount(global.UserCost + userCost)
		global.LastUpdated = record.Timestamp
		rawGlobal, _ := json.Marshal(global)
		if err := bGlobal.Put([]byte("summary"), rawGlobal); err != nil {
			return err
		}

		// 3. Update Client/User Key Stats
		key := record.APIKey
		if key == "" {
			key = "default-anonymous"
		}
		bKey := tx.Bucket(bucketKeyStats)
		var kStat KeyStat
		if v := bKey.Get([]byte(key)); v != nil {
			_ = json.Unmarshal(v, &kStat)
		}
		kStat.APIKey = key
		kStat.TotalRequests++
		kStat.TotalTokens += tokens
		kStat.ActualCost = billing.QuantizeAmount(kStat.ActualCost + actualCost)
		kStat.UserCost = billing.QuantizeAmount(kStat.UserCost + userCost)
		kStat.LastActive = record.Timestamp
		rawKey, _ := json.Marshal(kStat)
		if err := bKey.Put([]byte(key), rawKey); err != nil {
			return err
		}

		// 4. Update Upstream Auth / Account Stats
		authID := record.AuthID
		if authID == "" {
			authID = "default-auth"
		}
		bAuth := tx.Bucket(bucketAuthStats)
		var aStat AuthStat
		if v := bAuth.Get([]byte(authID)); v != nil {
			_ = json.Unmarshal(v, &aStat)
		}
		aStat.AuthID = authID
		aStat.Provider = record.Provider
		aStat.TotalRequests++
		aStat.TotalTokens += tokens
		aStat.ActualCost = billing.QuantizeAmount(aStat.ActualCost + actualCost)
		aStat.UserCost = billing.QuantizeAmount(aStat.UserCost + userCost)
		aStat.LastActive = record.Timestamp
		rawAuth, _ := json.Marshal(aStat)
		if err := bAuth.Put([]byte(authID), rawAuth); err != nil {
			return err
		}

		// 5. Update Model Stats
		model := record.Model
		if model == "" {
			model = "unknown"
		}
		bModel := tx.Bucket(bucketModelStats)
		var mStat ModelStat
		if v := bModel.Get([]byte(model)); v != nil {
			_ = json.Unmarshal(v, &mStat)
		}
		mStat.Model = model
		mStat.Provider = record.Provider
		mStat.TotalRequests++
		mStat.TotalTokens += tokens
		mStat.ActualCost = billing.QuantizeAmount(mStat.ActualCost + actualCost)
		mStat.UserCost = billing.QuantizeAmount(mStat.UserCost + userCost)
		rawModel, _ := json.Marshal(mStat)
		if err := bModel.Put([]byte(model), rawModel); err != nil {
			return err
		}

		// 6. Update Daily Stats (YYYY-MM-DD)
		dateStr := record.Timestamp.Format("2006-01-02")
		bDaily := tx.Bucket(bucketDailyStats)
		var dStat DailyStat
		if v := bDaily.Get([]byte(dateStr)); v != nil {
			_ = json.Unmarshal(v, &dStat)
		}
		dStat.Date = dateStr
		dStat.TotalRequests++
		dStat.TotalTokens += tokens
		dStat.ActualCost = billing.QuantizeAmount(dStat.ActualCost + actualCost)
		dStat.UserCost = billing.QuantizeAmount(dStat.UserCost + userCost)
		rawDaily, _ := json.Marshal(dStat)
		return bDaily.Put([]byte(dateStr), rawDaily)
	})
}
