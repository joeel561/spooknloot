package netcode

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
)

const (
	alpnProtocol  = "spooknloot"
	maxFrameSize  = 1 << 20
	handshakeWait = 5 * time.Second
)

var quicConfig = &quic.Config{
	EnableDatagrams: true,
	MaxIdleTimeout:  10 * time.Second,
	KeepAlivePeriod: 2 * time.Second,
}

// eventQueue collects events from network goroutines until the game loop polls them.
type eventQueue struct {
	mu     sync.Mutex
	events []Event
}

func (q *eventQueue) push(e Event) {
	q.mu.Lock()
	q.events = append(q.events, e)
	q.mu.Unlock()
}

func (q *eventQueue) poll() []Event {
	q.mu.Lock()
	ev := q.events
	q.events = nil
	q.mu.Unlock()
	return ev
}

// quicPeer is one QUIC connection with a single bidirectional stream used for
// reliable, length-prefixed frames. Unreliable messages use QUIC datagrams.
type quicPeer struct {
	id     PeerID
	conn   *quic.Conn
	stream *quic.Stream
	sendMu sync.Mutex
}

func (p *quicPeer) send(data []byte, reliable bool) error {
	if !reliable {
		return p.conn.SendDatagram(data)
	}
	if len(data) > maxFrameSize {
		return fmt.Errorf("frame too large: %d bytes", len(data))
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
	p.sendMu.Lock()
	defer p.sendMu.Unlock()
	if _, err := p.stream.Write(hdr[:]); err != nil {
		return err
	}
	_, err := p.stream.Write(data)
	return err
}

// run reads reliable frames and datagrams until the connection dies, then
// reports a single EventDisconnected.
func (p *quicPeer) run(q *eventQueue, onClose func()) {
	go func() {
		for {
			data, err := p.conn.ReceiveDatagram(p.conn.Context())
			if err != nil {
				return
			}
			q.push(Event{Kind: EventMessage, Peer: p.id, Data: data})
		}
	}()

	var hdr [4]byte
	var err error
	for {
		if _, err = io.ReadFull(p.stream, hdr[:]); err != nil {
			break
		}
		n := binary.BigEndian.Uint32(hdr[:])
		if n > maxFrameSize {
			err = fmt.Errorf("frame too large: %d bytes", n)
			break
		}
		buf := make([]byte, n)
		if _, err = io.ReadFull(p.stream, buf); err != nil {
			break
		}
		q.push(Event{Kind: EventMessage, Peer: p.id, Data: buf})
	}
	p.conn.CloseWithError(0, "")
	if onClose != nil {
		onClose()
	}
	q.push(Event{Kind: EventDisconnected, Peer: p.id, Err: closeReason(err)})
}

// closeReason turns a QUIC close into a readable error. A clean local close
// yields nil; a close by the remote side carries its reason text.
func closeReason(err error) error {
	var appErr *quic.ApplicationError
	if errors.As(err, &appErr) {
		if !appErr.Remote {
			return nil
		}
		if appErr.ErrorMessage != "" {
			return errors.New(appErr.ErrorMessage)
		}
		return errors.New("connection closed")
	}
	var idleErr *quic.IdleTimeoutError
	if errors.As(err, &idleErr) {
		return errors.New("connection timed out")
	}
	if errors.Is(err, io.EOF) {
		return errors.New("connection closed")
	}
	return err
}

// QUICHost accepts client connections.
type QUICHost struct {
	udp    *net.UDPConn
	tr     *quic.Transport
	ln     *quic.Listener
	queue  eventQueue
	mu     sync.Mutex
	peers  map[PeerID]*quicPeer
	nextID PeerID
	ctx    context.Context
	cancel context.CancelFunc
}

// Host starts listening for clients on a UDP address such as ":47778"
// (all interfaces) or "127.0.0.1:47778" (this machine only).
func Host(listenAddr string) (*QUICHost, error) {
	tlsConf, err := selfSignedTLS()
	if err != nil {
		return nil, err
	}
	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, err
	}
	// We own the socket (instead of quic.ListenAddr) so Close releases
	// the port immediately and the player can host again right away.
	udp, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	tr := &quic.Transport{Conn: udp}
	ln, err := tr.Listen(tlsConf, quicConfig)
	if err != nil {
		tr.Close()
		udp.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	h := &QUICHost{
		udp:    udp,
		tr:     tr,
		ln:     ln,
		peers:  make(map[PeerID]*quicPeer),
		nextID: HostPeerID + 1,
		ctx:    ctx,
		cancel: cancel,
	}
	go h.acceptLoop()
	return h, nil
}

func (h *QUICHost) acceptLoop() {
	for {
		conn, err := h.ln.Accept(h.ctx)
		if err != nil {
			return
		}
		go h.handshake(conn)
	}
}

func (h *QUICHost) handshake(conn *quic.Conn) {
	// The client opens the stream and writes its first message right away,
	// which is what makes the stream visible here.
	ctx, cancel := context.WithTimeout(h.ctx, handshakeWait)
	stream, err := conn.AcceptStream(ctx)
	cancel()
	if err != nil {
		conn.CloseWithError(0, "handshake timeout")
		return
	}

	h.mu.Lock()
	p := &quicPeer{id: h.nextID, conn: conn, stream: stream}
	h.nextID++
	h.peers[p.id] = p
	h.mu.Unlock()

	h.queue.push(Event{Kind: EventConnected, Peer: p.id})
	p.run(&h.queue, func() {
		h.mu.Lock()
		delete(h.peers, p.id)
		h.mu.Unlock()
	})
}

func (h *QUICHost) peer(id PeerID) *quicPeer {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.peers[id]
}

func (h *QUICHost) Send(to PeerID, data []byte, reliable bool) error {
	p := h.peer(to)
	if p == nil {
		return fmt.Errorf("unknown peer %d", to)
	}
	return p.send(data, reliable)
}

func (h *QUICHost) Broadcast(data []byte, reliable bool) {
	h.mu.Lock()
	peers := make([]*quicPeer, 0, len(h.peers))
	for _, p := range h.peers {
		peers = append(peers, p)
	}
	h.mu.Unlock()
	for _, p := range peers {
		_ = p.send(data, reliable)
	}
}

func (h *QUICHost) Disconnect(peer PeerID, reason string) {
	if p := h.peer(peer); p != nil {
		p.conn.CloseWithError(1, reason)
	}
}

func (h *QUICHost) Poll() []Event { return h.queue.poll() }

func (h *QUICHost) Close() {
	h.mu.Lock()
	for _, p := range h.peers {
		p.conn.CloseWithError(0, "host closed the game")
	}
	h.mu.Unlock()
	h.cancel()
	h.ln.Close()
	h.tr.Close()
	h.udp.Close()
}

// QUICClient is a single connection to a host.
type QUICClient struct {
	udp   *net.UDPConn
	tr    *quic.Transport
	peer  *quicPeer
	queue eventQueue
}

// Dial connects to a host and sends hello as the first reliable message.
// It blocks until the connection is established or the timeout expires.
func Dial(addr string, hello []byte, timeout time.Duration) (*QUICClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	tlsConf := &tls.Config{
		// LAN hosts use throwaway self-signed certificates. Internet play
		// will need certificate pinning or a relay that vouches for the host.
		InsecureSkipVerify: true,
		NextProtos:         []string{alpnProtocol},
	}
	remote, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	// Bind to loopback when the host is on this machine, which keeps
	// Windows Firewall from asking about local-only connections.
	local := &net.UDPAddr{}
	if remote.IP.IsLoopback() {
		local.IP = remote.IP
	}
	udp, err := net.ListenUDP("udp", local)
	if err != nil {
		return nil, err
	}
	c := &QUICClient{udp: udp, tr: &quic.Transport{Conn: udp}}
	conn, err := c.tr.Dial(ctx, remote, tlsConf, quicConfig)
	if err != nil {
		c.closeSocket()
		return nil, err
	}
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		conn.CloseWithError(0, "")
		c.closeSocket()
		return nil, err
	}
	c.peer = &quicPeer{id: HostPeerID, conn: conn, stream: stream}
	if err := c.peer.send(hello, true); err != nil {
		conn.CloseWithError(0, "")
		c.closeSocket()
		return nil, err
	}
	go c.peer.run(&c.queue, c.closeSocket)
	return c, nil
}

func (c *QUICClient) closeSocket() {
	c.tr.Close()
	c.udp.Close()
}

func (c *QUICClient) Send(_ PeerID, data []byte, reliable bool) error {
	return c.peer.send(data, reliable)
}

func (c *QUICClient) Broadcast(data []byte, reliable bool) { _ = c.peer.send(data, reliable) }

func (c *QUICClient) Disconnect(PeerID, string) {}

func (c *QUICClient) Poll() []Event { return c.queue.poll() }

func (c *QUICClient) Close() {
	c.peer.conn.CloseWithError(0, "client left")
	c.closeSocket()
}

func selfSignedTLS() (*tls.Config, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
		NextProtos:   []string{alpnProtocol},
	}, nil
}
