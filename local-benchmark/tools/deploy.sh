#!/bin/bash
set -euo pipefail
root=$(cd "$(dirname "$0")/../.." && pwd)
app=${1:-isucon13-app}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
# Only source-controlled application files; remove old Go sources within the managed directory.
git -C "$root" ls-files -co --exclude-standard webapp/go webapp/sql webapp/pdns | while IFS= read -r path; do
  mkdir -p "$tmp/$(dirname "$path")"
  cp "$root/$path" "$tmp/$path"
done
(cd "$tmp" && COPYFILE_DISABLE=1 tar --no-xattrs -czf source.tar.gz webapp)
multipass transfer "$tmp/source.tar.gz" "$app:/tmp/source.tar.gz"
multipass exec "$app" -- sudo bash -c 'set -euo pipefail
systemctl stop isupipe-go
find /home/isucon/webapp/go /home/isucon/webapp/sql /home/isucon/webapp/pdns -type f -name "._*" -delete
find /home/isucon/webapp/go -maxdepth 1 -name "*.go" -delete
# The PDNS directory contains managed source only; generated zones live in /home/isucon/dns.
find /home/isucon/webapp/pdns -maxdepth 1 -type f -delete
tar xzf /tmp/source.tar.gz -C /home/isucon
chown -R isucon:isucon /home/isucon/webapp
cd /home/isucon/webapp/go
sudo -u isucon /home/isucon/local/golang/bin/go build -o isupipe
systemctl start isupipe-go
systemctl is-active isupipe-go
sha256sum isupipe'
# Source-file set and bytes must match before any benchmark.
(cd "$tmp/webapp" && find go sql pdns -type f -exec shasum -a 256 {} \; | sort) > "$tmp/host.manifest"
multipass exec "$app" -- bash -c 'cd /home/isucon/webapp; find go sql pdns -type f ! -name isupipe -exec sha256sum {} \; | sort' > "$tmp/guest.manifest"
diff -u "$tmp/host.manifest" "$tmp/guest.manifest"
