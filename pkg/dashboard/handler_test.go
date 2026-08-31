package dashboard

import "testing"

func TestAuthIDIsActive(t *testing.T) {
	active := map[string]struct{}{
		"codex-current.json": {},
		"codex-index-1":      {},
	}

	tests := []struct {
		name   string
		authID string
		want   bool
	}{
		{name: "exact name", authID: "codex-current.json", want: true},
		{name: "name without extension", authID: "codex-current", want: true},
		{name: "path name", authID: `auths\\codex-current.json`, want: true},
		{name: "auth index", authID: "codex-index-1", want: true},
		{name: "deleted account", authID: "codex-deleted.json", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := authIDIsActive(tt.authID, active); got != tt.want {
				t.Fatalf("authIDIsActive(%q) = %v, want %v", tt.authID, got, tt.want)
			}
		})
	}
}
