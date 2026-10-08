# ISUCON13 current state

Environment now has two VM processes only (app 4vCPU/8GiB; benchmark 4vCPU/4GiB).
Old orphan PID 51591 required SIGKILL through macOS admin authentication and is gone.
User prefers unattended work; do not request repeated macOS admin dialogs. Explain and batch any future necessary host recovery first.
Official initial-code valid exploratory run: 9479, excluded from canonical baseline due to extra VM process.
Second attempted baseline failed pretest during resource instability; no valid score.
Next: obtain 3 stable initial-code passes, then prioritize large-work reductions (DNS scans, repeated stats/response construction, moderation).
No performance changes adopted yet. Source hash manifests matched before measurement.
Fork hiroto7/isucon13, branch codex/local-benchmark; local results include every preparation failure and attempt.
Benchmark Go 1.21.2 and official source unchanged; local TLS CA trusted, verification enabled.
Old isucon14 source/config/results backup remains in task workspace/backups. Host repositories retained.
Stop after fresh profile/spec alternatives/public solution review leaves no substantial-gain hypothesis.

Initial-code post-recovery scores: 9415, 7595, 6696; median 7595 (100x goal 759500). Host workload introduces significant variation; compare with initial observed best 9479 as well.
Current candidate: statistics grouped aggregation; not adopted until official score validates.

Profiling: Go pprof listens on 127.0.0.1:6060 in app VM. run_trial --cpu-profile starts 50s CPU sampling only after official load starts and saves CPU/heap files. Profiler runs are flagged; adoption requires a subsequent ordinary run. Current trial moderation-batch-profile tests single SQL spam detection/deletion. DNS index alone did not raise score.
