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
