# ISUCON13 current state

Phase: environment preparation blocked on Multipass daemon restart.
No performance changes; no benchmark run; baseline is not established.
Official source: upstream-commit.txt. Branch: codex/local-benchmark. Fork: hiroto7/isucon13.
Old isucon14 VM: source/config/results backup in calling task workspace/backups/isucon14-preserve.tar.gz.
All 11 guest Go source files match the host repo. Host repos must remain.
private-isu: inaccessible via SSH; no backup yet; MUST NOT DELETE before saving guest-only changes.
Normal stop and forced stop hung. Pending CLI processes terminated to avoid concurrent restart operations.
Administrator recovery requested: sudo launchctl kickstart -k system/com.canonical.multipassd.
After recovery: save private-isu, delete named old VMs individually, check available disk, launch app/bench VMs per DEPLOY.md.
Target: 100x median of 3 normal initial runs with fixed resources.
Stop after current profiling/spec alternatives/public solution review leaves no substantial gain hypothesis.
