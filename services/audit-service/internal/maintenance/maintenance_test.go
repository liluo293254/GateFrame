package maintenance

import (
	"testing"
	"time"
)

func TestParsePartitionMonth(t *testing.T) {
	t.Parallel()
	m, ok := parsePartitionMonth("audit_events_2026_06")
	if !ok {
		t.Fatal("expected ok")
	}
	if m.Year() != 2026 || m.Month() != time.June {
		t.Fatalf("unexpected month: %v", m)
	}
	if _, ok := parsePartitionMonth("audit_events_legacy"); ok {
		t.Fatal("legacy table name should not parse")
	}
}
