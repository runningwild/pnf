package core

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
  Remote_bundles chan<- FrameBundle

  // Bundles from remote hosts all come through here.
  remote_fan_in chan FrameBundle

  conns []Conn

  shutdown chan struct{}
}

func (c *Communicator) Start() {
  c.remote_fan_in = make(chan FrameBundle)
  c.shutdown = make(chan struct{})
  go c.routine()
}

func (c *Communicator) Shutdown() {
  c.shutdown <- struct{}{}
}

func (c *Communicator) connRoutine(conn Conn) {
  alive := true
  for alive {
    select {
    case _, ok := <-conn.RecvData():
      alive = alive && ok

    case bundle, ok := <-conn.RecvFrameBundle():
      alive = alive && ok
      c.remote_fan_in <- bundle
    }
  }
  // TODO: conn died, probably want to do something here.
}

func (c *Communicator) routine() {
  for {
    select {
    case conn := <-c.Net.NewConns():
      c.conns = append(c.conns, conn)
      go c.connRoutine(conn)

    case bundle := <-c.Broadcast_bundles:
      for _, conn := range c.conns {
        conn.SendFrameBundle(bundle)
      }

    case bundle := <-c.remote_fan_in:
      c.Remote_bundles <- bundle

    case <-c.shutdown:
      for _, conn := range c.conns {
        conn.Close()
      }
      close(c.Remote_bundles)
      break
    }
  }
}
