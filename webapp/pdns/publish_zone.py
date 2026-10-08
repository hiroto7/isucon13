#!/usr/bin/env python3
"""Keep the SQL registry canonical; durably publish a complete BIND zone."""
import fcntl
import os
from pathlib import Path
import subprocess
import sys
import tempfile

ROOT = Path("/home/isucon/dns")
CLI = ["pdnsutil", "--config-dir=" + str(ROOT / "admin")]
ZONE = "u.isucon.dev"

# Across registrations and initialization, no export can overwrite newer records.
with (ROOT / "publish.lock").open("a") as lock:
    fcntl.flock(lock, fcntl.LOCK_EX)
    if sys.argv[1] == "add":
        subprocess.run(CLI + ["add-record", ZONE, sys.argv[2], "A", "0", sys.argv[3]], check=True)
    elif sys.argv[1] == "load":
        subprocess.run(CLI + ["load-zone", ZONE, sys.argv[2]], check=True)
    elif sys.argv[1] != "export":
        raise ValueError("unknown operation")
    data = subprocess.check_output(CLI + ["list-zone", ZONE])
    fd, temporary = tempfile.mkstemp(prefix=".zone-", dir=ROOT)
    try:
        with os.fdopen(fd, "wb") as output:
            os.fchmod(output.fileno(), 0o644)
            output.write(data)
            output.flush()
            os.fsync(output.fileno())
        os.replace(temporary, ROOT / "active.zone")
        directory = os.open(ROOT, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(directory)
        finally:
            os.close(directory)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)
    if sys.argv[1] != "export":
        subprocess.run(["pdns_control", "bind-reload-now", ZONE], check=True)
        status = subprocess.check_output(["pdns_control", "bind-domain-status", ZONE], text=True)
        print(status, end="")
        if "parsed into memory" not in status:
            raise RuntimeError("zone reload failed")
