package udp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type ClientConfig struct {
	RemoteAddress string
	LocalAddres   string
}

type Client struct {
	config ClientConfig
	conn   *net.UDPConn
	mu     sync.Mutex
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewClient(config ClientConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *Client) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	laddr, err := net.ResolveUDPAddr("udp", c.config.LocalAddres)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

func (c *Client) write() {
	defer c.wg.Done()

	raddr, err := net.ResolveUDPAddr("udp", c.config.RemoteAddress)
	if err != nil {
		log.Println("Failed to resolve the remote address")
		c.End()
		return
	}

	inputCh := make(chan string)
	errCh := make(chan error, 1)
	defer close(inputCh)
	defer close(errCh)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			inputCh <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		}

	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case err := <-errCh:
			if err != nil {
				log.Printf("Error Scaning input: %v", err)
				continue
			}
		case msg := <-inputCh:
			if c.conn == nil {
				log.Println("Cannot write: connection is not active")
				c.End()
				return
			}

			if msg == "/quit" {
				log.Println("Quiting...")
				c.End()
				return
			}

			_, err := c.conn.WriteToUDP([]byte(msg+"\n"), raddr)
			if err != nil {
				log.Println("Failed to write")
				c.End()
				return
			}

		}
	}
}

func (c *Client) read() {
	defer c.wg.Done()

	buffer := make([]byte, 65535)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			err := c.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			if err != nil {
				log.Printf("Failed to set read deadline: %v", err)
				c.End()
				return
			}

			n, server, err := c.conn.ReadFromUDP(buffer)
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					continue
				}
				log.Printf("Failed to read: %v", err)
				c.End()
				return
			}

			fmt.Printf("%s:%s", server, buffer[:n])
		}
	}
}

func (c *Client) Start() {
	if err := c.connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	c.wg.Add(2)
	go c.write()
	go c.read()
	c.wg.Wait()
}

func (c *Client) End() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cancel()
	time.Sleep(250 * time.Millisecond)

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}
