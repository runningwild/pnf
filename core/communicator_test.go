package core_test

import (
  "runningwild/pnf/core"
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
)

func CommunicatorSpec(c gospec.Context) {
  c.Specify("Communicator picks up new connections properly.", func() {
    // Set up a simple star graph, everyone connects to Communicator 0.
    var net core.NetworkMock
    hms := []core.Network{
      core.NewHostMock(&net),
      core.NewHostMock(&net),
      core.NewHostMock(&net),
      core.NewHostMock(&net),
      core.NewHostMock(&net),
    }

    var remotes []chan core.FrameBundle
    for i := range hms {
      remotes = append(remotes, make(chan core.FrameBundle))
      defer hms[i].Shutdown()
    }

    var communicators []core.Communicator
    for i := range hms {
      c := core.Communicator{
        Net:               hms[i],
        Broadcast_bundles: make(chan core.FrameBundle),
        Remote_bundles:    remotes[i],
      }
      communicators = append(communicators, c)
      c.Start()
      defer c.Shutdown()
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
      defer conn.Close()
    }

    // We will send a different FrameBundle along each connection and make
    // sure that they are all picked up by Communicator 0.
    for i := 1; i < len(hms); i++ {
      go func(n int) {
        bundle := core.FrameBundle{
          Bundle: core.EventBundle{
            core.EngineId(n): []core.Event{
              EventA{},
              EventB{},
            },
          },
          Frame: core.StateFrame(n + 10),
        }
        conns[n].SendFrameBundle(bundle)
      }(i)
    }

    bundles := make(map[core.StateFrame]core.FrameBundle)
    for i := 1; i < len(hms); i++ {
      bundle := <-remotes[0]
      bundles[bundle.Frame] = bundle
    }
    c.Expect(len(bundles), Equals, len(hms)-1)
    for i := 1; i < len(hms); i++ {
      bundle, ok := bundles[core.StateFrame(i+10)]
      c.Expect(ok, Equals, true)
      if ok {
        c.Expect(len(bundle.Bundle), Equals, 1)
        events, ok := bundle.Bundle[core.EngineId(i)]
        c.Expect(ok, Equals, true)
        if ok {
          c.Expect(len(events), Equals, 2)
          _, ok_a := events[0].(EventA)
          _, ok_b := events[1].(EventB)
          c.Expect(ok_a, Equals, true)
          c.Expect(ok_b, Equals, true)

        }
      }
    }
  })
}
