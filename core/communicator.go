package core

import (
  "sync"
)

type RemoteFrameBundle struct {
  bundle FrameBundle
  conn   Conn
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

  // This is necessary for starting up a client engine.  A host can safely
  // leave this as nil.
  Host_conn Conn

  // Bundles from remote hosts all come through here.
  remote_fan_in chan RemoteFrameBundle

  conns []Conn

  // Easy way to accurately count live connections
  active_conns sync.WaitGroup

  shutdown chan struct{}
}

func (c *Communicator) Start() {
  c.remote_fan_in = make(chan RemoteFrameBundle)
  c.shutdown = make(chan struct{})
  if c.Host_conn != nil {
    c.conns = append(c.conns, c.Host_conn)
    c.active_conns.Add(1)
    go c.connRoutine(c.Host_conn)
  }
  go c.routine()
}

func (c *Communicator) Shutdown() {
  c.shutdown <- struct{}{}
}
func (c *Communicator) NumConns() int {
  return len(c.conns)
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

func (c *Communicator) routine() {
  for {
    select {
    case conn := <-c.Net.NewConns():
      c.conns = append(c.conns, conn)
      c.active_conns.Add(1)
      go c.connRoutine(conn)

    case bundle := <-c.Broadcast_bundles:
      for _, conn := range c.conns {
        go conn.SendFrameBundle(bundle)
      }

    case remote_bundle := <-c.remote_fan_in:
      go func() {
        c.Raw_remote_bundles <- remote_bundle.bundle
      }()
      for _, conn := range c.conns {
        if conn != remote_bundle.conn {
          go conn.SendFrameBundle(remote_bundle.bundle)
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
