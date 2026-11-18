package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/tarun1357/redis-top/internal/analyzer"
)

func TestFormatBytes(t *testing.T) {
	if got := FormatBytes(512); got != "512 B" {
		t.Errorf("got %q, want 512 B", got)
	}
	if got := FormatBytes(2048); got != "2.00 KB" {
		t.Errorf("got %q, want 2.00 KB", got)
	}
}

func TestFormatCount(t *testing.T) {
	if got := FormatCount(10000); got != "10,000" {
		t.Errorf("got %q, want 10,000", got)
	}
}

func TestPrintTableAndJSON(t *testing.T) {
	agg := analyzer.NewAggregator(":", 2)
	for i := 0; i < 12; i++ {
		agg.Record(analyzer.KeySample{
			Key:         "queue:dlq:item",
			MemoryBytes: 4096,
			TTLSeconds:  -1,
		})
	}
	report := agg.Summarize("127.0.0.1:6379", 0, 10, 25*time.Millisecond)

	var buf bytes.Buffer
	PrintTable(&buf, report, "v0.1.0", 200, 2, ":")
	out := buf.String()
	if !strings.Contains(out, "queue:dlq:*") {
		t.Errorf("expected table output to contain prefix 'queue:dlq:*', got:
%s", out)
	}

	var jsonBuf bytes.Buffer
	if err := PrintJSON(&jsonBuf, report); err != nil {
		t.Fatalf("PrintJSON failed: %v", err)
	}
	if !strings.Contains(jsonBuf.String(), `"prefix": "queue:dlq:*"`) {
		t.Errorf("expected JSON output to contain prefix, got:
%s", jsonBuf.String())
	}
}
