package unixsock

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

// Status constants
const (
	STATUS_OK   = "success"
	STATUS_FAIL = "failure"
)


// Args is a shorthand for a map of strings to interfaces
type Args map[string]interface{}

// Response contains a response from the UnixManager
type Response struct {
	Status  string `json:"status"`
	Error   string `json:"error"`
	Payload string `json:"payload"`
}

// Communicator represents a command sent over the unix socket
type Communicator interface {

	// Options set some options on the sending/receiving
	Options(maxLength int, timeout time.Duration, respond, close bool)

	// Receive reads all the data (a SocketMEssage) from a unix socket and stores
	// all the content inside the receiving SocketMessage
	Receive() error

	// Send sends this SocketMessage over the unix socket
	Send() error

	// GetCmd returns message command
	GetCmd() string

	// GetArgs returns command arguments
	GetArgs() Args

	// GetResponse returns message's response
	GetResponse() *Response

	// SetResponse sets the message response
	SetResponse(*Response)

	// ShouldRespond informs the message handler that a response is expected
	ShouldRespond() bool

	// ShouldClose informs the message handler that there's not going to be any
	// further communication and the connection can be closed
	ShouldClose() bool
}

// NewSender creates a blank message for the sender
func NewSender(conn net.Conn, cmd string, args Args, respond, close bool) Communicator {
	return newCommunicator(conn, cmd, args, &Response{}, respond, close)
}

// NewReceiver creates a blank message for the receiver
func NewReceiver(conn net.Conn) Communicator {
	return newCommunicator(conn, "", Args{}, &Response{}, true, true)
}

// newCommunicator creates a new socket message with default options
func newCommunicator(conn net.Conn, cmd string, args Args, resp *Response, respond, close bool) *communicator {
	return &communicator{
		Cmd:       cmd,
		Args:      args,
		Response:  resp,
		Respond:   respond,
		Close:     close,
		conn:      conn,
		maxLength: 1 << 20,
		timeout:   5 * time.Second,
	}
}

// communicator represents a command sent over the unix socket
type communicator struct {
	Cmd      string    `json:"cmd"`      // Command
	Args     Args      `json:"args"`     // Command arguments
	Response *Response `json:"response"` // Response to a message
	Respond  bool      `json:"respond"`  // Respond after receiving
	Close    bool      `json:"close"`    // Close connection after receiving

	conn      net.Conn      // Unix socket connection
	maxLength int           // Maximum size of the reading buffer (1Mb)
	timeout   time.Duration // Transaction time limit (for write/read)
}

// Options set some options on the sending/receiving
func (s *communicator) Options(maxLength int, timeout time.Duration, respond, close bool) {
	s.Respond = respond
	s.Close = close
	s.maxLength = maxLength
	s.timeout = timeout
}

// Send sends a socketMessage over the unix socket
func (s *communicator) Send() error {

	// Set timeout
	s.conn.SetDeadline(time.Now().Add(s.timeout))

	// Marshal message to JSON
	message, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("Send: could not marshal socketMessage: %s", err.Error())
	}

	// Prepare byte message
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(message)))

	byteMsg := []byte{}
	byteMsg = append(byteMsg, length...)
	byteMsg = append(byteMsg, []byte(":")...)
	byteMsg = append(byteMsg, []byte(message)...)

	// Send message
	if n, err := s.conn.Write(byteMsg); n != len(byteMsg) || err != nil {
		if err != nil {
			return fmt.Errorf("Send: failedwriting to the socket: %s", err.Error())
		}
		return fmt.Errorf("Send: sent only %d bytes (message was %d)", n, len(byteMsg))
	}

	return nil
}

// Receive reads all the data from a unix socket up to maxLength bytes.
// It expects the message to have the pattern length:message, where length
// is the length of the incoming message. It also expects the length to be
// 4 bytes long (i.e. uint32 on 64bit systems).
// Reading from the connection times out after timeout duration.
func (s *communicator) Receive() error {

	// Set timeout
	s.conn.SetDeadline(time.Now().Add(s.timeout))

	// Retrieve incoming message length
	length := make([]byte, 4)
	if n, err := s.conn.Read(length); n != 4 || err != nil {
		return fmt.Errorf("Receive: reading the length of the message failed")
	}

	// Retrieve the message
	msgLen := binary.BigEndian.Uint32(length) + 1 // Message will start with ":"
	content := make([]byte, msgLen)
	if n, err := s.conn.Read(content); uint32(n) != msgLen || (err != nil && err != io.EOF) {
		if err == nil {
			return fmt.Errorf("Receive: incorrect message length: %d (was expecting %d)", n, msgLen)
		}
		return fmt.Errorf("Receive: failed reading from unix socket: %s", err.Error())
	}

	// Unmarshal message
	newMsg := &communicator{}
	if err := json.Unmarshal(content[1:], newMsg); err != nil {
		return fmt.Errorf("Receive: cannot unmarshal response")
	}

	// Overwrite original values
	s.Cmd = newMsg.Cmd
	s.Args = newMsg.Args
	s.Response = newMsg.Response
	s.Respond = newMsg.Respond
	s.Close = newMsg.Close

	return nil
}

// GetResponse returns message's response
func (s *communicator) GetResponse() *Response {
	return s.Response
}

// SetResponse sets message's response
func (s *communicator) SetResponse(resp *Response) {
	s.Response = resp
}

// GetCmd returns message's command
func (s *communicator) GetCmd() string {
	return s.Cmd
}

// GetArgs returns message's command arguments
func (s *communicator) GetArgs() Args {
	return s.Args
}

// ShouldRespond informs the message handler that a response is expected
func (s *communicator) ShouldRespond() bool {
	return s.Respond
}

// ShouldClose informs the message handler that there's not going to be any
// further communication and the connection can be closed
func (s *communicator) ShouldClose() bool {
	return s.Close
}
