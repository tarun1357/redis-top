package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tarun1357/redis-top/internal/analyzer"
)

// PrintJSON writes the report as formatted JSON.
func PrintJSON(w io.Writer, report analyzer.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// PrintTable renders a human-readable ASCII summary table to w.
func PrintTable(w io.Writer, report analyzer.Report, version string, batchSize int, depth int, sep string) {
	fmt.Fprintf(w, "
redis-top %s — Connected to %s (DB %d)
", version, report.TargetAddr, report.DB)
	fmt.Fprintf(w, "Scanned %s keys in %dms (Batch: %d, Depth: %d, Delimiters: %q)

",
		FormatCount(report.ScannedKeys), report.DurationMs, batchSize, depth, sep)

	header := fmt.Sprintf("%-26s %-9s %-8s %-13s %-8s %-10s %-13s %-10s",
		"PREFIX", "KEYS", "% KEYS", "EST. MEMORY", "% MEM", "AVG SIZE", "NO-TTL (%)", "AVG TTL")
	divider := strings.Repeat("-", len(header))

	fmt.Fprintln(w, header)
	fmt.Fprintln(w, divider)

	for _, st := range report.Prefixes {
		noTTLTag := fmt.Sprintf("%.1f%%", st.NoTTLPercent)
		if st.KeyCount >= 10 && st.NoTTLPercent >= 80.0 {
			noTTLTag += " [!]"
		}

		avgTTLStr := "-"
		if st.AvgTTLSeconds > 0 {
			avgTTLStr = (time.Duration(st.AvgTTLSeconds) * time.Second).String()
		}

		fmt.Fprintf(w, "%-26s %-9s %-7.1f%% %-13s %-7.1f%% %-10s %-13s %-10s
",
			truncate(st.Prefix, 25),
			FormatCount(st.KeyCount),
			st.KeyPercent,
			FormatBytes(st.MemoryBytes),
			st.MemoryPercent,
			FormatBytes(int64(st.AvgBytes)),
			noTTLTag,
			avgTTLStr,
		)
	}

	fmt.Fprintln(w, divider)

	totalNoTTLPct := 0.0
	avgTotalBytes := int64(0)
	if report.ScannedKeys > 0 {
		totalNoTTLPct = (float64(report.TotalNoTTL) / float64(report.ScannedKeys)) * 100.0
		avgTotalBytes = report.TotalMemory / report.ScannedKeys
	}

	fmt.Fprintf(w, "%-26s %-9s %-8s %-13s %-8s %-10s %-13s %-10s
",
		"TOTAL",
		FormatCount(report.ScannedKeys),
		"100.0%",
		FormatBytes(report.TotalMemory),
		"100.0%",
		FormatBytes(avgTotalBytes),
		fmt.Sprintf("%.1f%%", totalNoTTLPct),
		"-",
	)

	if len(report.LeakWarnings) > 0 {
		fmt.Fprintf(w, "
[!] Warning: %d prefix(es) have >=80%% keys without a TTL (potential unbounded memory growth): %s
",
			len(report.LeakWarnings), strings.Join(report.LeakWarnings, ", "))
	}
	fmt.Fprintln(w)
}

// FormatBytes formats raw bytes into human-readable KB/MB/GB strings.
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// FormatCount formats an integer with thousand separators.
func FormatCount(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	rem := len(s) % 3
	if rem > 0 {
		out = append(out, s[:rem]...)
		if len(s) > rem {
			out = append(out, ',')
		}
	}
	for i := rem; i < len(s); i += 3 {
		out = append(out, s[i:i+3]...)
		if i+3 < len(s) {
			out = append(out, ',')
		}
	}
	return string(out)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
