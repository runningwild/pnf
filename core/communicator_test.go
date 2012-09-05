package core_test

import (
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
  "runningwild/pnf/core"
)

func CommunicatorSpec(c gospec.Context) {
  c.Specify("Communicator picks up new connections properly.", func() {
    // Set up a simple star graph, everyone connects to Communicator 0.
    var net core.NetworkMock
    var hms []core.Network
    for i := 0; i < 2; i++ {
      hms = append(hms, core.NewHostMock(&net))
    }

    var remotes []chan core.FrameBundle
    for i := range hms {
      remotes = append(remotes, make(chan core.FrameBundle))
      defer hms[i].Shutdown()
    }

    ping_func := func([]byte) ([]byte, error) {
      return []byte{}, nil
    }
    join_func := func([]byte) error {
      return nil
    }

    // conns[i] is the connection between hms[i] and hms[0]
    var conns []core.Conn
    conns = append(conns, nil)

    hms[0].Host(ping_func, join_func)
    for i := 1; i < len(hms); i++ {
      rhs := hms[i].Ping([]byte{})
      c.Expect(len(rhs), Equals, 1)
      if len(rhs) != 1 {
        return
      }
      conn, err := hms[i].Join(rhs[0], []byte{})
      c.Expect(err, Equals, error(nil))
      if err != nil {
        return
      }
      conns = append(conns, conn)
    }
    return
    // NEXT: Fill in appropriate fields, and then try to bootstrap something
  })
}
