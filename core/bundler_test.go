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
    events := make(chan []core.Event)
    local_event := make(chan core.Event)
    var bundler core.Bundler
    bundler.Params = &params
    bundler.Current_ms = 0
    bundler.Completed_frame = completed_frame
    bundler.Events = events
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
    frame := 0
    for bundles := range events {
      if frame%2 == 0 {
        c.Assume(len(bundles), Equals, 2)
        c.Specify("checking bundle values", func() {
          index_a := 0
          if frame%4 != 0 {
            index_a = 1
          }
          ea, aok := bundles[index_a].(*EventA)
          eb, bok := bundles[(index_a+1)%2].(*EventB)
          c.Assume(aok, Equals, true)
          c.Assume(bok, Equals, true)
          c.Specify("checking event data", func() {
            c.Expect(ea.Data, Equals, frame/2)
            c.Expect(eb.Data, Equals, fmt.Sprintf("%d", frame/2))
          })
        })
      } else {
        c.Expect(len(bundles), Equals, 0)
      }
      frame++
    }
  })
}
