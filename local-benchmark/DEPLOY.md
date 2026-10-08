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

Accepted relational indexes: apply config/relational-indexes.sql once to baseline database; rollback with config/rollback-relational-indexes.sql. Definitions also in webapp/sql/initdb.d/10_schema.sql for fresh schema. Normal initialization truncates data but preserves indexes.

DNS-cache candidate (not adopted): config/pdns-performance.conf, 2s positive/packet TTL, no negative-query caching, zone metadata60s and zone list10s. Existing localhost API flushes new usernames; init waits3s to expire removed names. No group/permission change. Rollback config/pdns-before-cache.conf and webapp to bae7069. Official API supports exact cache flush only in installed4.5.3; verified source https://raw.githubusercontent.com/PowerDNS/pdns/auth-4.5.3/pdns/ws-auth.cc . Performance reference https://doc.powerdns.com/authoritative/performance.html .

BIND candidate setup: deploy the new sources, transfer config/pdns-bind.conf to /tmp/pdns-bind.conf and tools/install_bind_backend.sh to the guest, then run the script as isucon. Generated /home/isucon/dns state is outside the source manifest. pdnsutil uses /home/isucon/dns/admin/pdns.conf with the original MySQL backend; the authoritative daemon uses launch=bind and a native zone. Supplemental pdns membership grants control-socket reload only within the isolated guest; restart the app to pick up membership. Normal initialization reloads both SQL and file state. Persisted zone is fsynced before a successful registration. Restore config/pdns-before-cache.conf and remove ISUCON13_POWERDNS_BACKEND from env.sh, restart pdns/app to roll back; SQL DNS registry stays populated throughout, so reverting needs no DNS data reconstruction. PowerDNS4.5.3 supports native zones and bind-reload-now; installed module verified.
