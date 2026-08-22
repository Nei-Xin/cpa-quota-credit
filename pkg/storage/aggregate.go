package storage

import (
	"encoding/json"
	"sort"

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

	stats := &FullStats{
		Keys:       make([]KeyStat, 0),
		Auths:      make([]AuthStat, 0),
		Models:     make([]ModelStat, 0),
		Daily:      make([]DailyStat, 0),
		RecentLogs: make([]Record, 0),
	}

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

		// 6. Recent Logs
		if limitRecent > 0 {
			bLogs := tx.Bucket(bucketUsageLogs)
			c := bLogs.Cursor()
			count := 0
			for k, v := c.Last(); k != nil && count < limitRecent; k, v = c.Prev() {
				var r Record
				if err := json.Unmarshal(v, &r); err == nil {
					stats.RecentLogs = append(stats.RecentLogs, r)
					count++
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
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
