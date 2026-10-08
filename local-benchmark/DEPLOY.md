# Deploy and recover

Launch app VM: multipass launch 22.04 --name isucon13-app --cpus 4 --memory 8G --disk 40G --cloud-init local-benchmark/app-cloud-init.cfg
Launch benchmark VM: multipass launch 22.04 --name isucon13-bench --cpus 4 --memory 4G --disk 12G --cloud-init local-benchmark/bench-cloud-init.cfg
Wait for successful cloud-init on both. Capture package/runtime versions and service configs.
Copy the app VM public TLS certificate into /usr/local/share/ca-certificates/ on the benchmark VM, then update-ca-certificates.
App DNS records must point to its reachable VM IP, recorded in /home/isucon/env.sh.
Run tools/deploy.sh, compare complete source manifests, verify service and binary.
Run tools/run_trial.py STAGE --ip APP_IP. Deployments and runs are serialized by one operator.
Rollback code, schema and config to the accepted state, then initialize and validate; do not roll back code alone.
Configuration changes must be stored under local-benchmark/config and applied before deployment.
Final: record persisted entities, reboot, verify reads BEFORE initialize, then run complete official benchmark.

DNS candidate: transfer config/dns-index.sql and execute with sudo mysql after baseline provisioning. Confirm SHOW INDEX FROM isudns.records. Rollback DROP INDEX nametype_index ON isudns.records. Zone initialization keeps the table and index.
