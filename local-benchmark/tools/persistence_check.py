#!/usr/bin/env python3
"""Run on app VM between official trials; fixtures are cleared by initialization."""
import http.cookies, base64, concurrent.futures, hashlib, http.cookiejar, json, pathlib, subprocess, time, urllib.error, urllib.request
BASE='http://127.0.0.1:8080'
def client():
 return {}
def call(opener,path,body=None,headers=None):
 req=urllib.request.Request(BASE+path,data=None if body is None else json.dumps(body).encode(),headers={'Content-Type':'application/json','Cookie':opener.get('cookie',''),**(headers or {})})
 try:
  with urllib.request.urlopen(req,timeout=30) as r:
   if path == '/api/login':
    cookies=http.cookies.SimpleCookie(); cookies.load('; '.join(r.headers.get_all('Set-Cookie') or [])); opener['cookie']='; '.join(k+'='+v.value for k,v in cookies.items())
   return r.status,r.read()
 except urllib.error.HTTPError as e: return e.code,e.read()
def expect(opener,path,body,status,headers=None):
 actual,data=call(opener,path,body,headers); assert actual==status,(path,actual,data[:300]); return data

import sys
PROOF=pathlib.Path('/home/isucon/persistence-proof.json')
MYSQL=['mysql','-uisucon','-pisucon','-h127.0.0.1','isupipe','-Nse']

def inventory():
 tables=['users','themes','icons','livestreams','livestream_tags','reservation_slots','livestream_viewers_history','livecomments','reactions','ng_words','livecomment_reports']
 sql=' UNION ALL '.join("SELECT '%s',COUNT(*),COALESCE(SUM(id),0) FROM %s"%(t,t) for t in tables)
 return subprocess.check_output(MYSQL+[sql],text=True)

def snapshot(proof):
 op={'cookie':proof['cookie']}; name=proof['name']; sid=proof['sid']
 paths=['/api/user/me', '/api/user/'+name+'/livestream',f'/api/livestream/{sid}/livecomment',f'/api/livestream/{sid}/reaction',f'/api/livestream/{sid}/ngwords',f'/api/livestream/{sid}/report',f'/api/livestream/{sid}/statistics','/api/user/'+name+'/statistics']
 result={p:json.loads(expect(op,p,None,200)) for p in paths}
 icon=expect(op,'/api/user/'+name+'/icon',None,200)
 result['icon_sha256']=hashlib.sha256(icon).hexdigest()
 result['dns']=subprocess.check_output(['dig','@127.0.0.1',name+'.u.isucon.dev','A','+short'],text=True).strip()
 result['inventory']=inventory()
 return result

if sys.argv[1]=='prepare':
 name='persist'+str(time.time_ns()); op=client()
 expect(op,'/api/register',dict(name=name,display_name='Persistent user',description='Before reboot',password='local-test',theme={'dark_mode':True}),201)
 expect(op,'/api/login',dict(username=name,password='local-test'),200)
 icon=pathlib.Path('/home/isucon/webapp/img/NoImage.jpg').read_bytes()+name.encode()
 expect(op,'/api/icon',{'image':base64.b64encode(icon).decode()},201)
 start,end=map(int,subprocess.check_output(MYSQL+['SELECT start_at,end_at FROM reservation_slots WHERE slot>0 ORDER BY id LIMIT 1'],text=True).split())
 tag=int(subprocess.check_output(MYSQL+['SELECT id FROM tags ORDER BY id LIMIT 1'],text=True))
 stream=json.loads(expect(op,'/api/livestream/reservation',dict(title='Persisted stream',description='Reboot verification',playlist_url='https://example.com/list',thumbnail_url='https://example.com/image',tags=[tag],start_at=start,end_at=end),201));sid=stream['id']
 expect(op,f'/api/livestream/{sid}/enter',{},200)
 comment=json.loads(expect(op,f'/api/livestream/{sid}/livecomment',{'comment':'persisted comment','tip':123},201))
 expect(op,f'/api/livestream/{sid}/reaction',{'emoji_name':'persistence-emoji'},201)
 expect(op,f'/api/livestream/{sid}/moderate',{'ng_word':'do-not-allow'},201)
 expect(op,f'/api/livestream/{sid}/livecomment/{comment["id"]}/report',{},201)
 proof={'name':name,'sid':sid,'cookie':op['cookie']};proof['snapshot']=snapshot(proof)
 assert proof['snapshot']['icon_sha256']==hashlib.sha256(icon).hexdigest()
 assert proof['snapshot']['dns']=='192.168.2.7'
 PROOF.write_text(json.dumps(proof,ensure_ascii=False,indent=2)+'\n')
 print('PASS prepare: committed user/theme/icon/stream/tags/reservation/viewer/comment/tip/reaction/NG/report and DNS; saved read snapshot')
elif sys.argv[1]=='check':
 proof=json.loads(PROOF.read_text());actual=snapshot(proof)
 assert actual==proof['snapshot'],[(k,actual[k],proof['snapshot'].get(k)) for k in actual if actual[k]!=proof['snapshot'].get(k)]
 op=client();expect(op,'/api/login',dict(username=proof['name'],password='local-test'),200)
 expect(op,f'/api/livestream/{proof["sid"]}/livecomment',{'comment':'do-not-allow','tip':0},400)
 print('PASS after reboot BEFORE initialize: exact API/icon/DNS/table inventory snapshots, password login and persisted moderation')
else: raise ValueError('expected prepare/check')
