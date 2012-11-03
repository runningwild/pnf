package pnf

import (
  "errors"
  "runningwild/pnf/core"
)

type Game interface {
  core.Game
}

type Event interface {
  core.Event
}

type Engine struct {
  bundler      *core.Bundler
  updater      *core.Updater
  communicator *core.Communicator
  auditor      *core.Auditor
  net          core.Network
  local_event  chan<- core.Event
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
  game, _ := e.updater.RequestFinalGameState(-1)
  return game
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
  local_bundles := make(chan core.FrameBundle)
  local_event := make(chan core.Event)
  engine.local_event = local_event
  engine.bundler.Params = params
  engine.bundler.Current_ms = params.Frame_ms
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

func makeUnstarted(params core.EngineParams, net core.Network, ticker core.Ticker) (
  chan<- core.Event, *core.Bundler, *core.Updater, *core.Communicator, *core.Auditor) {

  var bundler core.Bundler
  local_bundles := make(chan core.FrameBundle)
  local_event := make(chan core.Event)
  local_engine_event := make(chan core.EngineEvent)
  bundler.Params = params
  bundler.Local_bundles = local_bundles
  bundler.Local_event = local_event
  bundler.Local_engine_event = local_engine_event
  bundler.Ticker = ticker
  bundler.Time_delta = nil

  bootstrap_frames := make(chan core.BootstrapFrame)
  broadcast_bundles := make(chan core.FrameBundle)
  remote_bundles := make(chan core.FrameBundle)
  var updater core.Updater
  updater.Params = params
  updater.Bootstrap_frames = bootstrap_frames
  updater.Broadcast_bundles = broadcast_bundles
  updater.Local_bundles = local_bundles
  updater.Remote_bundles = remote_bundles

  var communicator core.Communicator
  raw_remote_bundles := make(chan core.FrameBundle)
  communicator.Bootstrap_frames = bootstrap_frames
  communicator.Broadcast_bundles = broadcast_bundles
  communicator.Local_engine_event = local_engine_event
  // communicator.Host_conn=
  communicator.Net = net
  communicator.Raw_remote_bundles = raw_remote_bundles

  var auditor core.Auditor
  auditor.Raw_remote_bundles = raw_remote_bundles
  auditor.Remote_bundles = remote_bundles

  return local_event, &bundler, &updater, &communicator, &auditor
}

func NewNetEngine(initial_state Game, frame_ms int64, max_frames, port int) (*Engine, error) {
  var params core.EngineParams
  params.Id = 1234
  params.Delay = core.StateFrame(frame_ms) + 10
  params.Frame_ms = frame_ms
  params.Max_frames = max_frames
  net, err := core.MakeTcpUdpNetwork(port)
  if err != nil {
    return nil, err
  }

  local_event, bundler, updater, communicator, auditor := makeUnstarted(params, net, core.NewBasicTicker())
  engine := Engine{
    bundler:      bundler,
    updater:      updater,
    communicator: communicator,
    auditor:      auditor,
    local_event:  local_event,
    net:          net,
  }
  data := core.FrameData{
    Bundle: make(core.EventBundle),
    Game:   initial_state,
    Info: core.EngineInfo{
      Engines: map[core.EngineId]bool{params.Id: true},
    },
  }

  bundler.Current_ms = frame_ms + 1
  bundler.Start()
  updater.Start(0, data)
  communicator.Start()
  auditor.Start()

  ping_func := func([]byte) ([]byte, error) {
    return []byte("I AM A HOST!!!"), nil
  }
  join_func := func([]byte) error {
    return nil
  }
  net.Host(ping_func, join_func)

  return &engine, nil
}

func NewNetClientEngine(frame_ms int64, max_frames, port int) (*Engine, error) {
  var params core.EngineParams
  params.Id = 1234
  params.Delay = core.StateFrame(frame_ms) + 10
  params.Frame_ms = frame_ms
  params.Max_frames = max_frames
  net, err := core.MakeTcpUdpNetwork(port)
  if err != nil {
    return nil, err
  }

  ticker := core.NewBasicTicker()
  local_event, bundler, updater, communicator, auditor := makeUnstarted(params, net, ticker)
  engine := Engine{
    bundler:      bundler,
    updater:      updater,
    communicator: communicator,
    auditor:      auditor,
    local_event:  local_event,
    net:          net,
  }
  // ticker.Start()

  rhs, err := net.Ping([]byte("ASDFADSFADSF"))
  if err != nil {
    return nil, err
  }
  if len(rhs) == 0 {
    return nil, errors.New("Didn't find any remote hosts.")
  }
  conn, err := net.Join(rhs[0], []byte("I am joinng"))
  if err != nil {
    return nil, err
  }

  boot, id, err := communicator.Join(conn)
  if err != nil {
    return nil, err
  }

  bundler.Params.Id = id
  updater.Params.Id = id
  bundler.Current_ms = params.Frame_ms * (int64(boot.Frame))
  bundler.Start()
  updater.Bootstrap(boot)
  communicator.Start()
  auditor.Start()

  return &engine, nil
}
