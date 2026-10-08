# Hypotheses

No performance adoption before baseline and load evidence.

Static observations to test, not adoption decisions:
- Statistics handlers scan all users/livestreams and issue repeated per-entity aggregates. Profile statement counts and total time.
- Moderation scans all comments for each NG word; inspect work volume and preserve SQL LIKE semantics and deletion visibility.
- Response construction repeats user/theme/icon/tag reads. Quantify before designing request-local batching or caching.
- DNS backend removes the name/type index during provisioning. Measure random-name load and interference with application DB.

These are official-code observations. No ISUCON13 external solutions have been read.
