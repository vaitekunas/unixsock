package client

import (
  "github.com/vaitekunas/unixsock"
  "fmt"
  "net"
  "time"
)

// UnixSockClient represents a client meant to communicate with a UnixSockSrv
type UnixSockClient interface {

  // Send sends a command to a UnixSockSrv
  Send(cmd string, args unixsock.Args, respond, close bool) (*unixsock.Response, error)

  // Options sets the options of the underlying communications
  Options(maxLength int, timeout time.Duration, respond, close bool)

  // Quit closes the client
  Quit()
}

// unixSockClient implements the UnixSockClient interface
type unixSockClient struct {
  maxLength int
  timeout time.Duration
  respond, close bool
  unixSockPath string
  conn net.Conn
  conntime time.Time
}

// New creates a new UnixSockClient connecting to the UnixSockPath
func New(UnixSockPath string) (UnixSockClient, error) {

  return &unixSockClient{
    maxLength: 1 << 20,
    timeout: 5*time.Second,
    respond: true,
    close: true,
    unixSockPath: UnixSockPath,
  }, nil

}

// Options sets communicator options
func (u *unixSockClient) Options(maxLength int, timeout time.Duration, respond, close bool)     {
  u.respond = respond
  u.close = close
  u.maxLength = maxLength
  u.timeout = timeout
}

// Send sends a single message to a UnixSockSrv
func (u *unixSockClient) Send(cmd string, args unixsock.Args, respond, close bool) (*unixsock.Response, error) {

  // Connect to the socket
  if err := u.reconnect(); err != nil {
    return nil, fmt.Errorf("Send: could not connect to the unix socket: %s", err.Error())
  }

  // Construct new message
  msg := unixsock.NewSender(u.conn, cmd, args, respond, close)

  // Set options
  msg.Options(u.maxLength, u.timeout, respond, close)

  // Send
	if err := msg.Send(); err != nil {
		return nil, fmt.Errorf("Send: could not send a command: %s", err.Error())
	}

  // Wait for response
  if respond {
  	if err := msg.Receive(); err != nil {
  		return nil, fmt.Errorf("Send: failed receiving a response: %s", err.Error())
  	}

  	return msg.GetResponse(), nil
  }

  return nil, nil

}

// reconnect reestablishes the connection to the unix socket
func (u *unixSockClient) reconnect() error {

  if u.conn != nil && time.Now().Unix() - u.conntime.Unix() < 5 {
    return nil
  }

  c, err := net.Dial("unix", u.unixSockPath)
  if err != nil {
    return fmt.Errorf("reconnect: could not connect to socket: %s", err.Error())
  }

  u.conn = c
  u.conntime = time.Now()

  return nil

}

// Quit closes the connection
func (u *unixSockClient) Quit() {
  if u.conn != nil {
    u.conn.Close()
  }
}
