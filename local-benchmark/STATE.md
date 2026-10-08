# ISUCON13 current state

Phase: app VM rebuilding after stop-all interrupted setup and daemon restart left orphan QEMU.
No performance changes or valid benchmarks yet. Initial baseline not established.
App VM: isucon13-app, Ubuntu 22.04 ARM64, 4 vCPU/8 GiB/40 GiB.
Benchmark VM: isucon13-bench, 4 vCPU/4 GiB/12 GiB, pinned official bench built with Go 1.21.2.
Benchmark HOME/GOPATH cloud-init issue repaired by explicit env; /opt/bench-ready exists.
private-isu: stopped, disk preserved; no guest backup obtained, do not delete it.
isucon14: deleted after complete backup and 11 Go files matched host; host repo retained.
Target 100x median of 3 normal initial runs, fixed resources. Fork hiroto7/isucon13, branch codex/local-benchmark.
Next: verify cloud-init/service readiness, trust app public certificate in benchmark VM, verify deployment source manifests, install stable diagnostic log config, run baseline.
Stop when fresh profiling/spec alternatives/public solution review leaves no grounded substantial-gain hypothesis.
