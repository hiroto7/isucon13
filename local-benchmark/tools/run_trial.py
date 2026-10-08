#!/usr/bin/env python3
"""Serialize deployment/benchmarks and retain valid, failed and interrupted trials."""
import argparse, datetime as dt, hashlib, json, os, pathlib, subprocess, sys, time
ROOT = pathlib.Path(__file__).resolve().parents[2]
BASE = ROOT / 'local-benchmark'
def run(argv, **kwargs):
    return subprocess.run(argv, check=True, text=True, timeout=180, **kwargs)
def mp(*argv, **kwargs):
    return run(['multipass', *argv], **kwargs)
def capture(argv):
    return subprocess.check_output(argv, text=True).strip()
def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('stage')
    parser.add_argument('--app', default='isucon13-app')
    parser.add_argument('--bench', default='isucon13-bench')
    parser.add_argument('--ip', required=True)
    parser.add_argument('--decision', default='pending')
    args = parser.parse_args()
    lock = BASE / '.run.lock'
    fd = os.open(lock, os.O_CREAT | os.O_EXCL | os.O_WRONLY)
    os.close(fd)
    stamp = dt.datetime.now(dt.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
    out = BASE / 'results' / (stamp + '-' + args.stage)
    out.mkdir(parents=True)
    diff = capture(['git', '-C', str(ROOT), 'diff', 'HEAD', '--', 'webapp', 'local-benchmark/config'])
    row = dict(started_at=stamp, stage=args.stage, decision=args.decision,
               commit=capture(['git', '-C', str(ROOT), 'rev-parse', 'HEAD']),
               diff_sha256=hashlib.sha256(diff.encode()).hexdigest(),
               status='preparation_failed', score=None, reported_score=None,
               app=args.app, benchmark=args.bench, ip=args.ip,
               resources='app:4vCPU/8GiB;bench:4vCPU/4GiB', diagnostics='nginx timing, MySQL digests, vmstat')
    (out/'source.diff').write_text(diff)
    record = out/'result.json'
    record.write_text(json.dumps(row, indent=2)+'\n')
    try:
        # Verify current service and binary before starting. Deploy is a separate operation.
        ready = mp('exec', args.app, '--', 'sudo', 'sh', '-c',
            'set -e; systemctl is-active mysql pdns nginx isupipe-go; sha256sum /home/isucon/webapp/go/isupipe; '
            'p=$(systemctl show isupipe-go -p MainPID --value); readlink /proc/$p/exe; '
            'mysql -e "TRUNCATE TABLE performance_schema.events_statements_summary_by_digest"', capture_output=True)
        (out/'ready.txt').write_text(ready.stdout)
        mp('exec', args.bench, '--', 'sudo', 'mkdir', '-p', '/opt/trial')
        mp('exec', args.bench, '--', 'sudo', 'rm', '-f', '/opt/trial/result.json')
        mp('exec', args.app, '--', 'sudo', 'sh', '-c',
           ': > /var/log/nginx/access.log; vmstat 1 150 > /tmp/trial-vmstat.txt &')
        argv = ['multipass','exec',args.bench,'--','sudo','sh','-c',
            'cd /opt/isucon13/bench && /opt/bench run --enable-ssl --target https://pipe.u.isucon.dev '
            f'--nameserver {args.ip} --result-path /opt/trial/result.json '
            '--staff-log-path /opt/trial/staff.log --contestant-log-path /opt/trial/contestant.log']
        row['status']='running'; record.write_text(json.dumps(row,indent=2)+'\n')
        with (out/'console.txt').open('w') as console:
            proc = subprocess.run(argv, stdout=console, stderr=subprocess.STDOUT, timeout=240)
        row['exit_code'] = proc.returncode
        for name in ['result.json','staff.log','contestant.log']:
            mp('transfer', f'{args.bench}:/opt/trial/{name}', str(out/('official-'+name)))
        official = json.loads((out/'official-result.json').read_text())
        row['reported_score'] = official.get('score')
        row['status'] = 'passed' if proc.returncode == 0 and official.get('pass') is True else 'failed'
        row['score'] = official.get('score') if row['status']=='passed' else None
        row['resolved_count'] = official.get('resolved_count')
        row['messages'] = official.get('messages')
        mp('exec',args.app,'--','sudo','sh','-c',
           'mysql -B -e "SELECT DIGEST_TEXT,COUNT_STAR,ROUND(SUM_TIMER_WAIT/1e12,3) seconds,SUM_ROWS_EXAMINED '
           'FROM performance_schema.events_statements_summary_by_digest ORDER BY SUM_TIMER_WAIT DESC LIMIT 30" '
           '> /tmp/trial-digests.tsv; chmod 644 /tmp/trial-digests.tsv')
        for remote, name in [('/tmp/trial-digests.tsv','digests.tsv'),('/tmp/trial-vmstat.txt','vmstat.txt'),('/var/log/nginx/access.log','access.log')]:
            mp('transfer', f'{args.app}:{remote}', str(out/name))
    except KeyboardInterrupt:
        row['status']='aborted'; row['score']=None
    except Exception as e:
        row['error']=str(e)
        if row['status']=='running': row['status']='aborted'
        if row['status'] != 'passed': row['score']=None
    finally:
        row['finished_at']=dt.datetime.now(dt.timezone.utc).isoformat()
        record.write_text(json.dumps(row,indent=2)+'\n')
        lock.unlink(missing_ok=True)
    print(json.dumps(row,ensure_ascii=False))
    return 0 if row['status']=='passed' else 1
if __name__=='__main__': sys.exit(main())
