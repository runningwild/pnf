package core_test

import (
  "fmt"
  "runningwild/pnf/core"
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
)

func BundlerSpec(c gospec.Context) {
  c.Specify("Basic Bundler functionality.", func() {
    var params core.EngineParams
    params.Id = 1234
    params.Delay_ms = 15
    params.Frame_ms = 5
    params.Max_frames = 25
    completed_frame := make(chan core.StateFrame)
    bundles := make(chan core.FrameBundle)
    local_event := make(chan core.Event)
    var bundler core.Bundler
    bundler.Params = params
    bundler.Current_ms = 0
    bundler.Completed_frame = completed_frame
    bundler.Local_bundles = bundles
    bundler.Local_event = local_event
    bundler.Time_delta = nil
    ticker := &core.FakeTicker{}
    bundler.Ticker = ticker
    bundler.Start()
    c.Expect(1, Equals, 1)
    go func() {
      for i := 0; i < 10; i++ {
        if i%2 == 0 {
          local_event <- &EventA{i}
          local_event <- &EventB{fmt.Sprintf("%d", i)}
        } else {
          local_event <- &EventB{fmt.Sprintf("%d", i)}
          local_event <- &EventA{i}
        }
        // Advance two frames, so we will have two events per every other
        // frame.
        ticker.Inc(10)
      }
      bundler.Shutdown()
    }()
    var frame core.StateFrame = 0
    for bundle := range bundles {
      c.Expect(bundle.Frame, Equals, frame)
      events, ok := bundle.Bundle[params.Id]
      c.Assume(ok, Equals, true)
      if ok {
        if frame%2 == 0 {
          c.Expect(len(events), Equals, 2)
        } else {
          c.Expect(len(events), Equals, 0)
        }
      }
      if frame%2 == 0 {
        c.Expect(len(bundle.Bundle), Equals, 1)
        c.Specify("checking bundle values", func() {
          index_a := 0
          if frame%4 != 0 {
            index_a = 1
          }
          ea, aok := events[index_a].(*EventA)
          eb, bok := events[(index_a+1)%2].(*EventB)
          c.Assume(aok, Equals, true)
          c.Assume(bok, Equals, true)
          c.Specify("checking event data", func() {
            c.Expect(ea.Data, Equals, int(frame/2))
            c.Expect(eb.Data, Equals, fmt.Sprintf("%d", frame/2))
          })
        })
      }
      frame++
    }
  })
}
