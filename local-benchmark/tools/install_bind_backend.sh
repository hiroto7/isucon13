#!/usr/bin/env bash
# Run INSIDE the application VM after deploying webapp/pdns/publish_zone.py.
set -euo pipefail
sudo usermod -aG pdns isucon
sudo install -d -o isucon -g isucon -m 755 /home/isucon/dns /home/isucon/dns/admin
# Preserve the MySQL backend config for pdnsutil and rollback on first install.
if ! test -f /home/isucon/dns/admin/pdns.conf; then
    sudo cp /etc/powerdns/pdns.conf /home/isucon/dns/admin/pdns.conf
    sudo sed -i '/^include-dir=/d' /home/isucon/dns/admin/pdns.conf
    sudo chown isucon:isucon /home/isucon/dns/admin/pdns.conf
fi
sudo -u isucon python3 /home/isucon/webapp/pdns/publish_zone.py export
printf '%s\n' 'zone "u.isucon.dev" { type native; file "/home/isucon/dns/active.zone"; };' | sudo tee /home/isucon/dns/named.conf > /dev/null
sudo cp /tmp/pdns-bind.conf /etc/powerdns/pdns.conf
sudo pdns_server --config=check
sudo systemctl restart pdns
sudo -u isucon pdns_control bind-domain-status u.isucon.dev
if ! grep -q '^ISUCON13_POWERDNS_BACKEND=' /home/isucon/env.sh; then
    printf '%s\n' 'ISUCON13_POWERDNS_BACKEND="bind"' | sudo tee -a /home/isucon/env.sh > /dev/null
fi
sudo systemctl restart isupipe-go
