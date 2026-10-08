# Local ISUCON13 optimization

Official source is pinned in upstream-commit.txt. VM-only adaptations are in cloud-init files.
Initial score is the median of 3 valid unoptimized runs. No failed score counts.
All runs and preparation failures are retained in results/. Graphs are regenerated from those records.

Profiling: tools/run_trial.py STAGE --ip APP_IP --cpu-profile samples Go CPU for 50s after official load starts and saves heap allocations. pprof binds only to app VM loopback 127.0.0.1:6060. Analyze with go tool pprof -top cpu.pprof and -top -alloc_space heap.pprof. HTTP timings, MySQL/schema, vmstat, and pidstat are retained. Profiler overhead is tagged; accepted score is verified without CPU sampling.
