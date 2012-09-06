package core_test

import (
  "fmt"
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
  "runningwild/pnf/core"
  "time"
)

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

// NEXT: Works with a normal ticker, get it working with a fake ticker as well.
// NEXT: Also double check that clients don't start on the wrong frame.
func EngineSpec(c gospec.Context) {
  c.Specify("Communicator picks up new connections properly.", func() {
    go func() {
      time.Sleep(2 * time.Second)
      fmt.Printf("")
      panic("TIMES UP!")
    }()

    host_ticker := core.FakeTicker{}
    client_ticker := core.FakeTicker{}

    var params core.EngineParams
    params.Id = 1234
    params.Delay = 1
    params.Frame_ms = 17
    params.Max_frames = 5

    var net core.NetworkMock
    ping_func := func([]byte) ([]byte, error) {
      return []byte{}, nil
    }
    join_func := func([]byte) error {
      return nil
    }
    host_net := core.NewHostMock(&net)
    host_net.Host(ping_func, join_func)
    local_event, bundler, updater, communicator, auditor := makeUnstarted(params, host_net, &host_ticker)
    data := core.FrameData{
      Bundle: make(core.EventBundle),
      Game:   &TestGame{},
      Info: core.EngineInfo{
        Engines: map[core.EngineId]bool{params.Id: true},
      },
    }

    bundler.Current_ms = 20
    bundler.Start()
    updater.Start(0, data)
    communicator.Start()
    auditor.Start()
    local_event <- EventA{3}
    for i := 0; i < 10; i++ {
      host_ticker.Inc(int(params.Frame_ms))
      gs := updater.RequestFinalGameState()
      c.Expect(gs.(*TestGame).Thinks, Equals, i+1)
      if gs.(*TestGame).Thinks != i+1 {
        return
      }
    }

    client_net := core.NewHostMock(&net)
    rhs := client_net.Ping([]byte{})
    c.Expect(len(rhs), Equals, 1)
    if len(rhs) != 1 {
      return
    }
    conn, err := client_net.Join(rhs[0], []byte{})
    c.Expect(err, Equals, error(nil))
    if err != nil {
      return
    }
    // Client-land
    {
      local_event, bundler, updater, communicator, auditor := makeUnstarted(params, client_net, &client_ticker)
      boot, id, err := communicator.Join(conn)
      bundler.Params.Id = id
      updater.Params.Id = id
      println("Boot: ", boot.Frame, id)
      c.Expect(err, Equals, error(nil))
      if err != nil {
        return
      }
      bundler.Current_ms = params.Frame_ms * (int64(boot.Frame) + 1)
      // bundler.Current_ms = 120
      bundler.Start()
      updater.Bootstrap(boot)
      communicator.Start()
      auditor.Start()
      for i := 0; i < 20; i++ {
        host_ticker.Inc(int(params.Frame_ms))
        client_ticker.Inc(int(params.Frame_ms))
        local_event <- EventA{3}
        gs := updater.RequestFinalGameState().(*TestGame)
        println("client: ", gs.Thinks, " ", gs.A, " ", gs.B)
      }
    }
    panic("Df")
  })
}
