package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"

	"code.rocket9labs.com/tslocum/bgammon"
)

type Client struct {
	Username   string
	Password   string
	Events     chan interface{}
	Out        chan []byte
	address    string
	connecting bool
	conn       *net.TCPConn
}

func NewClient(address, username, password string) *Client {
	const bufferSize = 10
	return &Client{
		address:  address,
		Username: username,
		Password: password,
		Events:   make(chan interface{}, bufferSize),
		Out:      make(chan []byte, bufferSize),
	}
}

func (c *Client) Connect() {
	dialConn, err := net.Dial("tcp", c.address)
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	conn := dialConn.(*net.TCPConn)

	// Log in.
	loginInfo := c.Username
	if c.Username != "" && c.Password != "" {
		loginInfo = fmt.Sprintf("%s %s", strings.ReplaceAll(c.Username, " ", "_"), strings.ReplaceAll(c.Password, " ", "_"))
	}
	conn.Write([]byte(fmt.Sprintf("lj bgammon-cli %s\nlist\n", loginInfo)))

	// Read a single line of text and parse remaining output as JSON.
	buf := make([]byte, 1)
	var readBytes int
	for {
		conn.Read(buf)

		if buf[0] == '\n' {
			break
		}

		readBytes++
		if readBytes == 512 {
			panic("failed to read server welcome message")
		}
	}

	c.conn = conn

	go c.handleWrite()
	c.handleRead()

	l("*** Disconnected.")
}

func (c *Client) handleWrite() {
	for buf := range c.Out {
		c.conn.Write(append(buf, '\n'))

		if debug > 0 {
			l(fmt.Sprintf("-> %s", buf))
		}
	}
}

func (c *Client) handleRead() {
	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		ev, err := bgammon.DecodeEvent(scanner.Bytes())
		if err != nil {
			log.Printf("message: %s", scanner.Bytes())
			panic(err)
		}
		c.Events <- ev

		if debug > 0 {
			l(fmt.Sprintf("<- %s", scanner.Bytes()))
		}
	}
}

func (c *Client) LoggedIn() bool {
	return c.conn != nil
}
