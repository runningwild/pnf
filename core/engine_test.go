package core_test

import (
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
  "github.com/runningwild/core"
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
func EngineUdpTcpSpec(c gospec.Context) {
  c.Specify("Standard network can drive engines properly.", func() {
    host_ticker := core.FakeTicker{}
    client_ticker := core.FakeTicker{}

    var params core.EngineParams
    params.Id = 1234
    params.Delay = 1
    params.Frame_ms = 17
    params.Max_frames = 50

    ping_func := func(data []byte) ([]byte, error) {
      println("ping with ", string(data))
      return []byte{}, nil
    }
    join_func := func(data []byte) error {
      println("Join on", string(data))
      return nil
    }

    port := int(core.RandomId()%10000 + 1000)
    host_net, err := core.MakeTcpUdpNetwork(port)
    c.Expect(err, Equals, error(nil))
    host_net.Host(ping_func, join_func)
    local_event, bundler, host_updater, communicator, auditor := makeUnstarted(params, host_net, &host_ticker)
    data := core.FrameData{
      Bundle: make(core.EventBundle),
      Game:   &TestGame{},
      Info: core.EngineInfo{
        Engines: map[core.EngineId]bool{params.Id: true},
      },
    }

    bundler.Current_ms = 20
    bundler.Start()
    host_updater.Start(0, data)
    communicator.Start()
    auditor.Start()
    local_event <- EventA{3}
    for i := 0; i < 10; i++ {
      host_ticker.Inc(int(params.Frame_ms))
      gs, _ := host_updater.RequestFinalGameState(-1)
      c.Expect(gs.(*TestGame).Thinks, Equals, i+1)
      if gs.(*TestGame).Thinks != i+1 {
        return
      }
    }
    time.Sleep(time.Millisecond * 10)
    client_net, err := core.MakeTcpUdpNetwork(port)
    c.Expect(err, Equals, error(nil))
    rhs, _ := client_net.Ping([]byte{})
    c.Expect(len(rhs), Equals, 1)
    if len(rhs) != 1 {
      return
    }
    conn, err := client_net.Join(rhs[0], []byte("asdfasdf"))
    c.Expect(err, Equals, error(nil))
    if err != nil {
      return
    }

    // Client-land
    {
      client_ticker.Start()
      params.Id = 2345
      local_event, bundler, client_updater, communicator, auditor := makeUnstarted(params, client_net, &client_ticker)
      done := make(chan bool)
      var boot *core.BootstrapFrame
      var id core.EngineId
      go func() {
        boot, id, err = communicator.Join(conn)
        done <- true
      }()
      time.Sleep(time.Millisecond * 10)
      count := 0
      for {
        count++
        select {
        case <-done:
          goto joined
        default:
          host_ticker.Inc(10)
          time.Sleep(time.Millisecond)
        }
      }

    joined:
      println("Host id: ", host_updater.Params.Id)
      println("Client id: ", client_updater.Params.Id)

      bundler.Params.Id = id
      client_updater.Params.Id = id
      c.Expect(err, Equals, error(nil))
      if err != nil {
        return
      }
      bundler.Current_ms = params.Frame_ms * (int64(boot.Frame))
      // bundler.Current_ms = 120
      bundler.Start()
      client_updater.Bootstrap(boot)
      communicator.Start()
      auditor.Start()

      // Run the engines until they've actually connected
      for client_updater.NumEngines() < 2 && host_updater.NumEngines() < 2 {
        client_ticker.Inc(int(params.Frame_ms))
        host_ticker.Inc(int(params.Frame_ms))
      }

      // Now make sure that the two engines are in sync.  Ideally this will
      // happen automatically, but for now, and for the sake of testing, 
      // we're doing it manually.
      _, current_client_frame := client_updater.RequestFinalGameState(-1)
      _, current_host_frame := host_updater.RequestFinalGameState(-1)
      if current_client_frame < current_host_frame {
        go client_ticker.Inc(int(current_host_frame-current_client_frame)*int(params.Frame_ms) + 2)
        current_client_frame = current_host_frame
        client_updater.RequestFinalGameState(current_client_frame)
      }
      for current_host_frame < current_client_frame {
        go host_ticker.Inc(int(current_client_frame-current_host_frame)*int(params.Frame_ms) + 2)
        current_host_frame = current_client_frame
        // TODO: Can still deadlock on the following line sometimes
        host_updater.RequestFinalGameState(current_host_frame)
      }

      target := 100
      client_a := -1
      host_a := -2
      max := 0
      for i := 0; i < 2000; i++ {
        current_client_frame++
        current_host_frame++
        go host_ticker.Inc(17)
        go client_ticker.Inc(17)

        local_event <- EventA{3}
        _gsc, _ := client_updater.RequestFinalGameState(current_client_frame)
        gsc := _gsc.(*TestGame)
        if gsc.Thinks == target {
          client_a = gsc.A
        }
        max = gsc.Thinks
        _gsh, _ := host_updater.RequestFinalGameState(current_host_frame)
        gsh := _gsh.(*TestGame)
        if gsh.Thinks == target {
          host_a = gsh.A
        }
        if client_a > 0 && host_a > 0 {
          break
        }
      }
      c.Expect(max, Equals, target)
      c.Expect(client_a, Not(Equals), -1)
      c.Expect(client_a, Equals, host_a)
    }
  })
}

// NEXT: Works with a normal ticker, get it working with a fake ticker as well.
// NEXT: Also double check that clients don't start on the wrong frame.
func EngineSpec(c gospec.Context) {
  c.Specify("Communicator picks up new connections properly.", func() {
    host_ticker := core.FakeTicker{}
    client_ticker := core.FakeTicker{}

    var params core.EngineParams
    params.Id = 1234
    params.Delay = 1
    params.Frame_ms = 17
    params.Max_frames = 50

    var net core.NetworkMock
    ping_func := func([]byte) ([]byte, error) {
      return []byte{}, nil
    }
    join_func := func([]byte) error {
      return nil
    }
    host_net := core.NewHostMock(&net)
    host_net.Host(ping_func, join_func)
    local_event, bundler, host_updater, communicator, auditor := makeUnstarted(params, host_net, &host_ticker)
    data := core.FrameData{
      Bundle: make(core.EventBundle),
      Game:   &TestGame{},
      Info: core.EngineInfo{
        Engines: map[core.EngineId]bool{params.Id: true},
      },
    }

    bundler.Current_ms = 20
    bundler.Start()
    host_updater.Start(0, data)
    communicator.Start()
    auditor.Start()
    local_event <- EventA{3}
    for i := 0; i < 10; i++ {
      host_ticker.Inc(int(params.Frame_ms))
      gs, _ := host_updater.RequestFinalGameState(-1)
      c.Expect(gs.(*TestGame).Thinks, Equals, i+1)
      if gs.(*TestGame).Thinks != i+1 {
        return
      }
    }

    client_net := core.NewHostMock(&net)
    rhs, _ := client_net.Ping([]byte{})
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
      client_ticker.Start()
      local_event, bundler, client_updater, communicator, auditor := makeUnstarted(params, client_net, &client_ticker)
      done := make(chan bool)
      var boot *core.BootstrapFrame
      var id core.EngineId
      go func() {
        boot, id, err = communicator.Join(conn)
        done <- true
      }()
      for {
        select {
        case <-done:
          goto joined
        default:
          host_ticker.Inc(1)
        }
      }
    joined:

      bundler.Params.Id = id
      client_updater.Params.Id = id
      c.Expect(err, Equals, error(nil))
      if err != nil {
        return
      }
      bundler.Current_ms = params.Frame_ms * (int64(boot.Frame))
      // bundler.Current_ms = 120
      bundler.Start()
      client_updater.Bootstrap(boot)
      communicator.Start()
      auditor.Start()

      // Run the engines until they've actually connected
      for client_updater.NumEngines() < 2 && host_updater.NumEngines() < 2 {
        client_ticker.Inc(int(params.Frame_ms))
        host_ticker.Inc(int(params.Frame_ms))
        net.Purge()
      }

      // Now make sure that the two engines are in sync.  Ideally this will
      // happen automatically, but for now, and for the sake of testing, 
      // we're doing it manually.
      _, current_client_frame := client_updater.RequestFinalGameState(-1)
      _, current_host_frame := host_updater.RequestFinalGameState(-1)
      if current_client_frame < current_host_frame {
        go client_ticker.Inc(int(current_host_frame-current_client_frame)*int(params.Frame_ms) + 2)
        current_client_frame = current_host_frame
        client_updater.RequestFinalGameState(current_client_frame)
      }
      for current_host_frame < current_client_frame {
        go host_ticker.Inc(int(current_client_frame-current_host_frame)*int(params.Frame_ms) + 2)
        current_host_frame = current_client_frame
        // TODO: Can still deadlock on the following line sometimes
        host_updater.RequestFinalGameState(current_host_frame)
      }

      target := 100
      client_a := -1
      host_a := -2
      max := 0
      for i := 0; i < 2000; i++ {
        current_client_frame++
        current_host_frame++
        go host_ticker.Inc(17)
        go client_ticker.Inc(17)

        local_event <- EventA{3}
        _gsc, _ := client_updater.RequestFinalGameState(current_client_frame)
        gsc := _gsc.(*TestGame)
        if gsc.Thinks == target {
          client_a = gsc.A
        }
        max = gsc.Thinks
        _gsh, _ := host_updater.RequestFinalGameState(current_host_frame)
        gsh := _gsh.(*TestGame)
        if gsh.Thinks == target {
          host_a = gsh.A
        }
        if client_a > 0 && host_a > 0 {
          break
        }
      }
      c.Expect(max, Equals, target)
      c.Expect(client_a, Not(Equals), -1)
      c.Expect(client_a, Equals, host_a)
    }
  })
}
