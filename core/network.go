package core

type EventBatch struct {
  Opaque_data int

  Event Event
}

type RemoteHost interface {
  Data() []byte
  Error() error
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
  Host(ping, join func([]byte) ([]byte, error))

  // Search for hosts on the LAN, sending them data along with the ping.
  Ping(data []byte) []RemoteHost

  // data can be anything.  Returns nil iff the join was successful.
  Join(remote RemoteHost, data []byte) ([]byte, error)

  // Broadcasts an event package.  Must immediately return.
  Send(batch EventBatch)

  // Event packages that have been received from other engines will be
  // available here.
  Receive() <-chan EventBatch

  ActiveConnections() int

  Shutdown()
}
