#!/usr/bin/env python3
"""Run on app guest after a trial; verify mutable owner and duplicate tag semantics."""
import base64, hashlib, json, pathlib, subprocess, time, urllib.parse
# Reuse HTTP fixture helpers only, without executing prepare/check.
source = pathlib.Path('/tmp/persistence_check.py').read_text()
exec(source.split('if sys.argv[1]')[0])
name='streamcache'+str(time.time_ns()); op=client()
expect(op,'/api/register',dict(name=name,display_name='Cache check',description='',password='local-test',theme={'dark_mode':False}),201)
expect(op,'/api/login',dict(username=name,password='local-test'),200)
tag_id, tag_name=subprocess.check_output(MYSQL+['SELECT id,name FROM tags ORDER BY id LIMIT 1'],text=True).strip().split('\t',1)
start,end=map(int,subprocess.check_output(MYSQL+['SELECT start_at,end_at FROM reservation_slots WHERE slot>0 ORDER BY id LIMIT 1'],text=True).split())
stream=json.loads(expect(op,'/api/livestream/reservation',dict(title='Cache check',description='Immutable',playlist_url='https://example.com/list',thumbnail_url='https://example.com/image',tags=[int(tag_id),int(tag_id)],start_at=start,end_at=end),201));sid=stream['id']
assert [t['id'] for t in stream['tags']]==[int(tag_id)]*2
warm=json.loads(expect(op,f'/api/livestream/{sid}',None,200));assert warm==stream
image=pathlib.Path('/home/isucon/webapp/img/NoImage.jpg').read_bytes()+name.encode();digest=hashlib.sha256(image).hexdigest()
expect(op,'/api/icon',{'image':base64.b64encode(image).decode()},201)
updated=json.loads(expect(op,f'/api/livestream/{sid}',None,200))
assert updated['owner']['icon_hash']==digest
comment=json.loads(expect(op,f'/api/livestream/{sid}/livecomment',{'comment':'cache owner','tip':123},201))
reaction=json.loads(expect(op,f'/api/livestream/{sid}/reaction',{'emoji_name':'cache-owner'},201))
for value in (comment,reaction):
 assert value['livestream']['owner']['icon_hash']==digest
 assert value['livestream']['tags']==stream['tags']
for kind in ('livecomment','reaction'):
 values=json.loads(expect(op,f'/api/livestream/{sid}/{kind}',None,200))
 assert values and all(v['livestream']['owner']['icon_hash']==digest for v in values)
search=json.loads(expect(op,'/api/livestream/search?tag='+urllib.parse.quote(tag_name),None,200))
matches=[v for v in search if v['id']==sid]
assert len(matches)==2 and all(v==updated for v in matches)
assert all(search[i]['id']>=search[i+1]['id'] for i in range(len(search)-1))
print('PASS warm stream cache retains duplicate tags/order, owner icon changes visible after201 in stream/comment/reaction reads and writes, tagged JOIN preserves duplicate results and descending order')
