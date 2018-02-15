package network

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Synaxis/bfheroesFesl/storage/level"
	"github.com/sirupsen/logrus"
)

type Client struct {
	name       string
	Conn       net.Conn
	recvBuffer []byte
	eventChan  chan ClientEvent
	IsActive   bool
	reader     *bufio.Reader
	HashState  *level.State
	IpAddr     net.Addr
	State      ClientState

	Options ClientOptions
}

type ClientOptions struct {
	FESL bool
}

type Clients struct {
	mu        *sync.Mutex
	connected map[ClientKey]*Client
}

func newClient() *Clients {
	return &Clients{
		connected: make(map[ClientKey]*Client, 500),
		mu:        new(sync.Mutex),
	}
}

func (cls *Clients) Add(cl *Client) {
	cls.mu.Lock()
	cls.connected[cl.Key()] = cl
	cls.mu.Unlock()
}

func (cls *Clients) Remove(cl *Client) {
	cls.mu.Lock()
	delete(cls.connected, cl.Key())
	cls.mu.Unlock()
}

type ClientKey struct {
	name, addr string
}

func (ck *ClientKey) String() string {
	return fmt.Sprintf("%s_%s", ck.name, ck.addr)
}

func newClientTCP(name string, conn net.Conn, fesl bool) *Client {
	return &Client{
		name:      name,
		Conn:      conn,
		IpAddr:    conn.RemoteAddr(),
		eventChan: make(chan ClientEvent, 500),
		reader:    bufio.NewReader(conn),
		IsActive:  true,
		Options: ClientOptions{
			FESL: fesl,
		},
	}
}

func newClientTLS(name string, conn *tls.Conn) *Client {
	return &Client{
		name:      name,
		Conn:      conn,
		IpAddr:    conn.RemoteAddr(),
		IsActive:  true,
		eventChan: make(chan ClientEvent, 500),
		Options: ClientOptions{
			FESL: true, // Always true
		},
	}
}

func (client *Client) handleRequestTLS() {
	client.IsActive = true
	buf := make([]byte, 8096) // buffer

	for client.IsActive {
		n, err := client.readBuf(buf)
		if err != nil {
			return
		}

		client.readTLSPacket(buf[:n])

		buf = make([]byte, 8096) // new fresh buffer
	}
}

func (client *Client) handleRequest() {
	client.IsActive = true
	buf := make([]byte, 8096) // buffer

	for client.IsActive {
		n, err := client.readBuf(buf)
		if err != nil {
			return
		}

		client.readFESL(buf[:n])
		buf = make([]byte, 8096) // new fresh buffer
	}
}

func (client *Client) readBuf(buf []byte) (int, error) {
	n, err := client.Conn.Read(buf)
	if err != nil {
		if err != io.EOF {
			logrus.Errorf("Error: %v on client %s", err, client.IpAddr)
			client.eventChan <- client.FireClose()
			return 0, err
		}
		client.eventChan <- client.FireClose()
		return 0, err
	}
	return n, nil
}

func (c *Client) Key() ClientKey {
	return ClientKey{c.name, c.IpAddr.String()}
}

func (c *Client) Close() {
	logrus.Printf("%s:Client Closing.", c.name)
	c.eventChan <- ClientEvent{Name: "close", Data: c}
	c.Conn.Close()
	c.IsActive = false
	// close(c.eventChan)
}

type ClientState struct {
	ServerChallenge string
	ClientChallenge string
	ClientResponse  string
	Username        string
	PlyName         string
	PlyEmail        string
	PlyCountry      string
	PlyPid          int
	Sessionkey      int
	Confirmed       bool
	IpAddress       net.Addr
	HasLogin        bool
	ProfileSent     bool
	LoggedOut       bool
	HeartTicker     *time.Ticker
}
