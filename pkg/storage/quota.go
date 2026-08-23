package storage

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

type rawQuotaWindow struct {
	usedPercent   *float64
	resetSeconds  *int
	windowMinutes *int
}

// UpdateCodexQuota stores the official Codex 5-hour and 7-day quota snapshot.
func (s *Store) UpdateCodexQuota(authID string, headers http.Header, observedAt time.Time) error {
	primary := parseRawQuotaWindow(headers, "primary")
	secondary := parseRawQuotaWindow(headers, "secondary")
	if primary.empty() && secondary.empty() {
		return nil
	}

	fiveHour, sevenDay := normalizeQuotaWindows(primary, secondary)
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	if authID == "" {
		authID = "default-auth"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketQuota)
		var snapshot QuotaSnapshot
		if value := bucket.Get([]byte(authID)); value != nil {
			_ = json.Unmarshal(value, &snapshot)
		}

		snapshot.FiveHour = mergeQuotaWindow(snapshot.FiveHour, fiveHour, observedAt)
		snapshot.SevenDay = mergeQuotaWindow(snapshot.SevenDay, sevenDay, observedAt)
		raw, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(authID), raw)
	})
}

func parseRawQuotaWindow(headers http.Header, name string) rawQuotaWindow {
	prefix := "x-codex-" + name + "-"
	return rawQuotaWindow{
		usedPercent:   parseHeaderFloat(headers, prefix+"used-percent"),
		resetSeconds:  parseHeaderInt(headers, prefix+"reset-after-seconds"),
		windowMinutes: parseHeaderInt(headers, prefix+"window-minutes"),
	}
}

func normalizeQuotaWindows(primary, secondary rawQuotaWindow) (rawQuotaWindow, rawQuotaWindow) {
	primaryIsFiveHour := false
	primaryIsSevenDay := false

	switch {
	case primary.windowMinutes != nil && secondary.windowMinutes != nil:
		if *primary.windowMinutes < *secondary.windowMinutes {
			primaryIsFiveHour = true
		} else {
			primaryIsSevenDay = true
		}
	case primary.windowMinutes != nil:
		if *primary.windowMinutes <= 360 {
			primaryIsFiveHour = true
		} else {
			primaryIsSevenDay = true
		}
	case secondary.windowMinutes != nil:
		if *secondary.windowMinutes <= 360 {
			primaryIsSevenDay = true
		} else {
			primaryIsFiveHour = true
		}
	default:
		primaryIsSevenDay = true
	}

	if primaryIsFiveHour {
		return primary, secondary
	}
	if primaryIsSevenDay {
		return secondary, primary
	}
	return rawQuotaWindow{}, rawQuotaWindow{}
}

func mergeQuotaWindow(existing *QuotaWindow, update rawQuotaWindow, observedAt time.Time) *QuotaWindow {
	if update.empty() {
		return existing
	}
	if existing == nil {
		existing = &QuotaWindow{}
	}
	if update.usedPercent != nil {
		existing.UsedPercent = *update.usedPercent
	}
	if update.windowMinutes != nil {
		existing.WindowMinutes = *update.windowMinutes
	}
	if update.resetSeconds != nil {
		seconds := *update.resetSeconds
		if seconds < 0 {
			seconds = 0
		}
		resetAt := observedAt.Add(time.Duration(seconds) * time.Second)
		existing.ResetAt = &resetAt
	}
	existing.UpdatedAt = observedAt
	return existing
}

func (w rawQuotaWindow) empty() bool {
	return w.usedPercent == nil && w.resetSeconds == nil && w.windowMinutes == nil
}

func parseHeaderFloat(headers http.Header, key string) *float64 {
	value := headerValue(headers, key)
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseHeaderInt(headers http.Header, key string) *int {
	value := headerValue(headers, key)
	if value == "" {
		return nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	return &parsed
}

func headerValue(headers http.Header, key string) string {
	if value := headers.Get(key); value != "" {
		return value
	}
	for candidate, values := range headers {
		if strings.EqualFold(candidate, key) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}
