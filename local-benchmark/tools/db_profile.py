"""MySQL-specific profiling. Restore settings even when the official trial fails."""
import json, pathlib, subprocess, shlex
class DBProfile:
 def __init__(self, app, out, mp, dns_service="pdns"):
  self.app,self.out,self.mp=app,out,mp; self.proc=None; self.original=None
  self.services=["isupipe-go"]+(["pdns"] if dns_service=="pdns" else [])
 def sql(self,statement):
  return self.mp('exec',self.app,'--','sudo','mysql','-NBe',statement,capture_output=True).stdout
 def setup(self):
  self.original=json.loads(self.sql("SELECT JSON_OBJECT('slow_query_log',@@GLOBAL.slow_query_log,'long_query_time',@@GLOBAL.long_query_time,'slow_query_log_file',@@GLOBAL.slow_query_log_file,'log_output',@@GLOBAL.log_output)"))
  (self.out/'mysql-original-settings.json').write_text(json.dumps(self.original,indent=2)+'\n')
  self.sql("SET GLOBAL slow_query_log=OFF; SET GLOBAL long_query_time=0; SET GLOBAL log_output='FILE'; SET GLOBAL slow_query_log_file='/var/lib/mysql/trial-slow.log'")
  self.mp('exec',self.app,'--','sudo','install','-o','mysql','-g','mysql','-m','640','/dev/null','/var/lib/mysql/trial-slow.log')
  # GLOBAL long_query_time is inherited only by new sessions, including PowerDNS.
  self.mp('exec',self.app,'--','sudo','systemctl','restart',*self.services)
 def start(self):
  self.sql('SET GLOBAL slow_query_log=ON')
  (self.out/'mysql-status-before.txt').write_text(self.sql('SHOW GLOBAL STATUS'))
  script = 'perf record -e cpu-clock -F 99 --call-graph dwarf,8192 -p $(pgrep -x mysqld) -o /tmp/mysql-perf.data -- sleep 50; code=$?; echo "$code" > /tmp/mysql-perf.exit'
  launcher = 'rm -f /tmp/mysql-perf.exit /tmp/mysql-perf.data; nohup sh -c '+shlex.quote(script)+' >/tmp/mysql-perf-stderr.txt 2>&1 </dev/null &'
  self.proc=subprocess.Popen(['multipass','exec',self.app,'--','sudo','sh','-c',launcher],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)

 def finish(self):
  if self.original is None: return
  self.sql('SET GLOBAL slow_query_log=OFF')
  try:
   (self.out/'mysql-status-after.txt').write_text(self.sql('SHOW GLOBAL STATUS'))
   if self.proc is not None:
    self.proc.wait(timeout=60)
    (self.out/'mysql-profiler-status.json').write_text(json.dumps({'launcher_exit_code':self.proc.returncode, 'exit_code':int(self.mp('exec',self.app,'--','cat','/tmp/mysql-perf.exit',capture_output=True).stdout)})+'\n')
    self.mp('exec',self.app,'--','sudo','sh','-c',
     'pt-query-digest --limit 20 /var/lib/mysql/trial-slow.log > /tmp/mysql-queries.txt; '
     'perf report -i /tmp/mysql-perf.data --stdio --no-children --sort dso,symbol > /tmp/mysql-cpu.txt; '
     'cp /var/lib/mysql/trial-slow.log /tmp/mysql-slow.log; chmod 644 /tmp/mysql-queries.txt /tmp/mysql-cpu.txt /tmp/mysql-slow.log /tmp/mysql-perf.data')
    for remote,name in [('/tmp/mysql-perf-stderr.txt','mysql-perf-stderr.txt'),('/tmp/mysql-queries.txt','mysql-queries.txt'),('/tmp/mysql-cpu.txt','mysql-cpu.txt'),('/tmp/mysql-slow.log','mysql-slow.log'),('/tmp/mysql-perf.data','mysql-perf.data')]:
     self.mp('transfer',f'{self.app}:{remote}',str(self.out/name))
  finally:
   def quoted(v): return "'"+str(v).replace("'","''")+"'"
   self.sql('SET GLOBAL long_query_time='+str(self.original['long_query_time'])+'; SET GLOBAL log_output='+quoted(self.original['log_output'])+'; SET GLOBAL slow_query_log_file='+quoted(self.original['slow_query_log_file'])+'; SET GLOBAL slow_query_log='+str(int(self.original['slow_query_log'])))
   self.mp('exec',self.app,'--','sudo','systemctl','restart',*self.services)
   (self.out/'mysql-restored-settings.txt').write_text(self.sql("SHOW VARIABLES WHERE Variable_name IN ('slow_query_log','long_query_time','slow_query_log_file','log_output')"))
