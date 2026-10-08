#!/usr/bin/env python3
import collections,json,re,sys
rows=collections.defaultdict(list)
for line in open(sys.argv[1]):
 try: row=json.loads(line)
 except ValueError: continue
 uri=re.sub(r'/[0-9]+(?=/|$)', '/:id', row['uri'])
 uri=re.sub(r'/api/user/[^/]+', '/api/user/:username', uri)
 rows[(row['method'],uri,row['status'])].append(row['request_time'])
print('method\turi\tstatus\tcount\ttotal_seconds\tmean_ms\tp95_ms')
for (method,uri,status),times in sorted(rows.items(),key=lambda r:sum(r[1]),reverse=True):
 times.sort()
 print(f'{method}\t{uri}\t{status}\t{len(times)}\t{sum(times):.3f}\t{1000*sum(times)/len(times):.2f}\t{times[int(.95*(len(times)-1))]*1000:.2f}')
