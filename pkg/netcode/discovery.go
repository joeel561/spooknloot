package netcode

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

const (
	DiscoveryPort    = 47777
	DefaultGamePort  = 47778
	announceInterval = time.Second
	announceTTL      = 4 * time.Second
	discoveryMagic   = "spooknloot-lan"
)

// Announcement is what a host broadcasts on the LAN once per second.
type Announcement struct {
	Magic      string `json:"magic"`
	Version    string `json:"version"`
	Name       string `json:"name"`
	Port       int    `json:"port"`
	Players    int    `json:"players"`
	MaxPlayers int    `json:"maxPlayers"`
	InGame     bool   `json:"inGame"`
}

// Announcer periodically broadcasts an Announcement on every local network.
type Announcer struct {
	mu   sync.Mutex
	ann  Announcement
	conn *net.UDPConn
	stop chan struct{}
}

func StartAnnouncer(ann Announcement) (*Announcer, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil, err
	}
	ann.Magic = discoveryMagic
	a := &Announcer{ann: ann, conn: conn, stop: make(chan struct{})}
	go a.loop()
	return a, nil
}

// Update changes the announced lobby info (player count, in-game state...).
func (a *Announcer) Update(fn func(*Announcement)) {
	a.mu.Lock()
	fn(&a.ann)
	a.mu.Unlock()
}

func (a *Announcer) loop() {
	t := time.NewTicker(announceInterval)
	defer t.Stop()
	for {
		a.mu.Lock()
		data, _ := json.Marshal(a.ann)
		a.mu.Unlock()
		for _, addr := range broadcastAddrs() {
			_, _ = a.conn.WriteToUDP(data, &net.UDPAddr{IP: addr, Port: DiscoveryPort})
		}
		select {
		case <-a.stop:
			return
		case <-t.C:
		}
	}
}

func (a *Announcer) Close() {
	close(a.stop)
	a.conn.Close()
}

// broadcastAddrs returns the directed broadcast address of every active IPv4
// interface plus the limited broadcast address. Windows only sends
// 255.255.255.255 out of one interface, so directed ones are needed too.
func broadcastAddrs() []net.IP {
	out := []net.IP{net.IPv4bcast}
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagBroadcast == 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipn.IP.To4()
			if ip4 == nil || ip4.IsLoopback() {
				continue
			}
			mask := net.IP(ipn.Mask).To4()
			if mask == nil {
				continue
			}
			bc := make(net.IP, 4)
			for i := range bc {
				bc[i] = ip4[i] | ^mask[i]
			}
			out = append(out, bc)
		}
	}
	return out
}

// FoundGame is a host seen on the LAN.
type FoundGame struct {
	Announcement
	Addr     string // host:port to dial
	lastSeen time.Time
}

// Browser listens for host announcements.
type Browser struct {
	mu    sync.Mutex
	games map[string]*FoundGame
	conn  *net.UDPConn
}

func StartBrowser() (*Browser, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: DiscoveryPort})
	if err != nil {
		return nil, fmt.Errorf("listen on discovery port %d: %w", DiscoveryPort, err)
	}
	b := &Browser{games: make(map[string]*FoundGame), conn: conn}
	go b.loop()
	return b, nil
}

func (b *Browser) loop() {
	buf := make([]byte, 2048)
	for {
		n, from, err := b.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		var ann Announcement
		if json.Unmarshal(buf[:n], &ann) != nil || ann.Magic != discoveryMagic {
			continue
		}
		addr := net.JoinHostPort(from.IP.String(), fmt.Sprint(ann.Port))
		b.mu.Lock()
		b.games[addr] = &FoundGame{Announcement: ann, Addr: addr, lastSeen: time.Now()}
		b.mu.Unlock()
	}
}

// Games returns the currently visible hosts, sorted by name.
func (b *Browser) Games() []FoundGame {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]FoundGame, 0, len(b.games))
	for k, g := range b.games {
		if time.Since(g.lastSeen) > announceTTL {
			delete(b.games, k)
			continue
		}
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Addr < out[j].Addr
	})
	return out
}

func (b *Browser) Close() { b.conn.Close() }
