package analyzer

import (
	"testing"
	"time"
)

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		delimiters string
		depth      int
		want       string
	}{
		{
			name:       "depth 2 standard colon",
			key:        "session:auth:user:98123",
			delimiters: ":",
			depth:      2,
			want:       "session:auth:*",
		},
		{
			name:       "depth 1 standard colon",
			key:        "session:auth:user:98123",
			delimiters: ":",
			depth:      1,
			want:       "session:*",
		},
		{
			name:       "key shorter than depth",
			key:        "cache:token123",
			delimiters: ":",
			depth:      3,
			want:       "cache:*",
		},
		{
			name:       "no delimiter in key",
			key:        "singleton_lock_key",
			delimiters: ":",
			depth:      2,
			want:       "<no-prefix>",
		},
		{
			name:       "multiple delimiters",
			key:        "service/valet:pass:1049",
			delimiters: ":/",
			depth:      2,
			want:       "service:valet:*",
		},
		{
			name:       "empty key",
			key:        "",
			delimiters: ":",
			depth:      2,
			want:       "<empty>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractPrefix(tc.key, tc.delimiters, tc.depth)
			if got != tc.want {
				t.Errorf("ExtractPrefix(%q, %q, %d) = %q; want %q", tc.key, tc.delimiters, tc.depth, got, tc.want)
			}
		})
	}
}

func TestAggregatorSummarize(t *testing.T) {
	agg := NewAggregator(":", 2)

	// Record 15 keys under cache:vehicle_pass:* without TTL (should trigger leak warning)
	for i := 0; i < 15; i++ {
		agg.Record(KeySample{
			Key:         "cache:vehicle_pass:id_" + string(rune('a'+i)),
			MemoryBytes: 2048,
			TTLSeconds:  -1,
		})
	}

	// Record 5 keys under session:auth:* with valid TTL
	for i := 0; i < 5; i++ {
		agg.Record(KeySample{
			Key:         "session:auth:token_" + string(rune('a'+i)),
			MemoryBytes: 1024,
			TTLSeconds:  3600,
		})
	}

	report := agg.Summarize("127.0.0.1:6379", 0, 10, 45*time.Millisecond)

	if report.ScannedKeys != 20 {
		t.Fatalf("expected 20 scanned keys, got %d", report.ScannedKeys)
	}

	expectedMem := int64(15*2048 + 5*1024)
	if report.TotalMemory != expectedMem {
		t.Fatalf("expected total memory %d, got %d", expectedMem, report.TotalMemory)
	}

	if len(report.Prefixes) != 2 {
		t.Fatalf("expected 2 prefixes, got %d", len(report.Prefixes))
	}

	// First prefix should be cache:vehicle_pass:* because it has higher memory
	top := report.Prefixes[0]
	if top.Prefix != "cache:vehicle_pass:*" {
		t.Errorf("expected top prefix 'cache:vehicle_pass:*', got %q", top.Prefix)
	}
	if top.NoTTLPercent != 100.0 {
		t.Errorf("expected 100%% NoTTL for top prefix, got %.1f%%", top.NoTTLPercent)
	}

	// Leak warning should flag cache:vehicle_pass:*
	if len(report.LeakWarnings) != 1 || report.LeakWarnings[0] != "cache:vehicle_pass:*" {
		t.Errorf("expected leak warning for 'cache:vehicle_pass:*', got %v", report.LeakWarnings)
	}
}
