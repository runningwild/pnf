package core_test

import (
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
    bundler.Ticker = &core.FakeTicker{}
    bundler.Start()
    c.Expect(1, Equals, 1)
    bundler.Shutdown()
  })
}
