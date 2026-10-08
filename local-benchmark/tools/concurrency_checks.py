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
suffix=str(time.time_ns())
users=[]
for n in range(8):
 name='concurrency'+suffix+str(n); op=client()
 expect(op,'/api/register',dict(name=name,display_name=name,description='',password='local-test',theme={'dark_mode':True}),201)
 expect(op,'/api/login',dict(username=name,password='local-test'),200)
 users.append((name,op))
image=pathlib.Path('/home/isucon/webapp/img/NoImage.jpg').read_bytes()
def update(pair,n):
 name,op=pair; data=image+str(n).encode()
 expect(op,'/api/icon',{'image':base64.b64encode(data).decode()},201)
with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
 list(pool.map(lambda n:update(users[n%8],n),range(64)))
with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
 list(pool.map(lambda n:update(users[0],n),range(32)))
name,op=users[0]; update(users[0],999)
actual=expect(op,'/api/user/'+name+'/icon',None,200); assert actual==image+b'999'
me=json.loads(expect(op,'/api/user/me',None,200)); assert me['icon_hash']==hashlib.sha256(actual).hexdigest()
print('PASS: 64 cross-owner updates, 32 same-owner updates, final bytes/hash')
status,matched=call(op,'/api/user/'+name+'/icon',headers={'If-None-Match':json.dumps(me['icon_hash'])})
assert status in (200,304),(status,matched[:300])
if status==200: assert matched==actual
expect(op,'/api/user/'+name+'/icon',None,200,{'If-None-Match':json.dumps('incorrect')})
update(users[0],1000)
changed=expect(op,'/api/user/'+name+'/icon',None,200,{'If-None-Match':json.dumps(me['icon_hash'])})
assert changed==image+b'1000'
newme=json.loads(expect(op,'/api/user/me',None,200)); assert newme['icon_hash']==hashlib.sha256(changed).hexdigest()
print('PASS: matching ETag 200/304, mismatching 200, stale ETag 200 and fresh hash')

slot=subprocess.check_output(['mysql','-uisucon','-pisucon','-h127.0.0.1','isupipe','-Nse','SELECT start_at,end_at FROM reservation_slots WHERE slot > 0 ORDER BY id LIMIT 1'],text=True).strip().split()
assert len(slot)==2,'no available slot'
stream=json.loads(expect(op,'/api/livestream/reservation',dict(title='Concurrency',description='',playlist_url='https://example.com/list',thumbnail_url='https://example.com/image',tags=[],start_at=int(slot[0]),end_at=int(slot[1])),201))
sid=stream['id']
for n in range(10):
 word='racecheck'+suffix+str(n)
 def post(_):
  status,data=call(op,f'/api/livestream/{sid}/livecomment',dict(comment=word,tip=0)); assert status in (201,400),(status,data)
 def moderate(_): expect(op,f'/api/livestream/{sid}/moderate',{'ng_word':word},201)
 with concurrent.futures.ThreadPoolExecutor(max_workers=17) as pool:
  tasks=[pool.submit(post,j) for j in range(8)]+[pool.submit(moderate,0)]+[pool.submit(post,j) for j in range(8)]
  for task in tasks: task.result()
 comments=json.loads(expect(op,f'/api/livestream/{sid}/livecomment',None,200))
 assert not any(word in c['comment'] for c in comments),(n,comments)
 expect(op,f'/api/livestream/{sid}/livecomment',dict(comment=word,tip=0),400)
print('PASS: 10 moderation races, 160 concurrent posts, no surviving spam')
