package core

import (
  "bytes"
  "sync"
  "errors"
  "encoding/gob"
)

// TODO: Pings and Joins should send the transitive closure of connected
// clients.
// TODO: Should let cycles happen.

var host_mutex sync.Mutex
var hosts []*NetworkMock
var host_id int

type ConnectionBatch struct {
  batch   EventBatch
  conn_id int
}
type ConnectionMock struct {
  id       int
  recv     <-chan []byte
  internal chan<- []byte
  send     chan EventBatch

  // Network -> cm.send -> cm.internal -> cm2.recv -> Network2
}

func (cm *ConnectionMock) Send(batch EventBatch) {
  cm.send <- batch
}
func (cm *ConnectionMock) routineRecv(nm *NetworkMock) {
  for batch_data := range cm.recv {
    var batch EventBatch
    err := gob.NewDecoder(bytes.NewBuffer(batch_data)).Decode(&batch)
    if err != nil {
      println("Error with gob decoding")
      // log an error
    } else {
      nm.collect <- ConnectionBatch{batch, cm.id}
    }
  }
}
func (cm *ConnectionMock) routineSend() {
  for batch := range cm.send {
    buffer := bytes.NewBuffer(nil)
    err := gob.NewEncoder(buffer).Encode(batch)
    if err != nil {
      println("Error with gob encoding")
      // log an error
    } else {
      cm.internal <- buffer.Bytes()
    }
  }
}
func makeConnectionMockPair(a, b *NetworkMock) (ConnectionMock, ConnectionMock) {
  a_to_b := make(chan []byte)
  b_to_a := make(chan []byte)
  conn_a := ConnectionMock{
    id:       b.id,
    recv:     b_to_a,
    internal: a_to_b,
    send:     make(chan EventBatch),
  }
  go conn_a.routineRecv(a)
  go conn_a.routineSend()
  conn_b := ConnectionMock{
    id:       a.id,
    recv:     a_to_b,
    internal: b_to_a,
    send:     make(chan EventBatch),
  }
  go conn_b.routineRecv(b)
  go conn_b.routineSend()
  return conn_a, conn_b
}

// NetworkMock is only useful for testing multiple engines in a single process
type NetworkMock struct {
  id   int
  data []byte

  ping, join func([]byte) ([]byte, error)

  connections []ConnectionMock
  collect     chan ConnectionBatch
  incoming    chan EventBatch
  shutdown    chan struct{}
}

func NewNetworkMock() Network {
  host_mutex.Lock()
  defer host_mutex.Unlock()
  var nm NetworkMock
  nm.collect = make(chan ConnectionBatch, 1000)
  nm.shutdown = make(chan struct{})
  nm.incoming = make(chan EventBatch)
  nm.id = host_id
  host_id++
  go nm.routine()
  return &nm
}

func (nm *NetworkMock) Host(ping, join func([]byte) ([]byte, error)) {
  host_mutex.Lock()
  defer host_mutex.Unlock()

  if ping == nil || join == nil {
    nm.ping = nil
    nm.join = nil
  }
  nm.ping = ping
  nm.join = join

  for i := range hosts {
    if hosts[i] == nm {
      if ping == nil || join == nil {
        hosts[i] = hosts[len(hosts)-1]
        hosts = hosts[0 : len(hosts)-1]
      }
      return
    }
  }

  hosts = append(hosts, nm)
}

type networkMockRemoteHost struct {
  data []byte
  err  error
  id   int
}

func (nmrh networkMockRemoteHost) Data() []byte {
  return nmrh.data
}
func (nmrh networkMockRemoteHost) Error() error {
  return nmrh.err
}
func (nm *NetworkMock) Ping(data []byte) []RemoteHost {
  host_mutex.Lock()
  defer host_mutex.Unlock()
  var rhs []RemoteHost
  for i := range hosts {
    data, err := hosts[i].ping(data)
    rh := networkMockRemoteHost{
      data: data,
      err:  err,
      id:   hosts[i].id,
    }
    rhs = append(rhs, rh)
  }
  return rhs
}

func (nm *NetworkMock) Join(remote RemoteHost, data []byte) ([]byte, error) {
  rh, ok := remote.(networkMockRemoteHost)
  if !ok {
    return nil, errors.New("Specified a remote host of an unknown type.")
  }
  if rh.id == nm.id {
    return nil, errors.New("Cannot connect a network to itself.")
  }
  for i := range hosts {
    if hosts[i].id == rh.id {
      for j := range hosts[i].connections {
        if hosts[i].connections[j].id == nm.id {
          return nil, errors.New("Tried to connect to an already connected network.")
        }
      }
      for j := range nm.connections {
        if nm.connections[j].id == hosts[i].id {
          return nil, errors.New("Tried to connect to a network twice.")
        }
      }
      conn_a, conn_b := makeConnectionMockPair(nm, hosts[i])
      nm.connections = append(nm.connections, conn_a)
      hosts[i].connections = append(hosts[i].connections, conn_b)
      return hosts[i].join(data)
    }
  }

  return nil, errors.New("Couldn't find the remote host.")
}

func (nm *NetworkMock) Send(batch EventBatch) {
  host_mutex.Lock()
  defer host_mutex.Unlock()
  for i := range nm.connections {
    nm.connections[i].send <- batch
  }
}

func (nm *NetworkMock) routine() {
  for {
    select {
    case conn_batch := <-nm.collect:
      for _, conn := range nm.connections {
        if conn.id == conn_batch.conn_id {
          continue
        }
        conn.send <- conn_batch.batch
      }
      nm.incoming <- conn_batch.batch

    case <-nm.shutdown:
      for _, conn := range nm.connections {
        close(conn.send)
        close(conn.internal)
      }
      host_mutex.Lock()
      for i := range hosts {
        if hosts[i] == nm {
          hosts[i] = hosts[len(hosts)-1]
          hosts = hosts[0 : len(hosts)-1]
        }
      }
      host_mutex.Unlock()
      return
    }
  }
  for conn_batch := range nm.collect {
    for _, conn := range nm.connections {
      if conn.id == conn_batch.conn_id {
        continue
      }
      conn.send <- conn_batch.batch
    }
  }
}

func (nm *NetworkMock) Receive() <-chan EventBatch {
  return nm.incoming
}

func (nm *NetworkMock) ActiveConnections() int {
  return len(nm.connections)
}

func (nm *NetworkMock) Shutdown() {
  nm.shutdown <- struct{}{}
}
