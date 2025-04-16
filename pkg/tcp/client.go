package tcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type ClientConfig struct {
	Address     string
	DialTimeout time.Duration
}

type Client struct {
	config ClientConfig
	conn   net.Conn
	mu     sync.Mutex
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func DefaultClientConfig(address string) ClientConfig {
	return ClientConfig{
		Address:     address,
		DialTimeout: 10 * time.Second,
	}
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
	conn, err := net.DialTimeout("tcp", c.config.Address, c.config.DialTimeout)
	if err != nil {
		c.mu.Unlock()
		return err
	}
	c.conn = conn
	c.mu.Unlock()

	return nil
}

func (c *Client) read() {
	defer c.wg.Done()
	buffer := make([]byte, 4096)

	for {
		select {
		case <-c.ctx.Done():
			// log.Printf("Read aborted: (%v)", c.ctx.Err())
			return
		default:
			if c.conn == nil {
				log.Println("Cannot read: connection is not active")
				c.End()
				return
			}

			c.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))

			n, err := c.conn.Read(buffer)
			if err != nil {
				if errors.Is(err, io.EOF) {
					log.Println("Connection closed by remote host (EOF)")
					c.End()
					return
				} else if errors.Is(err, os.ErrDeadlineExceeded) {
					continue
				} else {
					log.Printf("Failed to read: %v", err)
					c.End()
					return
				}
			}

			fmt.Printf("%s\n", buffer[:n-1])
		}
	}
}

func (c *Client) write() {
	defer c.wg.Done()

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
			// log.Printf("Write aborted: %v", c.ctx.Err())
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

			_, err := c.conn.Write([]byte(msg + "\n"))
			if err != nil {
				log.Println("Failed to write")
				c.End()
				return
			}

		}
	}
}

func (c *Client) Start() {
	err := c.connect()
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	c.wg.Add(2)

	go c.read()
	go c.write()

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
