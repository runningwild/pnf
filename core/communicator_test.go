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
    var hms []core.Network
    for i := 0; i < 15; i++ {
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
      defer conn.Close()
    }

    var communicators []*core.Communicator
    for i := range hms {
      c := core.Communicator{
        Net:                hms[i],
        Broadcast_bundles:  make(chan core.FrameBundle),
        Raw_remote_bundles: remotes[i],
        Host_conn:          conns[i],
      }
      communicators = append(communicators, &c)
      c.Start()
      defer c.Shutdown()
    }

    // We will send a different FrameBundle from each non-host communicator
    // and make sure all of those bundles are picked up by all other
    // communicators.
    for i := 1; i < len(hms); i++ {
      go func(n int) {
        bundle := core.FrameBundle{
          Bundle: core.EventBundle{
            core.EngineId(n): core.AllEvents{
              Game: []core.Event{
                EventA{},
                EventB{},
              },
            },
          },
          Frame: core.StateFrame(n + 10),
        }
        conns[n].SendFrameBundle(bundle)
      }(i)
    }

    // First we check that the host got all of the expected bundles
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
          c.Expect(len(events.Game), Equals, 2)
          if len(events.Game) == 2 {
            _, ok_a := events.Game[0].(EventA)
            _, ok_b := events.Game[1].(EventB)
            c.Expect(ok_a, Equals, true)
            c.Expect(ok_b, Equals, true)
          }
        }
      }
    }

    // Now we go through each client and make sure each one has received the
    // bundles sent from all of the other clients.
    for dst := 1; dst < len(hms); dst++ {
      bundles := make(map[core.StateFrame]core.FrameBundle)
      for i := 2; i < len(hms); i++ {
        bundle := <-remotes[dst]
        bundles[bundle.Frame] = bundle
      }
      c.Expect(len(bundles), Equals, len(hms)-2)
      for i := 1; i < len(hms); i++ {
        if i == dst {
          continue
        }
        bundle, ok := bundles[core.StateFrame(i+10)]
        c.Expect(ok, Equals, true)
        if ok {
          c.Expect(len(bundle.Bundle), Equals, 1)
          events, ok := bundle.Bundle[core.EngineId(i)]
          c.Expect(ok, Equals, true)
          if ok {
            c.Expect(len(events.Game), Equals, 2)
            if len(events.Game) == 2 {
              _, ok_a := events.Game[0].(EventA)
              _, ok_b := events.Game[1].(EventB)
              c.Expect(ok_a, Equals, true)
              c.Expect(ok_b, Equals, true)
            }
          }
        }
      }
    }
  })
}
