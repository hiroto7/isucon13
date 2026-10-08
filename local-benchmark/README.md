# Local ISUCON13 optimization

Final reboot-verified247702 / baseline median7595 =32.61x; best248839. Exploration complete;100x goal not reached. [Japanese report](REPORT.md), [complete history](score-all-trials.png), [all46 records](all-trials.csv), [state](STATE.md), [deploy/recovery](DEPLOY.md).

Official source is pinned in upstream-commit.txt. VM-only adaptations are in cloud-init files.
Initial score is the median of 3 valid unoptimized runs. No failed score counts.
All runs and preparation failures are retained in results/. Graphs are regenerated from those records.

Profiling: tools/run_trial.py STAGE --ip APP_IP --cpu-profile samples Go CPU for 50s after official load starts and saves heap allocations. pprof binds only to app VM loopback 127.0.0.1:6060. Analyze with go tool pprof -top cpu.pprof and -top -alloc_space heap.pprof. HTTP timings, MySQL/schema, vmstat, and pidstat are retained. Profiler overhead is tagged; accepted score is verified without CPU sampling.

MySQL profiling: --db-profile sets new app/DNS sessions long_query_time=0, enables FILE slow logging when official load begins, and samples mysqld CPU for 50s with perf cpu-clock/DWARF. It saves SQL frequency/time/rows via pt-query-digest and CPU symbols via perf report. Settings and service sessions are restored after collection. Diagnostic overhead means this run is not an adoption comparison. Raw slow log/perf data remain local; summaries are retained in Git.
