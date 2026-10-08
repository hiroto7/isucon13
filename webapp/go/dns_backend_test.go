package main

import (
	"fmt"
	"github.com/miekg/dns"
	"net"
	"sync"
	"testing"
)

type dnsTestWriter struct {
	remote   net.Addr
	messages []*dns.Msg
}

func (w *dnsTestWriter) LocalAddr() net.Addr  { return &net.UDPAddr{IP: net.IPv4zero, Port: 53} }
func (w *dnsTestWriter) RemoteAddr() net.Addr { return w.remote }
func (w *dnsTestWriter) WriteMsg(m *dns.Msg) error {
	w.messages = append(w.messages, m.Copy())
	return nil
}
func (w *dnsTestWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w *dnsTestWriter) Close() error                { return nil }
func (w *dnsTestWriter) TsigStatus() error           { return nil }
func (w *dnsTestWriter) TsigTimersOnly(bool)         {}
func (w *dnsTestWriter) Hijack()                     {}
func dnsFixture(t *testing.T) *dnsAuthority {
	t.Helper()
	a := newDNSAuthority()
	zone, err := buildDNSZone(nil, []dnsRecord{
		{Name: "known.u.isucon.dev", Type: "A", Content: "192.0.2.7", TTL: 0},
		{Name: "u.isucon.dev", Type: "SOA", Content: "ns.u.isucon.dev. hostmaster.u.isucon.dev. 1 3600 600 86400 0", TTL: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	a.zone.Store(zone)
	return a
}
func dnsRequest(a *dnsAuthority, name string, qtype uint16, tcp bool) *dnsTestWriter {
	remote := net.Addr(&net.UDPAddr{IP: net.IPv4(192, 0, 2, 100), Port: 12345})
	if tcp {
		remote = &net.TCPAddr{IP: net.IPv4(192, 0, 2, 100), Port: 12345}
	}
	w := &dnsTestWriter{remote: remote}
	m := new(dns.Msg)
	m.SetQuestion(name, qtype)
	a.ServeDNS(w, m)
	return w
}
func TestDNSNegativeFloodDoesNotThrottleKnownNames(t *testing.T) {
	a := dnsFixture(t)
	dropped := 0
	for n := 0; n < 500; n++ {
		w := dnsRequest(a, fmt.Sprintf("missing%d.u.isucon.dev.", n), dns.TypeA, false)
		if len(w.messages) == 0 {
			dropped++
		} else if w.messages[0].Rcode != dns.RcodeNameError || len(w.messages[0].Answer) != 0 {
			t.Fatal("unknown must not receive an A record")
		}
		known := dnsRequest(a, "KNOWN.U.ISUCON.DEV.", dns.TypeA, false)
		if len(known.messages) != 1 || !known.messages[0].Authoritative || len(known.messages[0].Answer) != 1 {
			t.Fatal("known query throttled or not authoritative")
		}
		if known.messages[0].Answer[0].(*dns.A).A.String() != "192.0.2.7" {
			t.Fatal("wrong address")
		}
	}
	if dropped == 0 {
		t.Fatal("negative flood was not limited")
	}
	tcp := dnsRequest(a, "missing.u.isucon.dev.", dns.TypeA, true)
	if len(tcp.messages) != 1 || tcp.messages[0].Rcode != dns.RcodeNameError {
		t.Fatal("TCP negative fallback lost")
	}
}
func TestDNSPublicationKeepsOldSnapshotAndConcurrentReads(t *testing.T) {
	a := dnsFixture(t)
	before := a.zone.Load()
	after, err := buildDNSZone(before, []dnsRecord{{Name: "new.u.isucon.dev", Type: "A", Content: "192.0.2.8", TTL: 0}})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := before.names["new.u.isucon.dev."]; exists {
		t.Fatal("published old snapshot mutated")
	}
	a.zone.Store(after)
	if len(dnsRequest(a, "new.u.isucon.dev.", dns.TypeA, false).messages[0].Answer) != 1 {
		t.Fatal("new name invisible")
	}
	var group sync.WaitGroup
	for n := 0; n < 8; n++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for j := 0; j < 500; j++ {
				w := dnsRequest(a, "known.u.isucon.dev.", dns.TypeA, false)
				if len(w.messages) != 1 || len(w.messages[0].Answer) != 1 {
					t.Error("known query lost during publication")
					return
				}
			}
		}()
	}
	for n := 0; n < 100; n++ {
		update, err := buildDNSZone(after, []dnsRecord{{Name: fmt.Sprintf("addition%d.u.isucon.dev", n), Type: "A", Content: "192.0.2.9", TTL: 0}})
		if err != nil {
			t.Fatal(err)
		}
		a.zone.Store(update)
	}
	group.Wait()
}
func TestDNSNegativeClientStateIsBounded(t *testing.T) {
	a := dnsFixture(t)
	m := new(dns.Msg)
	m.SetQuestion("missing.u.isucon.dev.", dns.TypeA)
	for n := 0; n < 6000; n++ {
		w := &dnsTestWriter{remote: &net.UDPAddr{IP: net.IPv4(192, byte(n/65536), byte(n/256), byte(n)), Port: 12345}}
		a.ServeDNS(w, m)
	}
	if len(a.clients) > 4096 {
		t.Fatal("unbounded client limiter state")
	}
}
