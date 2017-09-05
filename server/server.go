package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/vaitekunas/unixsock"
	context "golang.org/x/net/context"
)

// UnixSockSrv is a unix-socket server interface
type UnixSockSrv interface {
	// Stop stops the server and all supporting goroutines
	Stop()
}

// New starts a unix-socket server listening on UnixSockPath
func New(UnixSockPath string, handler func(cmd string, args unixsock.Args) *unixsock.Response) (UnixSockSrv, error) {

	// Internal context
	internalCTX, cancel := context.WithCancel(context.Background())

	// Listen on to the unix socket
	listenUnix, err := net.Listen("unix", UnixSockPath)
	if err != nil {
		return nil, fmt.Errorf("New: could not listen on the unix socket: %s", err.Error())
	}

	// New instance of unixSockSrv
	srv := &unixSockSrv{
		listenUnix: listenUnix,
		cancelCTX:  cancel,
	}

	// Unix handler
	unixHandler := newUnixRequestHandler(handler)

	// Serve socket requests
	connChan := make(chan net.Conn, 1)

	// Wait group for goroutine startup
	wg := &sync.WaitGroup{}
	wg.Add(2)

	// Listen for incoming unix connections
	go func() {
		wg.Done()
	Loop:
		for {
			fd, errUnix := listenUnix.Accept()
			if errUnix != nil {
				continue
			}
			select {
			case connChan <- fd:
			case <-internalCTX.Done():
				break Loop
			}
		}
	}()

	// Process unix connections
	go func() {
		wg.Done()
	Loop:
		for {
			select {
			case conn := <-connChan:
				go unixHandler(conn)
			case <-internalCTX.Done():
				break Loop
			}
		}
	}()

	// Wait for goroutines
	wg.Wait()

	// New server
	return srv, nil

}

// unixSockSrv implements the UnixSockSrv interface
type unixSockSrv struct {
	listenUnix net.Listener
	cancelCTX  func()
}

// Stop stops the server and all supporting goroutines
func (u *unixSockSrv) Stop() {
	u.cancelCTX()
	u.listenUnix.Close()
}

// newUnixRequestHandler creates a new unix request handler using executor to
// execute incoming commands. The created function handles a request via a
// unix socket connection. It expects to read only a single message and respond
// to it immediately
func newUnixRequestHandler(handler func(cmd string, args unixsock.Args) *unixsock.Response) func(net.Conn) {
	return func(c net.Conn) {
		defer c.Close()

	Loop:
		for {

			// Receive the command
			receiver := unixsock.NewReceiver(c)
			if err := receiver.Receive(); err != nil {
				break Loop
			}

			// Handle the command
			response := handler(receiver.GetCmd(), receiver.GetArgs())

			// Respond
			if receiver.ShouldRespond() {
				receiver.SetResponse(response)
				receiver.Send()
			}

			// Close connection
			if receiver.ShouldClose() {
				break Loop
			}

		}

	}
}
