# ISUCON13 current state

Accepted score 48361 (initial median 7595: 6.37x; initial exploratory best 9479: 5.10x). Goal: 759500, 100x median.
Accepted: grouped snapshot statistics; DNS name/type index; bulk SQL LIKE moderation with original general_ci semantics; relation lookup indexes; icon write READ COMMITTED + owner lock; stream row lock for posting/moderation; request-local response batching.
Response batching: user/theme/icon reads deduplicated and batched; stream tags joined in one batch; comment/reaction lists reuse stream/user DTOs. All use the caller's transaction snapshot and no cache crosses requests.
Official full benchmark passed 32448; 64 cross-owner and 32 same-owner icon updates and 10 moderation races/160 posts passed. Nonfatal report/deletion errors and benchmark moderated-spam 400/201 discrepancies remain and require investigation.
Fixed environment: app 4vCPU/8GiB; bench 4vCPU/4GiB; IP 192.168.2.7. Official benchmark unchanged, TLS verification on. Baseline canonical runs 9415,7595,6696 (third overlapped host archive CPU); exploratory 9479 excluded because old orphan VM.
Go CPU/heap + full MySQL slow log and CPU perf captured. DB profile: 869260 commands, 236797 Prepare; tags 90467; icons 35831; COMMIT44 aggregate seconds. Profiling restored slow log OFF, long_query_time=10. Next structural hypothesis: interpolate driver parameters to remove prepare/execute/close round trips.
Icon metadata/conditional GET candidate score11501 not adopted, retained in e26ffb6; accepted binary restored before batching.
User asleep, may close Mac. No interactive macOS administrator recovery. Discard measurements spanning sleep; retry after wake. Final persistence/reboot, full reproducibility and complete-history graphs/push outstanding.
Fork hiroto7/isucon13 branch codex/local-benchmark; push authorized, no PR. At structural stall: fresh profile/spec audit and authorized public-solution comparison before stopping.

Adopted additional bundle: driver parameter interpolation + generated stored icon SHA256 with covering lookup index + metadata-only user DTO and conditional GET. Combined48361 vs32448; standalone interpolation35561 inconclusive and not independently adopted. Source uses generated hash so writes/initialization cannot omit updating it; metadata and image are read in same RR snapshot.
Next: fresh profiling of accepted state, then structural DNS/application bottleneck.
