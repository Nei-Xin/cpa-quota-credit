package storage

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/router-for-me/cpa-quota-credit/pkg/billing"
	bolt "go.etcd.io/bbolt"
)

type FullStats struct {
	Summary    GlobalSummary `json:"summary"`
	Keys       []KeyStat     `json:"keys"`
	Auths      []AuthStat    `json:"auths"`
	Models     []ModelStat   `json:"models"`
	Daily      []DailyStat   `json:"daily"`
	RecentLogs []Record      `json:"recent_logs"`
}

func (s *Store) GetFullStats(limitRecent int) (*FullStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	stats := &FullStats{
		Keys:       make([]KeyStat, 0),
		Auths:      make([]AuthStat, 0),
		Models:     make([]ModelStat, 0),
		Daily:      make([]DailyStat, 0),
		RecentLogs: make([]Record, 0),
	}

	fiveHourSummary := &WindowStat{}
	sevenDaySummary := &WindowStat{}
	authFiveHour := make(map[string]*WindowStat)
	authSevenDay := make(map[string]*WindowStat)
	keyFiveHour := make(map[string]*WindowStat)
	keySevenDay := make(map[string]*WindowStat)
	modelFiveHour := make(map[string]*WindowStat)
	modelSevenDay := make(map[string]*WindowStat)
	quotaSnapshots := make(map[string]*QuotaSnapshot)

	err := s.db.View(func(tx *bolt.Tx) error {
		if value := tx.Bucket(bucketGlobal).Get([]byte("summary")); value != nil {
			_ = json.Unmarshal(value, &stats.Summary)
		}

		_ = tx.Bucket(bucketKeyStats).ForEach(func(_, value []byte) error {
			var item KeyStat
			if json.Unmarshal(value, &item) == nil {
				stats.Keys = append(stats.Keys, item)
			}
			return nil
		})
		_ = tx.Bucket(bucketAuthStats).ForEach(func(_, value []byte) error {
			var item AuthStat
			if json.Unmarshal(value, &item) == nil {
				stats.Auths = append(stats.Auths, item)
			}
			return nil
		})
		_ = tx.Bucket(bucketModelStats).ForEach(func(_, value []byte) error {
			var item ModelStat
			if json.Unmarshal(value, &item) == nil {
				stats.Models = append(stats.Models, item)
			}
			return nil
		})
		_ = tx.Bucket(bucketDailyStats).ForEach(func(_, value []byte) error {
			var item DailyStat
			if json.Unmarshal(value, &item) == nil {
				stats.Daily = append(stats.Daily, item)
			}
			return nil
		})
		_ = tx.Bucket(bucketQuota).ForEach(func(key, value []byte) error {
			var snapshot QuotaSnapshot
			if json.Unmarshal(value, &snapshot) == nil {
				quotaSnapshots[string(key)] = &snapshot
			}
			return nil
		})

		logs := tx.Bucket(bucketUsageLogs)
		cursor := logs.Cursor()
		recentCount := 0
		earliestStart := now.Add(-7 * 24 * time.Hour)
		for _, snapshot := range quotaSnapshots {
			start := quotaWindowStart(snapshot.SevenDay, 7*24*time.Hour, now)
			if start.Before(earliestStart) {
				earliestStart = start
			}
		}

		for key, value := cursor.Last(); key != nil; key, value = cursor.Prev() {
			var record Record
			if json.Unmarshal(value, &record) != nil {
				continue
			}
			if recentCount < limitRecent {
				stats.RecentLogs = append(stats.RecentLogs, record)
				recentCount++
			}
			if record.Timestamp.Before(earliestStart) {
				if recentCount >= limitRecent {
					break
				}
				continue
			}

			authID := normalizedAuthID(record.AuthID)
			apiKey := normalizedAPIKey(record.APIKey)
			model := normalizedModel(record.Model)
			snapshot := quotaSnapshots[authID]
			fiveHourStart := now.Add(-5 * time.Hour)
			sevenDayStart := now.Add(-7 * 24 * time.Hour)
			if snapshot != nil {
				fiveHourStart = quotaWindowStart(snapshot.FiveHour, 5*time.Hour, now)
				sevenDayStart = quotaWindowStart(snapshot.SevenDay, 7*24*time.Hour, now)
			}

			if !record.Timestamp.Before(fiveHourStart) {
				addRecord(fiveHourSummary, record)
				addRecord(windowFor(authFiveHour, authID), record)
				addRecord(windowFor(keyFiveHour, apiKey), record)
				addRecord(windowFor(modelFiveHour, model), record)
			}
			if !record.Timestamp.Before(sevenDayStart) {
				addRecord(sevenDaySummary, record)
				addRecord(windowFor(authSevenDay, authID), record)
				addRecord(windowFor(keySevenDay, apiKey), record)
				addRecord(windowFor(modelSevenDay, model), record)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	stats.Summary.FiveHour = fiveHourSummary
	stats.Summary.SevenDay = sevenDaySummary
	for i := range stats.Auths {
		authID := stats.Auths[i].AuthID
		stats.Auths[i].FiveHour = windowOrEmpty(authFiveHour[authID])
		stats.Auths[i].SevenDay = windowOrEmpty(authSevenDay[authID])
		stats.Auths[i].Quota = visibleQuotaSnapshot(quotaSnapshots[authID], now)
	}
	for i := range stats.Keys {
		apiKey := stats.Keys[i].APIKey
		stats.Keys[i].FiveHour = windowOrEmpty(keyFiveHour[apiKey])
		stats.Keys[i].SevenDay = windowOrEmpty(keySevenDay[apiKey])
	}
	for i := range stats.Models {
		model := stats.Models[i].Model
		stats.Models[i].FiveHour = windowOrEmpty(modelFiveHour[model])
		stats.Models[i].SevenDay = windowOrEmpty(modelSevenDay[model])
	}

	sort.Slice(stats.Keys, func(i, j int) bool { return stats.Keys[i].UserCost > stats.Keys[j].UserCost })
	sort.Slice(stats.Auths, func(i, j int) bool { return stats.Auths[i].ActualCost > stats.Auths[j].ActualCost })
	sort.Slice(stats.Models, func(i, j int) bool { return stats.Models[i].TotalTokens > stats.Models[j].TotalTokens })
	sort.Slice(stats.Daily, func(i, j int) bool { return stats.Daily[i].Date < stats.Daily[j].Date })
	return stats, nil
}

func quotaWindowStart(window *QuotaWindow, fallback time.Duration, now time.Time) time.Time {
	if window != nil && window.ResetAt != nil {
		if now.Before(*window.ResetAt) {
			return window.ResetAt.Add(-fallback)
		}
		return *window.ResetAt
	}
	return now.Add(-fallback)
}

func visibleQuotaSnapshot(snapshot *QuotaSnapshot, now time.Time) *QuotaSnapshot {
	if snapshot == nil {
		return nil
	}
	return &QuotaSnapshot{
		FiveHour: visibleQuotaWindow(snapshot.FiveHour, now),
		SevenDay: visibleQuotaWindow(snapshot.SevenDay, now),
	}
}

func visibleQuotaWindow(window *QuotaWindow, now time.Time) *QuotaWindow {
	if window == nil {
		return nil
	}
	visible := *window
	if visible.ResetAt != nil && !now.Before(*visible.ResetAt) {
		visible.UsedPercent = 0
	}
	return &visible
}

func addRecord(stat *WindowStat, record Record) {
	stat.TotalRequests++
	if record.Cost == nil {
		return
	}
	stat.TotalTokens += record.Cost.TotalTokens
	stat.ActualCost = billing.QuantizeAmount(stat.ActualCost + record.Cost.ActualCost)
	stat.UserCost = billing.QuantizeAmount(stat.UserCost + record.Cost.UserCost)
}

func windowFor(items map[string]*WindowStat, key string) *WindowStat {
	if items[key] == nil {
		items[key] = &WindowStat{}
	}
	return items[key]
}

func windowOrEmpty(stat *WindowStat) *WindowStat {
	if stat == nil {
		return &WindowStat{}
	}
	return stat
}

func normalizedAuthID(value string) string {
	if value == "" {
		return "default-auth"
	}
	return value
}

func normalizedAPIKey(value string) string {
	if value == "" {
		return "default-anonymous"
	}
	return value
}

func normalizedModel(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
