# Hypotheses

No performance adoption before baseline and load evidence.

Static observations to test, not adoption decisions:
- Statistics handlers scan all users/livestreams and issue repeated per-entity aggregates. Profile statement counts and total time.
- Moderation scans all comments for each NG word; inspect work volume and preserve SQL LIKE semantics and deletion visibility.
- Response construction repeats user/theme/icon/tag reads. Quantify before designing request-local batching or caching.
- DNS backend removes the name/type index during provisioning. Measure random-name load and interference with application DB.

These are official-code observations. No ISUCON13 external solutions have been read.

Stats batching: passed 8454, user statistics mean 59.75ms vs initial exploratory 10694ms. Score not beyond variation, adoption pending.
DNS name/type index: passed 7102, DNS responses 30895 vs 9291. Not a score improvement; adoption pending in combination, record concurrent diagnostic build.
Moderation: replace all-words x all-comments DELETE round trips with one SQL EXISTS/DELETE, preserve utf8mb4_general_ci and LIKE wildcards verified with same Go driver connection (case, accent, percent/underscore/escape, Japanese, empty).

Request-local response batching adopted: official32448 vs15589, same transaction snapshot; targeted concurrent write checks passed. Next: SQL driver interpolation to eliminate repeated Prepare/CloseStmt commands documented in full DB profile.

Icon metadata retried after response batching: combined with interpolation48361, +49% vs32448; adopted. Earlier11501 on unbatched predecessor remains rejected, illustrating dependencies.

Synchronized icon hash cache + compact JSON adopted87349 (+81%vs48361). Hash-only entries, matching304 noSQL, generation fence acrosswrite andinit; stale-read HTTP races passed. Next immutable user DTO cache and reservation interval indexing/remove per-slot reread.

DNS BIND candidate: remove all per-query SQL backend work, using the installed PowerDNS BIND module and a native zone. Preserve the SQL registry and original pdnsutil name validation; registration exports the canonical full zone, fsyncs a temporary file, renames and fsyncs the directory, then synchronously reloads PowerDNS before committing the user and responding201. File updates serialize with flock; registration and initialize serialize through a lifecycle RW lock before user transaction creation, avoiding TRUNCATE/writer deadlock. Complete export is deliberately simple; optimize only if measured. Unknown names remain NXDOMAIN, no attacker-name special case. All caches0 avoid negative-cache publication races. The user authorized pdns group membership confined to this VM. Official backend documentation: https://doc.powerdns.com/authoritative/backends/bind.html .

Fresh detached profile after DNS rollback: 617570 commands, COMMIT159s/61.9%; Go syscall33.65% +futex8.89%. mysqld perf shows TCP wakeup paths9.38% flat. SQL rank aggregates total only2s, so more ranking changes cannot be explosive at this load. Slow log settings restored and verified OFF/long_query_time10. Deployment caught an unused publish_zone.py left after candidate rollback; app binary and source paths actually executed were correct, but complete manifest differed. Fixed deploy.sh to synchronize the entire managed pdns source directory; generated runtime zones are outside it. This run is diagnostic only.
Unix-socket candidate failed initialization first: Go accepted unix but init.sh passed the path to --host. Preserve failure; change initialization connection array for --protocol=SOCKET and --socket, then retry.
Another grounded structural candidate: live schema.txt has log_bin ON, sync_binlog1, flush_at_trx_commit1. Earlier assumption of binlog OFF was wrong. SHOW REPLICA STATUS/SHOW REPLICAS and replication_connection_configuration empty, gtid_modeOFF. Disable unused binary logging with skip-log-bin, retain InnoDB redo and commit fsync1; old logs retained. Official MySQL8.0 docs explain default binary logging and startup disable option: https://dev.mysql.com/doc/refman/8.0/en/binary-log.html .
