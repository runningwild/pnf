package core_test

import (
  "fmt"
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
  "runningwild/pnf/core"
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
        Engines: map[core.EngineId]bool{
          params.Id:     true,
          params.Id + 1: true,
        },
      },
    }
    var start_frame core.StateFrame = 10
    updater.Start(start_frame, data)
    go func() {
      for _ = range broadcast_bundles {
      }
    }()
    defer close(broadcast_bundles)
    cur_frame := start_frame + 1
    c.Specify("Local game Events are applied properly.", func() {
      for cur_frame = start_frame + 1; cur_frame <= start_frame+5; cur_frame++ {
        local_bundles <- core.FrameBundle{
          Frame: cur_frame,
          Bundle: core.EventBundle{
            params.Id: core.AllEvents{
              Game: []core.Event{
                EventA{2},
                EventB{fmt.Sprintf("%d", cur_frame)},
              },
            },
            params.Id + 1: core.AllEvents{
              Game: []core.Event{
                EventA{1},
                EventB{fmt.Sprintf("%d", cur_frame)},
              },
            },
          },
        }
      }
      state, _ := updater.RequestFinalGameState(cur_frame - 1)
      c.Expect(state, Not(Equals), nil)
      tg := state.(*TestGame)
      c.Expect(tg.Thinks, Equals, 5)
      c.Expect(tg.A, Equals, 15)
      c.Expect(tg.B, Equals, fmt.Sprintf("%d", cur_frame-1))
    })

    // Same test as above, but one of the engine's events come through the
    // remote_bundles channel.
    c.Specify("Remote game Events are applied properly.", func() {
      for cur_frame = start_frame + 1; cur_frame <= start_frame+5; cur_frame++ {
        local_bundles <- core.FrameBundle{
          Frame: cur_frame,
          Bundle: core.EventBundle{
            params.Id: core.AllEvents{
              Game: []core.Event{
                EventA{2},
                EventB{fmt.Sprintf("%d", cur_frame)},
              },
            },
          },
        }
        remote_bundles <- core.FrameBundle{
          Frame: cur_frame,
          Bundle: core.EventBundle{
            params.Id + 1: core.AllEvents{
              Game: []core.Event{
                EventA{1},
                EventB{fmt.Sprintf("%d", cur_frame)},
              },
            },
          },
        }
      }
      state, _ := updater.RequestFinalGameState(cur_frame - 1)
      c.Expect(state, Not(Equals), nil)
      tg := state.(*TestGame)
      c.Expect(tg.Thinks, Equals, 5)
      c.Expect(tg.A, Equals, 15)
      c.Expect(tg.B, Equals, fmt.Sprintf("%d", cur_frame-1))
    })

    // Similar test as above but we drop and rejoin one of the engines.
    c.Specify("Engine Events are applied properly.", func() {
      local_bundles <- core.FrameBundle{
        Frame: cur_frame,
        Bundle: core.EventBundle{
          params.Id: core.AllEvents{
            Engine: []core.EngineEvent{
              core.EngineDropped{params.Id + 1},
            },
            Game: []core.Event{
              EventA{1},
            },
          },
          params.Id + 1: core.AllEvents{
            Game: []core.Event{
              EventA{1},
            },
          },
        },
      }

      local_bundles <- core.FrameBundle{
        Frame: cur_frame + 1,
        Bundle: core.EventBundle{
          params.Id: core.AllEvents{
            Engine: []core.EngineEvent{
              core.EngineJoined{params.Id + 1},
            },
            Game: []core.Event{
              EventA{1},
            },
          },
          params.Id + 1: core.AllEvents{ // These events should not get applied
            Game: []core.Event{
              EventA{1},
            },
          },
        },
      }

      local_bundles <- core.FrameBundle{
        Frame: cur_frame + 2,
        Bundle: core.EventBundle{
          params.Id: core.AllEvents{
            Game: []core.Event{
              EventA{1},
            },
          },
          params.Id + 1: core.AllEvents{ // These events should get applied
            Game: []core.Event{
              EventA{1},
            },
          },
        },
      }

      state, _ := updater.RequestFinalGameState(-1)
      c.Expect(state, Not(Equals), nil)
      tg := state.(*TestGame)
      c.Expect(tg.Thinks, Equals, 3)
      c.Expect(tg.A, Equals, 5)
    })
  })
}
