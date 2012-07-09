package pnf

import (
  "runningwild/pnf/core"
)

type Game interface {
  core.Game
}

type Event interface {
  core.Event
}

type Engine struct {
  bundler     core.Bundler
  updater     core.Updater
  local_event chan<- core.Event
}

type RemoteHost struct{}

func (e *Engine) Host(ping, join func([]byte) []byte) {}
func (e *Engine) FindHosts(data []byte) []RemoteHost {
  return nil
}
func (e *Engine) JoinHost(data []byte) {}

func (e *Engine) Start(game Game) {}

func NewEngine(params string) *Engine {
  // var n core.Network
  return nil
}
func (e *Engine) GetState() Game {
  return e.updater.RequestFinalGameState()
}
func (e *Engine) ApplyEvent(event Event) {
  e.local_event <- event
}
func NewLocalEngine(initial_state Game, frame_ms int64) *Engine {
  var engine Engine
  var params core.EngineParams
  params.Id = 1234
  params.Delay = 1
  params.Frame_ms = frame_ms
  params.Max_frames = 2
  completed_frame := make(chan core.StateFrame)
  local_bundles := make(chan core.FrameBundle)
  local_event := make(chan core.Event)
  engine.local_event = local_event
  engine.bundler.Params = params
  engine.bundler.Current_ms = params.Frame_ms
  engine.bundler.Completed_frame = completed_frame
  engine.bundler.Local_bundles = local_bundles
  engine.bundler.Local_event = local_event
  engine.bundler.Time_delta = nil
  engine.bundler.Ticker = core.NewBasicTicker()
  engine.bundler.Start()

  engine.updater.Params = params
  broadcast_bundles := make(chan core.FrameBundle)
  engine.updater.Local_bundles = local_bundles
  engine.updater.Broadcast_bundles = broadcast_bundles
  engine.updater.Remote_bundles = nil
  data := core.FrameData{
    Bundle: nil,
    Game:   initial_state,
    Info: core.EngineInfo{
      Engines: map[core.EngineId]bool{params.Id: true},
    },
  }
  var start_frame core.StateFrame = 0
  engine.updater.Start(start_frame, data)
  go func() {
    for _ = range broadcast_bundles {
    }
  }()

  return &engine
}
