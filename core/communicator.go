package core

import (
  // "encoding/binary"
  "sync"
)

type RemoteFrameBundle struct {
  bundle FrameBundle
  conn   Conn
}

type bootstrap struct {
  conn Conn

  // The frame for which this conn should start its engine, i.e. the first
  // frame for which we sent this conn a completed frame.
  start StateFrame
}

// The Communicator has the following tasks:
// - It sends all local FrameBundles to all remote hosts.
// - It collects all remote FrameBundles and sends them to tha auditor.
// - It accepts new connections and bootstraps them into the game.
type Communicator struct {
  Net Network

  // Bundles from the Updater come through here and get broadcast to all
  // remote hosts.
  Broadcast_bundles <-chan FrameBundle

  // Remote bundles are eventually sent to the auditor through here.
  Raw_remote_bundles chan<- FrameBundle

  // When all of the data for a frame has been received it is sent here from
  // the Updater.  An engine that does not want to host can safely leave this
  // as nil.
  Bootstrap_frames <-chan BootstrapFrame

  // Used to send EngineEvents to the Bundler.
  Local_engine_event chan<- EngineEvent

  // This is necessary for starting up a client engine.  A host can safely
  // leave this as nil.
  host_conn Conn

  // Bundles from remote hosts all come through here.
  remote_fan_in chan RemoteFrameBundle

  // All Conns, bootstrapped and not-yet-boostrapped
  conns []Conn

  // All bootstrapping conns
  bootstraps []bootstrap

  // Earliest StateFrame for which we have seen no events from an engines.
  // This will be the frame on which we start any new connections.
  horizon StateFrame

  // Easy way to accurately count live connections.
  active_conns sync.WaitGroup

  shutdown chan struct{}
}

func (c *Communicator) Start() {
  c.remote_fan_in = make(chan RemoteFrameBundle)
  c.shutdown = make(chan struct{})
  if c.host_conn != nil {
    c.conns = append(c.conns, c.host_conn)
    c.active_conns.Add(1)
    go c.connRoutine(c.host_conn)
  }
  go c.routine()
}

// For engines attempting to connect to a host engine, once the connection has
// been established this function will handle the initial bootstrap.  If
// successful the BootstrapFrame that is returned should be passed to
// Updater.Bootstrap() and all other components can be Start()ed.
func (c *Communicator) Join(conn Conn) (*BootstrapFrame, EngineId, error) {
  // TODO: Should have a timeout on here, maybe 10 seconds?
  data := <-conn.RecvData()
  var initial bootstrapInitialData
  err := QuickGobDecode(&initial, data)
  if err != nil {
    conn.Close()
    return nil, 0, err
  }
  var remote_bundles []FrameBundle
  for {
    select {
    case bundle := <-conn.RecvFrameBundle():
      if bundle.Frame > initial.Horizon {
        remote_bundles = append(remote_bundles, bundle)
      }

    case data := <-conn.RecvData():
      var boot BootstrapFrame
      err = QuickGobDecode(&boot, data)
      if err != nil {
        conn.Close()
        return nil, 0, err
      }
      data, err := QuickGobEncode(true)
      if err != nil {
        conn.Close()
        return nil, 0, err
      }
      conn.SendData(data)

      go func() {
        for _, bundle := range remote_bundles {
          c.Raw_remote_bundles <- bundle
        }
      }()
      c.host_conn = conn
      return &boot, initial.Id, nil
    }
  }
  panic("Unreachable")
}

func (c *Communicator) Shutdown() {
  c.shutdown <- struct{}{}
}
func (c *Communicator) NumConns() int {
  return len(c.conns)
}

// Bootstrapping works as follows:
// Host                   ---                   Client
// StateFrame and Id   ->
// BootstrapFrame      ->
//                            <- Confirmation
// Apply EngineJoined
//                            Listen for the appropriate EngineJoined event
//                Bootstrapping complete
// 
func (c *Communicator) bootstrapRoutine(conn Conn, id EngineId) {
  data, ok := <-conn.RecvData()
  if !ok {
    // TODO: Log this error
    // TODO: If anything in this function fails the conn still needs to be
    // removed from the boostrapping conns list.
    conn.Close()
    return
  }
  var ready bool
  err := QuickGobDecode(&ready, data)
  if err != nil || !ready {
    // TODO: LOG this error
    conn.Close()
  } else {
    // TODO: Make an engine event that joins conn to the game
    c.Local_engine_event <- EngineJoined{id}
    go c.connRoutine(conn)
  }
}

func (c *Communicator) connRoutine(conn Conn) {
  alive := true
  for alive {
    select {
    case _, ok := <-conn.RecvData():
      alive = alive && ok

    case bundle, ok := <-conn.RecvFrameBundle():
      alive = alive && ok
      c.remote_fan_in <- RemoteFrameBundle{bundle, conn}
    }
  }
  c.active_conns.Done()
  // TODO: conn died, probably want to do something here.
}

type bootstrapInitialData struct {
  Horizon StateFrame
  Id      EngineId
}

func (c *Communicator) routine() {
  for {
    select {
    case conn := <-c.Net.NewConns():
      // We send them the stateframe they're starting on and the id they will
      // be assigned when they join the game.
      initial := bootstrapInitialData{
        c.horizon + 1,
        EngineId(RandomId()),
      }
      data, err := QuickGobEncode(initial)
      if err != nil {
        // TODO: LOG this error!
        break
      }
      conn.SendData(data)
      c.conns = append(c.conns, conn)
      boot := bootstrap{
        conn:  conn,
        start: c.horizon + 1,
      }
      c.bootstraps = append(c.bootstraps, boot)
      c.active_conns.Add(1)
      go c.bootstrapRoutine(conn, initial.Id)

    case bundle := <-c.Broadcast_bundles:
      if bundle.Frame > c.horizon {
        c.horizon = bundle.Frame
      }
      for _, conn := range c.conns {
        go conn.SendFrameBundle(bundle)
      }

    case remote_bundle := <-c.remote_fan_in:
      if remote_bundle.bundle.Frame > c.horizon {
        c.horizon = remote_bundle.bundle.Frame
      }
      go func() {
        c.Raw_remote_bundles <- remote_bundle.bundle
      }()
      for _, conn := range c.conns {
        if conn != remote_bundle.conn {
          go conn.SendFrameBundle(remote_bundle.bundle)
        }
      }

    case boostrap_frame := <-c.Bootstrap_frames:
      for _, boot := range c.bootstraps {
        if boostrap_frame.Frame == boot.start {
          data, err := QuickGobEncode(boostrap_frame)
          if err != nil {
            panic(err.Error())
            // TODO: LOG error
            boot.conn.Close()
            continue
          }
          boot.conn.SendData(data)
        }
      }
      // Now remove these bootstrap conns from our list since we've sent them
      // everything they need.
      for i := 0; i < len(c.bootstraps); i++ {
        if boostrap_frame.Frame == c.bootstraps[i].start {
          c.bootstraps[i] = c.bootstraps[len(c.bootstraps)-1]
          c.bootstraps = c.bootstraps[0 : len(c.bootstraps)-1]
        }
      }

    case <-c.shutdown:
      for _, conn := range c.conns {
        conn.Close()
      }
      // Clean out remote_fan_in so that our conn routines can terminate.
      go func() {
        for _ = range c.remote_fan_in {
        }
      }()
      c.active_conns.Wait()
      close(c.remote_fan_in)
      close(c.Raw_remote_bundles)
      return
    }
  }
}
