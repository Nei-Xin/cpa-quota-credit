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
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	sevenDayStart := now.Add(-7 * 24 * time.Hour)

	stats := &FullStats{
		Keys:       make([]KeyStat, 0),
		Auths:      make([]AuthStat, 0),
		Models:     make([]ModelStat, 0),
		Daily:      make([]DailyStat, 0),
		RecentLogs: make([]Record, 0),
	}

	// Memory maps to accumulate sub2api window stats
	todaySummary := &WindowStat{}
	sevenDaySummary := &WindowStat{}

	authToday := make(map[string]*WindowStat)
	authSevenDay := make(map[string]*WindowStat)

	keyToday := make(map[string]*WindowStat)
	keySevenDay := make(map[string]*WindowStat)

	modelToday := make(map[string]*WindowStat)
	modelSevenDay := make(map[string]*WindowStat)

	err := s.db.View(func(tx *bolt.Tx) error {
		// 1. Global summary
		bGlobal := tx.Bucket(bucketGlobal)
		if v := bGlobal.Get([]byte("summary")); v != nil {
			_ = json.Unmarshal(v, &stats.Summary)
		}

		// 2. Client / User Keys
		bKey := tx.Bucket(bucketKeyStats)
		_ = bKey.ForEach(func(k, v []byte) error {
			var ks KeyStat
			if err := json.Unmarshal(v, &ks); err == nil {
				stats.Keys = append(stats.Keys, ks)
			}
			return nil
		})

		// 3. Upstream Auth / Accounts
		bAuth := tx.Bucket(bucketAuthStats)
		_ = bAuth.ForEach(func(k, v []byte) error {
			var as AuthStat
			if err := json.Unmarshal(v, &as); err == nil {
				stats.Auths = append(stats.Auths, as)
			}
			return nil
		})

		// 4. Models
		bModel := tx.Bucket(bucketModelStats)
		_ = bModel.ForEach(func(k, v []byte) error {
			var ms ModelStat
			if err := json.Unmarshal(v, &ms); err == nil {
				stats.Models = append(stats.Models, ms)
			}
			return nil
		})

		// 5. Daily stats
		bDaily := tx.Bucket(bucketDailyStats)
		_ = bDaily.ForEach(func(k, v []byte) error {
			var ds DailyStat
			if err := json.Unmarshal(v, &ds); err == nil {
				stats.Daily = append(stats.Daily, ds)
			}
			return nil
		})

		// 6. Scan logs for WindowStats (Today & 7-Day rolling sub2api window) + RecentLogs
		bLogs := tx.Bucket(bucketUsageLogs)
		c := bLogs.Cursor()
		countRecent := 0

		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var r Record
			if err := json.Unmarshal(v, &r); err != nil {
				continue
			}

			if countRecent < limitRecent {
				stats.RecentLogs = append(stats.RecentLogs, r)
				countRecent++
			}

			tokens := int64(0)
			actualCost := 0.0
			userCost := 0.0
			if r.Cost != nil {
				tokens = r.Cost.TotalTokens
				actualCost = r.Cost.ActualCost
				userCost = r.Cost.UserCost
			}

			// Check 7-Day window
			if r.Timestamp.After(sevenDayStart) || r.Timestamp.Equal(sevenDayStart) {
				sevenDaySummary.TotalRequests++
				sevenDaySummary.TotalTokens += tokens
				sevenDaySummary.ActualCost = billing.QuantizeAmount(sevenDaySummary.ActualCost + actualCost)
				sevenDaySummary.UserCost = billing.QuantizeAmount(sevenDaySummary.UserCost + userCost)

				// Auth 7-day
				aid := r.AuthID
				if aid == "" {
					aid = "default-auth"
				}
				if authSevenDay[aid] == nil {
					authSevenDay[aid] = &WindowStat{}
				}
				authSevenDay[aid].TotalRequests++
				authSevenDay[aid].TotalTokens += tokens
				authSevenDay[aid].ActualCost = billing.QuantizeAmount(authSevenDay[aid].ActualCost + actualCost)
				authSevenDay[aid].UserCost = billing.QuantizeAmount(authSevenDay[aid].UserCost + userCost)

				// Key 7-day
				kid := r.APIKey
				if kid == "" {
					kid = "default-anonymous"
				}
				if keySevenDay[kid] == nil {
					keySevenDay[kid] = &WindowStat{}
				}
				keySevenDay[kid].TotalRequests++
				keySevenDay[kid].TotalTokens += tokens
				keySevenDay[kid].ActualCost = billing.QuantizeAmount(keySevenDay[kid].ActualCost + actualCost)
				keySevenDay[kid].UserCost = billing.QuantizeAmount(keySevenDay[kid].UserCost + userCost)

				// Model 7-day
				mid := r.Model
				if mid == "" {
					mid = "unknown"
				}
				if modelSevenDay[mid] == nil {
					modelSevenDay[mid] = &WindowStat{}
				}
				modelSevenDay[mid].TotalRequests++
				modelSevenDay[mid].TotalTokens += tokens
				modelSevenDay[mid].ActualCost = billing.QuantizeAmount(modelSevenDay[mid].ActualCost + actualCost)
				modelSevenDay[mid].UserCost = billing.QuantizeAmount(modelSevenDay[mid].UserCost + userCost)
			} else {
				// Since cursor is in reverse order, if timestamp is older than 7 days, we can stop window calculation
				if countRecent >= limitRecent {
					break
				}
			}

			// Check Today window
			if r.Timestamp.After(todayStart) || r.Timestamp.Equal(todayStart) {
				todaySummary.TotalRequests++
				todaySummary.TotalTokens += tokens
				todaySummary.ActualCost = billing.QuantizeAmount(todaySummary.ActualCost + actualCost)
				todaySummary.UserCost = billing.QuantizeAmount(todaySummary.UserCost + userCost)

				aid := r.AuthID
				if aid == "" {
					aid = "default-auth"
				}
				if authToday[aid] == nil {
					authToday[aid] = &WindowStat{}
				}
				authToday[aid].TotalRequests++
				authToday[aid].TotalTokens += tokens
				authToday[aid].ActualCost = billing.QuantizeAmount(authToday[aid].ActualCost + actualCost)
				authToday[aid].UserCost = billing.QuantizeAmount(authToday[aid].UserCost + userCost)

				kid := r.APIKey
				if kid == "" {
					kid = "default-anonymous"
				}
				if keyToday[kid] == nil {
					keyToday[kid] = &WindowStat{}
				}
				keyToday[kid].TotalRequests++
				keyToday[kid].TotalTokens += tokens
				keyToday[kid].ActualCost = billing.QuantizeAmount(keyToday[kid].ActualCost + actualCost)
				keyToday[kid].UserCost = billing.QuantizeAmount(keyToday[kid].UserCost + userCost)

				mid := r.Model
				if mid == "" {
					mid = "unknown"
				}
				if modelToday[mid] == nil {
					modelToday[mid] = &WindowStat{}
				}
				modelToday[mid].TotalRequests++
				modelToday[mid].TotalTokens += tokens
				modelToday[mid].ActualCost = billing.QuantizeAmount(modelToday[mid].ActualCost + actualCost)
				modelToday[mid].UserCost = billing.QuantizeAmount(modelToday[mid].UserCost + userCost)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Attach Window Stats to Summary
	stats.Summary.Today = todaySummary
	stats.Summary.SevenDay = sevenDaySummary

	// Attach Window Stats to Auths
	for i := range stats.Auths {
		aid := stats.Auths[i].AuthID
		if t, ok := authToday[aid]; ok {
			stats.Auths[i].Today = t
		} else {
			stats.Auths[i].Today = &WindowStat{}
		}
		if s7, ok := authSevenDay[aid]; ok {
			stats.Auths[i].SevenDay = s7
		} else {
			stats.Auths[i].SevenDay = &WindowStat{}
		}
	}

	// Attach Window Stats to Keys
	for i := range stats.Keys {
		kid := stats.Keys[i].APIKey
		if t, ok := keyToday[kid]; ok {
			stats.Keys[i].Today = t
		} else {
			stats.Keys[i].Today = &WindowStat{}
		}
		if s7, ok := keySevenDay[kid]; ok {
			stats.Keys[i].SevenDay = s7
		} else {
			stats.Keys[i].SevenDay = &WindowStat{}
		}
	}

	// Attach Window Stats to Models
	for i := range stats.Models {
		mid := stats.Models[i].Model
		if t, ok := modelToday[mid]; ok {
			stats.Models[i].Today = t
		} else {
			stats.Models[i].Today = &WindowStat{}
		}
		if s7, ok := modelSevenDay[mid]; ok {
			stats.Models[i].SevenDay = s7
		} else {
			stats.Models[i].SevenDay = &WindowStat{}
		}
	}

	// Sort Keys by UserCost desc
	sort.Slice(stats.Keys, func(i, j int) bool {
		return stats.Keys[i].UserCost > stats.Keys[j].UserCost
	})

	// Sort Auths by ActualCost desc
	sort.Slice(stats.Auths, func(i, j int) bool {
		return stats.Auths[i].ActualCost > stats.Auths[j].ActualCost
	})

	// Sort Models by TotalTokens desc
	sort.Slice(stats.Models, func(i, j int) bool {
		return stats.Models[i].TotalTokens > stats.Models[j].TotalTokens
	})

	// Sort Daily by Date asc
	sort.Slice(stats.Daily, func(i, j int) bool {
		return stats.Daily[i].Date < stats.Daily[j].Date
	})

	return stats, nil
}
