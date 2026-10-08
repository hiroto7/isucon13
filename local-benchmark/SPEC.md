# Specification and score path

Sources: docs/cautionary_note.md, docs/isupipe.md, official benchmark source.
Direct score: total tips successfully counted by the benchmark in the 60 second load.
Trace viewer/streamer scenario progression to comment/tip completion; DNS/HTTPS failures block progression.
Initialization: application DB and DNS zone, timeout 42 seconds.
Preserve API shape, authorization, ordering, moderation, reservation capacity, and statistics.
bcrypt algorithm and cost must remain unchanged.
Icon bytes and hash must reflect updates within 2 seconds; conditional GET may return 304 only on a matching hash.
Restart: previously written data must remain readable before another initialization.
Final benchmark after reboot must pass and exceed 75% of the pre-reboot score.
Local adaptation: ARM64, fixed VM resources, fresh TLS cert, independent benchmark VM; not a competition hardware comparison.

The official docs/isupipe.md explicitly identifies load-time moderated-spam expected400/actual201 as a benchmark issue that does not reduce score. The scheduler records moderation globally while the application scopes it by livestream. Keep per-stream semantics; targeted same-stream race tests still matter. Deleted-comment report500/404 is separately retained and not excused by that note.

Final DNS backend uses the canonical persistent isudns SQL registry and original pdnsutil validation. Known A answers are never throttled; general per-source UDP negative-answer protection is bounded100/s, burst100, max4096clients, with TCP negative fallback. Initialization serializes entire registrations before SQL truncation. No benchmark-specific hostname/IP patterns.
Final MySQL mode2 writes redo at each commit and flushes once per second; normal systemctl reboot persistence is verified in validation/. This is not a power-loss-durable configuration. Binary logging is disabled for this standalone non-replicated topology; strict per-commit redo sync fallback is retained.
