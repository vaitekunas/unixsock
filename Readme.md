# unixsock [![godoc](https://img.shields.io/badge/go-documentation-blue.svg)](https://godoc.org/github.com/vaitekunas/unixsock) [![Build Status](https://travis-ci.org/vaitekunas/unixsock.svg?branch=master)](https://travis-ci.org/vaitekunas/unixsock) [![Coverage Status](https://coveralls.io/repos/github/vaitekunas/unixsock/badge.svg?branch=master)](https://coveralls.io/github/vaitekunas/unixsock?branch=master)

UNIX domain sockets are a method by which processes on the same host can
communicate with each other. The communication is bidirectional, so that both the
client and the server can send and receive messages.

One special case, which `unixsock` has been written to address, is to enable
communication with a running process (e.g. a web server), so that it could be
reconfigured without downtime and monitored without additional tools (e.g.
retrieving real-time statistics).

Some of the advantages of using UNIX domain sockets instead of alternatives (
reconfiguration via environment variables, or web-based UI):

* Accessible only on the host by a user having permission to read the socket file.
* No need to implement additional authentication methods - authentication can be
handled by the ssh agent
* No need to build a UI (although a UI could be more comfortable in some cases).
Building a sysadmin UI usually requires additional security measures, user isolation
and so on. In case of UNIX sockets all this is handled by the host.

The distributed logging facility [journald](https://github.com/vaitekunas/journal)
is an example of using the UNIX domain socket to configure and monitor a running
application.

## Server

A server's core is the request handler.

```Go
func handler(cmd string, args unixsock.Args) *unixsock.Response{

  switch strings.ToLower(cmd) {
  case "echo":
    return &unixsock.Response{
      Status: unixsock.STATUS_OK,
      Payload: cmd
    }
  default:
    return &unixsock.Response{
      Status: unixsock.STATUS_FAIL,
      Error: fmt.Errorf("handler: unknown command '%s'", cmd),      
    }
  }  
}
```

having written a request handler, we can start the server. If the `UnixSockSrv`
is used for configuration and monitoring, then it will usually run in its own
routine until the main application exits, e.g.:

```Go
unixSockPath := fmt.Sprintf("%s/server.sock", os.Getenv("HOME"))

quitChan := make(chan bool, 1)

go func() {
  srv, err := server.New(unixSockPath, handler)
  if err != nil {
    log.Fatal(err.Error())
  }
  <- quitChan
  srv.Stop()
}
...
```

## Client

The client must know the path to the socket file as well as the API that the
server is handling. Other than that, the communication with an instance of
`UnixSockSrv` is straightforward:

```Go
unixSockPath := fmt.Sprintf("%s/server.sock", os.Getenv("HOME"))

client, err := client.New(unixSockPath)
if err != nil {
  log.Fatal(err.Error())
}

cmd := "echo"
args := map[string]interface{}{}

resp, err := client.Send(cmd, args, true, true)
if err != nil {
  log.Fatal(err.Error())
}
```
