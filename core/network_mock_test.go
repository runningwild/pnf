package core_test

import (
  "fmt"
  "runningwild/pnf/core"
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
  "errors"
  "sync"
)

var network_mutex sync.Mutex

func NetworkMockSpec(c gospec.Context) {
  c.Specify("NetworkMocks can connect to eachother and send Events.", func() {
    var net core.NetworkMock
    hm1 := core.NewHostMock(&net)
    hm2 := core.NewHostMock(&net)
    c.Expect(hm1, Not(Equals), nil)
    c.Expect(hm2, Not(Equals), nil)
    ping_func := func(data []byte) ([]byte, error) {
      return []byte(fmt.Sprintf("Ping: %s", data)), nil
    }
    join_func := func(data []byte) error {
      if string(data) == "password" {
        return nil
      }
      return errors.New("fail!")
    }
    hm1.Host(ping_func, join_func)
    rhs := hm2.Ping([]byte("MONKEY"))
    c.Expect(len(rhs), Equals, 1)
    conn, err := hm2.Join(rhs[0], []byte("I am the monkey"))
    c.Expect(conn, Equals, nil)
    c.Expect(err, Not(Equals), nil)
    conn, err = hm2.Join(rhs[0], []byte("password"))
    c.Expect(conn, Not(Equals), nil)
    c.Expect(err, Equals, error(nil))

    // We've connected, so hm1 should be able to find a new connection.
    conn2 := <-hm1.NewConns()
    c.Expect(conn2, Not(Equals), nil)
    fb := core.FrameBundle{}
    fb.Frame = 10
    fb.Bundle = core.EventBundle{
      1: []core.Event{EventA{}},
      2: nil,
    }
    go func() {
      conn.SendFrameBundle(fb)
    }()
    fb2 := <-conn2.RecvFrameBundle()
    c.Expect(fb2.Frame, Equals, fb.Frame)
    c.Expect(len(fb2.Bundle), Equals, len(fb.Bundle))
    if len(fb2.Bundle) == len(fb.Bundle) {
      c.Expect(len(fb2.Bundle[1]), Equals, 1)
      if len(fb2.Bundle[1]) == 1 {
        _, ok := fb2.Bundle[1][0].(EventA)
        c.Expect(ok, Equals, true)
        _, ok = fb2.Bundle[1][0].(EventB)
        c.Expect(ok, Equals, false)
      }
      c.Expect(len(fb2.Bundle[0]), Equals, 0)
    }
  })
}
