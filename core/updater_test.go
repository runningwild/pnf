package core_test

import (
  "fmt"
  "runningwild/pnf/core"
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
)

func UpdaterSpec(c gospec.Context) {
  c.Specify("Basic Updater functionality.", func() {
    var params core.EngineParams
    params.Id = 1234
    params.Delay = 2
    params.Frame_ms = 5
    params.Max_frames = 25
    var updater core.Updater
    updater.Params = params
    local_bundles := make(chan core.FrameBundle)
    broadcast_bundles := make(chan core.FrameBundle)
    remote_bundles := make(chan core.FrameBundle)
    updater.Local_bundles = local_bundles
    updater.Broadcast_bundles = broadcast_bundles
    updater.Remote_bundles = remote_bundles
    data := core.FrameData{
      Bundle: nil,
      Game:   &TestGame{},
      Info: core.EngineInfo{
        Engines: map[core.EngineId]bool{params.Id: true},
      },
    }
    var start_frame core.StateFrame = 10
    updater.Start(start_frame, data)
    go func() {
      for _ = range broadcast_bundles {
      }
    }()
    var cur_frame core.StateFrame
    for cur_frame = start_frame + 1; cur_frame <= start_frame+5; cur_frame++ {
      local_bundles <- core.FrameBundle{
        Frame: cur_frame,
        Bundle: core.EventBundle{
          params.Id: []core.Event{
            EventA{int(cur_frame)},
            EventB{fmt.Sprintf("%d", cur_frame)},
          },
        },
      }
    }
    state := updater.RequestFinalGameState()
    c.Expect(state, Not(Equals), nil)
    tg := state.(*TestGame)
    c.Expect(tg.Thinks, Equals, 5)
    c.Expect(tg.B, Equals, fmt.Sprintf("%d", cur_frame-1))
  })
}
