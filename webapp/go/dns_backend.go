package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/miekg/dns"
	"golang.org/x/time/rate"
)

// The SQL registry remains the durable source of truth and pdnsutil validates names.
var dnsLifecycle sync.RWMutex
var dnsRegistryConn *sqlx.DB
var dnsAuthorityService atomic.Pointer[dnsAuthority]

type dnsRecord struct {
	ID      int64  `db:"id"`
	Name    string `db:"name"`
	Type    string `db:"type"`
	Content string `db:"content"`
	TTL     uint32 `db:"ttl"`
}
type dnsZone struct {
	names map[string][]dns.RR
	soa   dns.RR
}
type dnsRate struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}
type dnsAuthority struct {
	zone      atomic.Pointer[dnsZone]
	publishMu sync.Mutex
	seenIDs   map[int64]struct{}
	mu        sync.Mutex
	clients   map[string]*dnsRate
	fallback  *rate.Limiter
	positive  atomic.Uint64
	negative  atomic.Uint64
	dropped   atomic.Uint64
}

func newDNSAuthority() *dnsAuthority {
	return &dnsAuthority{clients: make(map[string]*dnsRate), fallback: rate.NewLimiter(100, 100)}
}
func (a *dnsAuthority) allowNegative(w dns.ResponseWriter) bool {
	// UDP negative-response limiting is general abuse protection, never a name/IP pattern.
	// TCP can still retrieve an authoritative negative answer.
	if strings.HasPrefix(w.RemoteAddr().Network(), "tcp") {
		return true
	}
	ip, _, err := net.SplitHostPort(w.RemoteAddr().String())
	if err != nil {
		ip = w.RemoteAddr().String()
	}
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	entry := a.clients[ip]
	if entry == nil {
		if len(a.clients) >= 4096 {
			for key, value := range a.clients {
				if now.Sub(value.lastSeen) > time.Minute {
					delete(a.clients, key)
				}
			}
			if len(a.clients) >= 4096 {
				return a.fallback.AllowN(now, 1)
			}
		}
		entry = &dnsRate{limiter: rate.NewLimiter(100, 100)}
		a.clients[ip] = entry
	}
	entry.lastSeen = now
	return entry.limiter.AllowN(now, 1)
}
func (a *dnsAuthority) ServeDNS(w dns.ResponseWriter, request *dns.Msg) {
	response := new(dns.Msg)
	response.SetReply(request)
	response.Authoritative = true
	response.Compress = true
	if request.Opcode != dns.OpcodeQuery || len(request.Question) != 1 {
		response.Rcode = dns.RcodeRefused
		_ = w.WriteMsg(response)
		return
	}
	question := request.Question[0]
	if question.Qclass != dns.ClassINET {
		response.Rcode = dns.RcodeRefused
		_ = w.WriteMsg(response)
		return
	}
	zone := a.zone.Load()
	if zone == nil {
		response.Rcode = dns.RcodeServerFailure
		_ = w.WriteMsg(response)
		return
	}
	records, exists := zone.names[dns.CanonicalName(question.Name)]
	for _, record := range records {
		if record.Header().Rrtype == question.Qtype || question.Qtype == dns.TypeANY {
			response.Answer = append(response.Answer, dns.Copy(record))
		}
	}
	if len(response.Answer) > 0 {
		a.positive.Add(1)
		_ = w.WriteMsg(response)
		return
	}
	if !a.allowNegative(w) {
		a.dropped.Add(1)
		return
	}
	if !exists {
		response.Rcode = dns.RcodeNameError
	}
	if zone.soa != nil {
		response.Ns = []dns.RR{dns.Copy(zone.soa)}
	}
	a.negative.Add(1)
	_ = w.WriteMsg(response)
}
func buildDNSZone(old *dnsZone, records []dnsRecord) (*dnsZone, error) {
	zone := &dnsZone{names: make(map[string][]dns.RR)}
	if old != nil {
		zone.soa = old.soa
		for name, values := range old.names {
			zone.names[name] = values
		}
	}
	for _, record := range records {
		rr, err := dns.NewRR(fmt.Sprintf("%s %d IN %s %s", dns.Fqdn(record.Name), record.TTL, record.Type, record.Content))
		if err != nil {
			return nil, fmt.Errorf("DNS record %q: %w", record.Name, err)
		}
		key := dns.CanonicalName(rr.Header().Name)
		// Neither an old snapshot nor its record slices may be modified after publication.
		values := append([]dns.RR(nil), zone.names[key]...)
		zone.names[key] = append(values, rr)
		if rr.Header().Rrtype == dns.TypeSOA && key == "u.isucon.dev." {
			zone.soa = rr
		}
	}
	return zone, nil
}

const dnsRecordsSQL = `SELECT r.id,r.name,r.type,r.content,r.ttl FROM records r
 JOIN domains d ON d.id=r.domain_id WHERE d.name='u.isucon.dev' AND r.disabled=0`

func refreshDNSZone() error {
	authority := dnsAuthorityService.Load()
	if authority == nil {
		return nil
	}
	var records []dnsRecord
	if err := dnsRegistryConn.Select(&records, dnsRecordsSQL+" ORDER BY r.id"); err != nil {
		return err
	}
	zone, err := buildDNSZone(nil, records)
	if err != nil {
		return err
	}
	authority.publishMu.Lock()
	defer authority.publishMu.Unlock()
	authority.seenIDs = make(map[int64]struct{}, len(records))
	for _, record := range records {
		authority.seenIDs[record.ID] = struct{}{}
	}
	authority.zone.Store(zone)
	return nil
}
func registerDNS(name string) ([]byte, error) {
	command := exec.Command("pdnsutil", "add-record", "u.isucon.dev", name, "A", "0", powerDNSSubdomainAddress)
	authority := dnsAuthorityService.Load()
	if authority == nil {
		return command.CombinedOutput()
	}
	var maximum int64
	if err := dnsRegistryConn.Get(&maximum, "SELECT COALESCE(MAX(id),0) FROM records"); err != nil {
		return nil, err
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return output, err
	}
	var records []dnsRecord
	if err := dnsRegistryConn.Select(&records, dnsRecordsSQL+" AND r.id>? ORDER BY r.id", maximum); err != nil {
		return output, err
	}
	return output, authority.publishRecords(records)
}

// Each caller reads its lower ID bound before its own CLI insert. A global cursor
// would lose older IDs whose transactions commit after a newer ID is published.
func (a *dnsAuthority) publishRecords(records []dnsRecord) error {
	a.publishMu.Lock()
	defer a.publishMu.Unlock()
	fresh := make([]dnsRecord, 0, len(records))
	for _, record := range records {
		if _, exists := a.seenIDs[record.ID]; !exists {
			fresh = append(fresh, record)
		}
	}
	if len(fresh) == 0 {
		return nil
	}
	zone, err := buildDNSZone(a.zone.Load(), fresh)
	if err != nil {
		return err
	}
	if a.seenIDs == nil {
		a.seenIDs = make(map[int64]struct{})
	}
	for _, record := range fresh {
		a.seenIDs[record.ID] = struct{}{}
	}
	a.zone.Store(zone)
	return nil
}
func startDNSAuthority() error {
	if os.Getenv("ISUCON13_POWERDNS_BACKEND") != "go" {
		return nil
	}
	config := mysql.NewConfig()
	config.User = "isudns"
	config.Passwd = "isudns"
	config.DBName = "isudns"
	config.Net = "unix"
	config.Addr = "/var/run/mysqld/mysqld.sock"
	config.InterpolateParams = true
	conn, err := sqlx.Open("mysql", config.FormatDSN())
	if err != nil {
		return err
	}
	conn.SetMaxOpenConns(2)
	conn.SetMaxIdleConns(2)
	if err := conn.Ping(); err != nil {
		conn.Close()
		return err
	}
	dnsRegistryConn = conn
	authority := newDNSAuthority()
	dnsAuthorityService.Store(authority)
	if err := refreshDNSZone(); err != nil {
		return err
	}
	udp, err := net.ListenPacket("udp", ":53")
	if err != nil {
		return err
	}
	tcp, err := net.Listen("tcp", ":53")
	if err != nil {
		udp.Close()
		return err
	}
	for _, server := range []*dns.Server{{PacketConn: udp, Handler: authority}, {Listener: tcp, Handler: authority}} {
		go func(server *dns.Server) {
			if err := server.ActivateAndServe(); err != nil {
				fmt.Fprintln(os.Stderr, "DNS server failed:", err)
				os.Exit(1)
			}
		}(server)
	}
	return nil
}
