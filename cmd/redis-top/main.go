package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tarun1357/redis-top/internal/analyzer"
	"github.com/tarun1357/redis-top/internal/redis"
	"github.com/tarun1357/redis-top/internal/render"
)

var version = "v0.1.0"

func main() {
	var (
		addr       = flag.String("addr", "127.0.0.1:6379", "Target Redis host:port")
		password   = flag.String("password", os.Getenv("REDIS_PASSWORD"), "Redis password (or REDIS_PASSWORD env)")
		db         = flag.Int("db", 0, "Redis database number")
		match      = flag.String("match", "*", "SCAN glob pattern filter")
		sample     = flag.Int("sample", 5000, "Max keys to sample (0 for unlimited scan)")
		count      = flag.Int("count", 250, "SCAN COUNT hint per batch")
		depth      = flag.Int("depth", 2, "Namespace hierarchy depth")
		sep        = flag.String("sep", ":", "Namespace delimiter characters")
		topN       = flag.Int("top", 15, "Number of top prefixes to show")
		throttleMs = flag.Int("throttle-ms", 0, "Sleep duration in ms between batches")
		jsonOut    = flag.Bool("json", false, "Output report as JSON")
		demo       = flag.Bool("demo", false, "Run using synthetic demo workload (no Redis server needed)")
		showVer    = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("redis-top %s
", version)
		return
	}

	agg := analyzer.NewAggregator(*sep, *depth)
	start := time.Now()

	if *demo {
		populateDemoWorkload(agg)
		report := agg.Summarize("demo.redis.internal:6379", *db, *topN, 142*time.Millisecond)
		outputReport(report, *jsonOut, *count, *depth, *sep)
		return
	}

	client, err := redis.Dial(*addr, *password, *db, 5*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Redis: %v
Tip: Run 'redis-top --demo' to preview output without a live Redis server.
", err)
		os.Exit(1)
	}
	defer client.Close()

	cursor := "0"
	sampled := 0

	for {
		nextCursor, keys, err := client.Scan(cursor, *match, *count)
		if err != nil {
			fmt.Fprintf(os.Stderr, "SCAN failed: %v
", err)
			os.Exit(1)
		}

		if len(keys) > 0 {
			if *sample > 0 && sampled+len(keys) > *sample {
				keys = keys[:*sample-sampled]
			}

			memList, ttlList, err := client.InspectBatch(keys)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Pipeline inspection failed: %v
", err)
				os.Exit(1)
			}

			for i, k := range keys {
				agg.Record(analyzer.KeySample{
					Key:         k,
					MemoryBytes: memList[i],
					TTLSeconds:  ttlList[i],
				})
			}
			sampled += len(keys)
		}

		cursor = nextCursor
		if cursor == "0" || (*sample > 0 && sampled >= *sample) {
			break
		}

		if *throttleMs > 0 {
			time.Sleep(time.Duration(*throttleMs) * time.Millisecond)
		}
	}

	report := agg.Summarize(*addr, *db, *topN, time.Since(start))
	outputReport(report, *jsonOut, *count, *depth, *sep)
}

func outputReport(report analyzer.Report, jsonOut bool, count int, depth int, sep string) {
	if jsonOut {
		if err := render.PrintJSON(os.Stdout, report); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write JSON: %v
", err)
			os.Exit(1)
		}
		return
	}
	render.PrintTable(os.Stdout, report, version, count, depth, sep)
}

func populateDemoWorkload(agg *analyzer.Aggregator) {
	// Realistic production Redis memory distribution
	for i := 0; i < 4120; i++ {
		agg.Record(analyzer.KeySample{
			Key:         fmt.Sprintf("session:auth:usr_%d", i),
			MemoryBytes: 8008,
			TTLSeconds:  2892,
		})
	}
	for i := 0; i < 2890; i++ {
		ttl := int64(-1)
		if i%18 == 0 {
			ttl = 720
		}
		agg.Record(analyzer.KeySample{
			Key:         fmt.Sprintf("cache:vehicle_pass:corp_%d", i),
			MemoryBytes: 6864,
			TTLSeconds:  ttl,
		})
	}
	for i := 0; i < 1940; i++ {
		agg.Record(analyzer.KeySample{
			Key:         fmt.Sprintf("ratelimit:api:ip_%d", i),
			MemoryBytes: 1690,
			TTLSeconds:  45,
		})
	}
	for i := 0; i < 640; i++ {
		agg.Record(analyzer.KeySample{
			Key:         fmt.Sprintf("queue:dlq:msg_%d", i),
			MemoryBytes: 16120,
			TTLSeconds:  -1,
		})
	}
	for i := 0; i < 410; i++ {
		ttl := int64(22500)
		if i%8 == 0 {
			ttl = -1
		}
		agg.Record(analyzer.KeySample{
			Key:         fmt.Sprintf("user:profile:id_%d", i),
			MemoryBytes: 2660,
			TTLSeconds:  ttl,
		})
	}
}
