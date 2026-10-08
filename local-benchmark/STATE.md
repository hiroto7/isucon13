# ISUCON13 current state

Accepted score 15589 (initial median 7595: 2.05x; initial best observed 9479: 1.64x). Goal remains 100x.
Accepted bundle: grouped snapshot stats, DNS name/type index, bulk SQL LIKE moderation, relation lookup indexes, icon write-only READ COMMITTED plus owner lock, stream row locking for moderation/posting.
Official full benchmark passed; targeted 64 cross-owner and 32 same-owner icon updates and 10 moderation races/160 posts passed. Existing nonfatal report/delete races still occur.
VM resources fixed: app 4vCPU/8GiB, bench 4vCPU/4GiB; app IP 192.168.2.7. Only two VM processes. No more macOS administrator dialogs expected.
Go pprof 50s CPU and heap captured; 62.54% allocation under fillUserResponse, mostly SQL icon blob copying. Next large hypothesis: persist icon SHA256 metadata and use conditional GET to eliminate blob transfers.
Initial post-recovery scores 9415,7595,6696; third overlapped host Archive Utility 438% CPU. Exploratory 9479 excluded from median due extra old VM process. All attempts including setup failures retained.
Fork hiroto7/isucon13 branch codex/local-benchmark; push authorized, no PR. Benchmark official code unchanged, TLS verification enabled.
At genuine structural-gain stall: reread spec and fresh profile, then public solution comparison before ending. Final persistence/reboot and full benchmark still outstanding.
