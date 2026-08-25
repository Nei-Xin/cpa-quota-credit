package storage

import (
	"testing"
	"time"
)

func TestQuotaWindowStart(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	fallback := 5 * time.Hour
	futureReset := now.Add(2 * time.Hour)
	pastReset := now.Add(-30 * time.Minute)

	tests := []struct {
		name   string
		window *QuotaWindow
		want   time.Time
	}{
		{
			name:   "active official window",
			window: &QuotaWindow{ResetAt: &futureReset},
			want:   futureReset.Add(-fallback),
		},
		{
			name:   "official window just reset",
			window: &QuotaWindow{ResetAt: &pastReset},
			want:   pastReset,
		},
		{
			name: "no official window",
			want: now.Add(-fallback),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quotaWindowStart(tt.window, fallback, now); !got.Equal(tt.want) {
				t.Fatalf("quotaWindowStart() = %v, want %v", got, tt.want)
			}
		})
	}
}
