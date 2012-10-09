package core

type EventBatch struct {
  Opaque_data int

  Event Event
}

type RemoteHost interface {
  Data() []byte
  Error() error
}

type Heartbeat struct {
  Id int
}

// Higher level version of net.Conn
// Can send/recv the following:
//   FrameBundles
//   Heartbeats/pings
type Conn interface {
  SendData([]byte)
  RecvData() <-chan []byte
  SendFrameBundle(bundle FrameBundle)
  RecvFrameBundle() <-chan FrameBundle

  // Primarily for testing.  Returns an id that is unique among all connection
  // pairs.  For mock networks and mock conns we can guarantee that both
  // connections in a pair will have the same Id, which can help sometimes
  // with debugging.
  Id() int

  // TODO: Must be able to tell if the connection died

  Close() error
}

// A Network maintains connections with other engines.
// Host - allow others to connect to it.
// Find - find hosts.
// Join - connect to another engine.
// Send and Recv events from other engines.
// Keep this engine in sync with other engines as much as possible.
type Network interface {
  // Calling Host with join == nil will turn hosting off.
  // ping: a function called when someone pings this host, the return value
  // will be sent to that network.  if error is not nil it indicates that the
  // game cannot be joined.
  // join: like ping, called when someone requests to join, if error is not
  // nil it indicates that the join failed.
  // both ping and join may be called concurrently, so lock if you need to.
  Host(ping func([]byte) ([]byte, error), join func([]byte) error)

  // Search for hosts on the LAN, sending them data along with the ping.
  Ping(data []byte) ([]RemoteHost, error)

  // data can be anything.
  Join(remote RemoteHost, data []byte) (Conn, error)

  // All new connections will be made available on this channel.
  NewConns() <-chan Conn

  ActiveConnections() int

  Shutdown()
}
