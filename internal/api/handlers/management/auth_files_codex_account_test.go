package management

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

func TestBuildAuthFileEntry_ExposesCodexPlanAndQuota(t *testing.T) {
	recoverAt := time.Now().Add(90 * time.Minute).UTC().Truncate(time.Second)
	weeklyRecoverAt := time.Now().Add(72 * time.Hour).UTC().Truncate(time.Second)
	auth := &coreauth.Auth{
		ID:       "codex-user.json",
		FileName: "codex-user.json",
		Provider: "codex",
		Attributes: map[string]string{
			"path":      "/tmp/codex-user.json",
			"plan_type": "pro",
		},
		Metadata: map[string]any{
			"codex_quota_5h_limit":         float64(100),
			"codex_quota_5h_remaining":     float64(12),
			"codex_quota_5h_reset_at":      recoverAt.Format(time.RFC3339),
			"codex_quota_weekly_limit":     "1000",
			"codex_quota_weekly_remaining": "700",
			"codex_quota_weekly_reset_at":  weeklyRecoverAt.Unix(),
		},
		Quota: coreauth.QuotaState{
			Exceeded:      true,
			Reason:        "quota",
			NextRecoverAt: recoverAt,
			BackoffLevel:  2,
		},
	}

	entry := (&Handler{}).buildAuthFileEntry(auth)
	if entry == nil {
		t.Fatal("expected auth entry")
	}

	if got := entry["plan_type"]; got != "pro" {
		t.Fatalf("plan_type = %#v, want %q", got, "pro")
	}
	if got := entry["quota_5h_amount"]; got != "12 / 100 remaining" {
		t.Fatalf("quota_5h_amount = %#v, want %q", got, "12 / 100 remaining")
	}
	if got := entry["quota_5h_remaining"]; got != "12" {
		t.Fatalf("quota_5h_remaining = %#v, want %q", got, "12")
	}
	if got := entry["quota_5h_limit"]; got != "100" {
		t.Fatalf("quota_5h_limit = %#v, want %q", got, "100")
	}
	if got := entry["quota_5h_next_recover_at"]; got != recoverAt {
		t.Fatalf("quota_5h_next_recover_at = %#v, want %#v", got, recoverAt)
	}
	if got := entry["quota_weekly_amount"]; got != "700 / 1000 remaining" {
		t.Fatalf("quota_weekly_amount = %#v, want %q", got, "700 / 1000 remaining")
	}
	if got := entry["quota_weekly_next_recover_at"]; got != weeklyRecoverAt {
		t.Fatalf("quota_weekly_next_recover_at = %#v, want %#v", got, weeklyRecoverAt)
	}
	if got := entry["quota_status"]; got != "recovering" {
		t.Fatalf("quota_status = %#v, want %q", got, "recovering")
	}
	if got := entry["quota_reason"]; got != "quota" {
		t.Fatalf("quota_reason = %#v, want %q", got, "quota")
	}
	if got := entry["quota_next_recover_at"]; got != recoverAt {
		t.Fatalf("quota_next_recover_at = %#v, want %#v", got, recoverAt)
	}
	seconds, ok := entry["quota_recover_after_seconds"].(int64)
	if !ok || seconds <= 0 {
		t.Fatalf("quota_recover_after_seconds = %#v, want positive int64", entry["quota_recover_after_seconds"])
	}
	if got, ok := entry["quota_recover_in"].(string); !ok || got == "" {
		t.Fatalf("quota_recover_in = %#v, want non-empty string", entry["quota_recover_in"])
	}
	if got := entry["quota_backoff_level"]; got != 2 {
		t.Fatalf("quota_backoff_level = %#v, want %d", got, 2)
	}
}

func TestBuildAuthFileEntry_PreservesCooldownWhenQuotaWindowMetadataIsPartial(t *testing.T) {
	recoverAt := time.Now().Add(90 * time.Minute).UTC().Truncate(time.Second)
	auth := &coreauth.Auth{
		ID:       "codex-user.json",
		FileName: "codex-user.json",
		Provider: "codex",
		Attributes: map[string]string{
			"path":      "/tmp/codex-user.json",
			"plan_type": "plus",
		},
		Metadata: map[string]any{
			"codex_quota_5h_limit":     float64(100),
			"codex_quota_5h_remaining": float64(0),
		},
		Quota: coreauth.QuotaState{
			Exceeded:      true,
			Reason:        "quota",
			NextRecoverAt: recoverAt,
		},
	}

	entry := (&Handler{}).buildAuthFileEntry(auth)
	if got := entry["quota_5h_amount"]; got != "0 / 100 remaining" {
		t.Fatalf("quota_5h_amount = %#v, want %q", got, "0 / 100 remaining")
	}
	if got := entry["quota_5h_status"]; got != "recovering" {
		t.Fatalf("quota_5h_status = %#v, want %q", got, "recovering")
	}
	if got := entry["quota_5h_next_recover_at"]; got != recoverAt {
		t.Fatalf("quota_5h_next_recover_at = %#v, want %#v", got, recoverAt)
	}
	if got, ok := entry["quota_5h_recover_in"].(string); !ok || got == "" {
		t.Fatalf("quota_5h_recover_in = %#v, want non-empty string", entry["quota_5h_recover_in"])
	}
}

func TestBuildAuthFileEntry_ExposesCodexQuotaForDisabledAuth(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "codex-disabled.json")
	if err := os.WriteFile(authPath, []byte(`{"type":"codex"}`), 0o600); err != nil {
		t.Fatalf("failed to write auth file: %v", err)
	}

	auth := &coreauth.Auth{
		ID:       "codex-disabled.json",
		FileName: "codex-disabled.json",
		Provider: "codex",
		Disabled: true,
		Status:   coreauth.StatusDisabled,
		Attributes: map[string]string{
			"path":      authPath,
			"plan_type": "plus",
		},
		Metadata: map[string]any{
			"codex_quota_5h_limit":         float64(100),
			"codex_quota_5h_remaining":     float64(42),
			"codex_quota_weekly_limit":     float64(1000),
			"codex_quota_weekly_remaining": float64(900),
		},
	}

	entry := (&Handler{}).buildAuthFileEntry(auth)
	if entry == nil {
		t.Fatal("expected disabled auth entry to remain visible")
	}
	if got := entry["disabled"]; got != true {
		t.Fatalf("disabled = %#v, want true", got)
	}
	if got := entry["plan_type"]; got != "plus" {
		t.Fatalf("plan_type = %#v, want %q", got, "plus")
	}
	if got := entry["quota_5h_amount"]; got != "42 / 100 remaining" {
		t.Fatalf("quota_5h_amount = %#v, want %q", got, "42 / 100 remaining")
	}
	if got := entry["quota_weekly_amount"]; got != "900 / 1000 remaining" {
		t.Fatalf("quota_weekly_amount = %#v, want %q", got, "900 / 1000 remaining")
	}
}
