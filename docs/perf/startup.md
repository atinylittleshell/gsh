# Startup profiling (2026-01-04)

## Method
- `BenchmarkStartupInitialization` now reuses a single temp `$HOME` so we measure steady-state startup after the initial setup, while still resetting cached paths between iterations. Telemetry is disabled to avoid network noise.【F:cmd/gsh/startup_benchmark_test.go†L8-L59】
- Profiling command:\
  `GOMAXPROCS=2 go test ./cmd/gsh -run ^$ -bench BenchmarkStartupInitialization -benchmem -count=1 -cpuprofile /tmp/startup.cpu`.
- CPU profile inspected via `go tool pprof -top /tmp/startup.cpu`.【3c7420†L1-L120】

## Results
- Benchmark throughput: **~1.58ms/op**, **503KB allocations/op**, **763 allocs/op** once the initial migration is done.
- CPU time is concentrated in history initialization (SQLite open plus the `HasTable` check) and the associated allocator work; logger setup is a secondary contributor.【3c7420†L1-L120】
- Repeated migrations are avoided by recording a history schema version marker and skipping `AutoMigrate` when the marker/table is already present.【F:internal/history/history.go†L33-L105】

## Bottlenecks and opportunities
- **SQLite cold-path costs** remain the hot spots: opening the history database and running `HasTable` still dominate samples. Keeping the history handle alive or deferring creation until first history access would shave more off steady-state startup.【3c7420†L1-L120】
- If additional schemas are introduced, incrementing the history schema version (and migrating once) preserves the no-repeat migration behavior while allowing upgrades.【F:internal/history/history.go†L33-L105】
