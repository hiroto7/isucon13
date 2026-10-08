# Hypotheses and final decisions

| Structural hypothesis | Evidence / score | Decision |
|---|---|---|
| Aggregate statistics, bulk SQL moderation, relation indexes and concurrency locks | Indexed consistent predecessor15589; per-entity repeated queries reduced | Incorporated; isolated8454/7102 not claimed as gains |
| Batch response user/theme/icon/tags within the same transaction snapshot | 32448 vs15589 | Adopted833bcbe |
| Remove Prepare round trips; stored icon hash and metadata-only reads | 48361 vs32448 | Adoptedbae7069; standalone icon11501 rejected |
| Generation-fenced hash cache and compact JSON | 87349 vs48361; HTTP races passed | Adopted8008cd6 |
| Immutable DTO, reservation range, idle reuse, upstream keepalive, Unix socket, remove unused binlog | 133004 vs87349 | Adopted93afca4; provisional steps retained, not independently claimed |
| Shared durable write transactions | 96404 /107746 vs106506; waiting negated commit reduction | Rejected2a95b15 |
| PowerDNS BIND persistent canonical zone | 108205 vs112272; full zone export serialized registration | Rejected799123b |
| Quiet logs and per-second redo flush | 195726 vs133004 | Adopteda4ab49d; normal reboot verified later, power-loss caveat |
| Canonical memory DNS with general negative-answer rate limit | 224359 before/225280 after reboot | Adopted7bd08d8 |
| Concurrent pdnsutil + per-call ID lower bound and deduplicated publication | Race tests passed;212786 vs225280 | Rejected0e32969, restoredf72c217 |

DNS2s TTL cache49885 and pool32 score52612 were inconclusive small changes and rejected. A failed Unix initialization (socket incorrectly passed as --host), unused-import build failure, duplicate nginx log_format failure and interrupted provisioning runs are preserved as failures rather than score zero. Profiles are diagnostics rather than adoption evidence.

Final profile indicates distributed SQL/Go work rather than a remaining high-cost all-history/N+1 path. Tag joins17%, COMMIT21.8%, reservation update6.8% of instrumented SQL time; these percentages alone do not prove potential score gain. Fixed bcrypt cost cannot be lowered. No observed nginx connection/file-limit error supports changing network limits. Last registration serialization hypothesis did not improve total score. Stop here per user's requirement; no claim that further improvement is mathematically impossible.

The supplied ISUCON14 playbook informed evidence/state/deployment workflow only. Current specifications came from official ISUCON13 docs and code. Public winner retrospective was consulted only after independent structural work stalled: https://zenn.dev/tohutohu/articles/923bdf5dcd73af . Logging/DNS concepts were treated as hypotheses and measured locally; hardware expansion and sleep-based changes were not imported. General UDP negative response limiting reference: https://bind9.readthedocs.io/en/v9.18.13/reference.html . MySQL binary-log and redo semantics: https://dev.mysql.com/doc/refman/8.0/en/binary-log.html and https://dev.mysql.com/doc/refman/8.0/en/innodb-parameters.html .
