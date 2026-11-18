package analyzer

import (
	"sort"
	"time"
)

// KeySample represents a single sampled Redis key and its inspected metadata.
type KeySample struct {
	Key         string
	MemoryBytes int64
	TTLSeconds  int64 // -1 means no TTL, -2 means key expired/missing
}

// PrefixStat holds aggregated metrics for a single namespace prefix.
type PrefixStat struct {
	Prefix        string  `json:"prefix"`
	KeyCount      int64   `json:"key_count"`
	KeyPercent    float64 `json:"key_percent"`
	MemoryBytes   int64   `json:"memory_bytes"`
	MemoryPercent float64 `json:"memory_percent"`
	AvgBytes      float64 `json:"avg_bytes"`
	NoTTLCount    int64   `json:"no_ttl_count"`
	NoTTLPercent  float64 `json:"no_ttl_percent"`
	AvgTTLSeconds int64   `json:"avg_ttl_seconds"`
	ttlSumSeconds int64
	ttlValidCount int64
}

// Report contains the complete summary of a scan session.
type Report struct {
	TargetAddr   string        `json:"target_addr"`
	DB           int           `json:"db"`
	ScannedKeys  int64         `json:"scanned_keys"`
	TotalMemory  int64         `json:"total_memory_bytes"`
	TotalNoTTL   int64         `json:"total_no_ttl_keys"`
	DurationMs   int64         `json:"duration_ms"`
	Prefixes     []PrefixStat  `json:"prefixes"`
	LeakWarnings []string      `json:"leak_warnings"`
}

// Aggregator accumulates sampled keys into namespace buckets.
type Aggregator struct {
	delimiters string
	depth      int
	buckets    map[string]*PrefixStat
	totalKeys  int64
	totalMem   int64
	totalNoTTL int64
}

// NewAggregator initializes a new prefix aggregator.
func NewAggregator(delimiters string, depth int) *Aggregator {
	return &Aggregator{
		delimiters: delimiters,
		depth:      depth,
		buckets:    make(map[string]*PrefixStat),
	}
}

// Record ingests a single KeySample into its matching namespace bucket.
func (a *Aggregator) Record(sample KeySample) {
	prefix := ExtractPrefix(sample.Key, a.delimiters, a.depth)
	stat, exists := a.buckets[prefix]
	if !exists {
		stat = &PrefixStat{Prefix: prefix}
		a.buckets[prefix] = stat
	}

	mem := sample.MemoryBytes
	if mem < 0 {
		mem = 0
	}

	stat.KeyCount++
	stat.MemoryBytes += mem
	a.totalKeys++
	a.totalMem += mem

	if sample.TTLSeconds == -1 {
		stat.NoTTLCount++
		a.totalNoTTL++
	} else if sample.TTLSeconds > 0 {
		stat.ttlSumSeconds += sample.TTLSeconds
		stat.ttlValidCount++
	}
}

// Summarize computes percentages, sorts prefixes by memory descending, and returns a Report.
func (a *Aggregator) Summarize(addr string, db int, topN int, elapsed time.Duration) Report {
	stats := make([]PrefixStat, 0, len(a.buckets))
	var warnings []string

	for _, st := range a.buckets {
		if a.totalKeys > 0 {
			st.KeyPercent = (float64(st.KeyCount) / float64(a.totalKeys)) * 100.0
		}
		if a.totalMem > 0 {
			st.MemoryPercent = (float64(st.MemoryBytes) / float64(a.totalMem)) * 100.0
		}
		if st.KeyCount > 0 {
			st.AvgBytes = float64(st.MemoryBytes) / float64(st.KeyCount)
			st.NoTTLPercent = (float64(st.NoTTLCount) / float64(st.KeyCount)) * 100.0
		}
		if st.ttlValidCount > 0 {
			st.AvgTTLSeconds = st.ttlSumSeconds / st.ttlValidCount
		}
		stats = append(stats, *st)
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].MemoryBytes == stats[j].MemoryBytes {
			return stats[i].KeyCount > stats[j].KeyCount
		}
		return stats[i].MemoryBytes > stats[j].MemoryBytes
	})

	for _, st := range stats {
		if st.KeyCount >= 10 && st.NoTTLPercent >= 80.0 {
			warnings = append(warnings, st.Prefix)
		}
	}

	if topN > 0 && len(stats) > topN {
		stats = stats[:topN]
	}

	return Report{
		TargetAddr:   addr,
		DB:           db,
		ScannedKeys:  a.totalKeys,
		TotalMemory:  a.totalMem,
		TotalNoTTL:   a.totalNoTTL,
		DurationMs:   elapsed.Milliseconds(),
		Prefixes:     stats,
		LeakWarnings: warnings,
	}
}
