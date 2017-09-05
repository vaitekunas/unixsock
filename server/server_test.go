package server

import (
  "github.com/vaitekunas/unixsock"
  "github.com/vaitekunas/unixsock/client"
  "testing"
  "sync"
  "os"
)

func fakeHandler(cmd string, args unixsock.Args) *unixsock.Response{
  return &unixsock.Response{
    Status: unixsock.STATUS_OK,
    Error: "",
    Payload: "",
  }
}

func TestNew(t *testing.T) {

  tests := []struct{
    unixSockPath string
    isErr bool
  }{
    {os.Getenv("HOME")+"/_test_sock.sock", false},
    {"/root/_test_sock.sock", true},
  }

  for i, test := range tests {
    srv, err := New(test.unixSockPath, fakeHandler)
    if (err != nil) != test.isErr {
      if err != nil {
        t.Errorf("TestNew: test %d failed: %s",i+1,err.Error())
      }else{
        t.Errorf("TestNew: test %d failed")
      }
    }

    if test.isErr {
      continue
    }

    routines := 100
    wg := &sync.WaitGroup{}
    wg.Add(routines)

    for j:=1; j<=routines;j++ {
      go func() {
        defer wg.Done()

        client, err := client.New(test.unixSockPath)
        if err != nil {
          return
        }

        cmd := "hello.world"
      	args := map[string]interface{}{
      		"code":  1,
      		"message": "nonsense",
      	}

        resp, err := client.Send(cmd, args, true, true)
        if err != nil {
          t.Errorf("TestNew: test %d failed: could not receive response from server: %s", i+1, err.Error())
          return
        }
        if resp == nil {
          t.Errorf("TestNew: test %d failed: got nil response", i+1)
          return
        }
        if resp.Status != unixsock.STATUS_OK {
          t.Errorf("TestNew: test %d failed: expected STATUS_OK, got: %s", i+1, resp.Status)
        }
      }()
    }

    wg.Wait()

    srv.Stop()

  }

}
