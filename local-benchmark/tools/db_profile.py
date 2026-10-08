"""MySQL-specific profiling. Restore settings even when the official trial fails."""
import json, pathlib, subprocess
class DBProfile:
 def __init__(self, app, out, mp):
  self.app,self.out,self.mp=app,out,mp; self.proc=None; self.original=None
 def sql(self,statement):
  return self.mp('exec',self.app,'--','sudo','mysql','-NBe',statement,capture_output=True).stdout
 def setup(self):
  self.original=json.loads(self.sql("SELECT JSON_OBJECT('slow_query_log',@@GLOBAL.slow_query_log,'long_query_time',@@GLOBAL.long_query_time,'slow_query_log_file',@@GLOBAL.slow_query_log_file,'log_output',@@GLOBAL.log_output)"))
  (self.out/'mysql-original-settings.json').write_text(json.dumps(self.original,indent=2)+'\n')
  self.sql("SET GLOBAL slow_query_log=OFF; SET GLOBAL long_query_time=0; SET GLOBAL log_output='FILE'; SET GLOBAL slow_query_log_file='/var/lib/mysql/trial-slow.log'")
  self.mp('exec',self.app,'--','sudo','install','-o','mysql','-g','mysql','-m','640','/dev/null','/var/lib/mysql/trial-slow.log')
  # GLOBAL long_query_time is inherited only by new sessions, including PowerDNS.
  self.mp('exec',self.app,'--','sudo','systemctl','restart','isupipe-go','pdns')
 def start(self):
  self.sql('SET GLOBAL slow_query_log=ON')
  (self.out/'mysql-status-before.txt').write_text(self.sql('SHOW GLOBAL STATUS'))
  self.proc=subprocess.Popen(['multipass','exec',self.app,'--','sudo','sh','-c',
   'perf record -e cpu-clock -F 99 --call-graph dwarf,8192 -p $(pgrep -x mysqld) -o /tmp/mysql-perf.data -- sleep 50'],
   stdout=subprocess.DEVNULL,stderr=(self.out/'mysql-perf-stderr.txt').open('w'))
 def finish(self):
  if self.original is None: return
  self.sql('SET GLOBAL slow_query_log=OFF')
  try:
   (self.out/'mysql-status-after.txt').write_text(self.sql('SHOW GLOBAL STATUS'))
   if self.proc is not None:
    self.proc.wait(timeout=60)
    (self.out/'mysql-profiler-status.json').write_text(json.dumps({'exit_code':self.proc.returncode})+'\n')
    self.mp('exec',self.app,'--','sudo','sh','-c',
     'pt-query-digest --limit 20 /var/lib/mysql/trial-slow.log > /tmp/mysql-queries.txt; '
     'perf report -i /tmp/mysql-perf.data --stdio --no-children --sort dso,symbol > /tmp/mysql-cpu.txt; '
     'cp /var/lib/mysql/trial-slow.log /tmp/mysql-slow.log; chmod 644 /tmp/mysql-queries.txt /tmp/mysql-cpu.txt /tmp/mysql-slow.log /tmp/mysql-perf.data')
    for remote,name in [('/tmp/mysql-queries.txt','mysql-queries.txt'),('/tmp/mysql-cpu.txt','mysql-cpu.txt'),('/tmp/mysql-slow.log','mysql-slow.log'),('/tmp/mysql-perf.data','mysql-perf.data')]:
     self.mp('transfer',f'{self.app}:{remote}',str(self.out/name))
  finally:
   def quoted(v): return "'"+str(v).replace("'","''")+"'"
   self.sql('SET GLOBAL long_query_time='+str(self.original['long_query_time'])+'; SET GLOBAL log_output='+quoted(self.original['log_output'])+'; SET GLOBAL slow_query_log_file='+quoted(self.original['slow_query_log_file'])+'; SET GLOBAL slow_query_log='+str(int(self.original['slow_query_log'])))
   self.mp('exec',self.app,'--','sudo','systemctl','restart','isupipe-go','pdns')
   (self.out/'mysql-restored-settings.txt').write_text(self.sql("SHOW VARIABLES WHERE Variable_name IN ('slow_query_log','long_query_time','slow_query_log_file','log_output')"))
