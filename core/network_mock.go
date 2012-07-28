package core

import (
  "fmt"
  "bytes"
  "sync"
  "errors"
  "encoding/gob"
)

// TODO: Pings and Joins should send the transitive closure of connected
// clients.
// TODO: Shouldn't let cycles happen.

type NetworkMock struct {
  host_mutex sync.Mutex
  hosts      []*HostMock
  host_id    int
}

type ConnMock struct {
  // mimics transfer of data over tcp
  send chan<- []byte
  recv <-chan []byte

  // typed data, before and after it travels through recv
  send_bytes, recv_bytes   chan []byte
  send_bundle, recv_bundle chan FrameBundle

  shutdown chan struct{}
}
type dataContainer struct {
  Data         []byte
  Frame_bundle *FrameBundle
}

func (c *ConnMock) routine() {
  for {
    var dc dataContainer
    send := false
    select {
    case <-c.shutdown:
      close(c.recv_bytes)
      close(c.recv_bundle)
      return

    case data := <-c.send_bytes:
      dc.Data = data
      send = true

    case frame_bundle := <-c.send_bundle:
      dc.Frame_bundle = &frame_bundle
      send = true

    case data := <-c.recv:
      dec := gob.NewDecoder(bytes.NewBuffer(data))
      var dc dataContainer
      err := dec.Decode(&dc)
      if err != nil {
        panic(err)
        // TODO: What to do?
      }
      go func() {
        switch {
        case dc.Data != nil:
          c.recv_bytes <- dc.Data
        case dc.Frame_bundle != nil:
          c.recv_bundle <- *dc.Frame_bundle
        }
      }()
    }
    if send {
      buf := bytes.NewBuffer(nil)
      enc := gob.NewEncoder(buf)
      err := enc.Encode(dc)
      if err != nil {
        panic(err)
        // TODO: What to do?
      }
      go func() {
        c.send <- buf.Bytes()
      }()
    }
  }
}

func (c *ConnMock) SendData(data []byte) {
  c.send_bytes <- data
}
func (c *ConnMock) RecvData() <-chan []byte {
  return c.recv_bytes
}
func (c *ConnMock) SendFrameBundle(frame_bundle FrameBundle) {
  c.send_bundle <- frame_bundle
}
func (c *ConnMock) RecvFrameBundle() <-chan FrameBundle {
  return c.recv_bundle
}
func (c *ConnMock) Close() error {
  fmt.Printf("Close(%p): %v\n", c, c.shutdown)
  c.shutdown <- struct{}{}
  c.shutdown <- struct{}{}
  return nil
}

func makeConnMockPair(hm1, hm2 *HostMock) (Conn, Conn) {
  var c1, c2 ConnMock

  c1.recv_bundle = make(chan FrameBundle)
  c1.send_bundle = make(chan FrameBundle)
  c1.recv_bytes = make(chan []byte)
  c1.send_bytes = make(chan []byte)
  c1.shutdown = make(chan struct{})
  c2.recv_bundle = make(chan FrameBundle)
  c2.send_bundle = make(chan FrameBundle)
  c2.recv_bytes = make(chan []byte)
  c2.send_bytes = make(chan []byte)
  c2.shutdown = make(chan struct{})

  send_1 := make(chan []byte)
  recv_1 := make(chan []byte)
  send_2 := make(chan []byte)
  recv_2 := make(chan []byte)

  c1.send = send_1
  c1.recv = recv_1
  c2.send = send_2
  c2.recv = recv_2

  cd1 := hostConnMockData{
    remote_id:   hm2.id,
    send:        send_1,
    recv:        recv_1,
    remote_send: send_2,
    remote_recv: recv_2,
    shutdown:    c1.shutdown,
  }
  cd2 := hostConnMockData{
    remote_id:   hm1.id,
    send:        send_2,
    recv:        recv_2,
    remote_send: send_1,
    remote_recv: recv_1,
    shutdown:    c2.shutdown,
  }
  fmt.Printf("Open(%p): %v\n", c1.shutdown)
  fmt.Printf("Open(%p): %v\n", c2.shutdown)
  go c1.routine()
  go c2.routine()

  hm1.mutex.Lock()
  hm2.mutex.Lock()
  hm1.conn_data[&c1] = &cd1
  hm2.conn_data[&c2] = &cd2
  go hm1.connRoutine(&cd1)
  go hm2.connRoutine(&cd2)
  hm2.mutex.Unlock()
  hm1.mutex.Unlock()

  return &c1, &c2
}

type hostConnMockData struct {
  // host_id of the remote host
  remote_id int

  // send and recv correspond to send and recv on the local ConnMock
  send <-chan []byte
  recv chan<- []byte

  // remote_send and remote_recv correspond to send and recv on the remote
  // ConnMock
  remote_send <-chan []byte
  remote_recv chan<- []byte

  shutdown chan struct{}
}

// HostMock is only useful for testing multiple engines in a single process
type HostMock struct {
  net *NetworkMock

  id   int
  data []byte

  ping func([]byte) ([]byte, error)
  join func([]byte) error

  conn_data map[*ConnMock]*hostConnMockData

  new_conns chan Conn

  mutex sync.Mutex
}

func NewHostMock(net *NetworkMock) Network {
  net.host_mutex.Lock()
  defer net.host_mutex.Unlock()
  var hm HostMock
  hm.net = net
  hm.id = hm.net.host_id
  hm.net.host_id++
  hm.conn_data = make(map[*ConnMock]*hostConnMockData)
  hm.new_conns = make(chan Conn)
  hm.net.hosts = append(hm.net.hosts, &hm)
  return &hm
}

func (hm *HostMock) connRoutine(cd *hostConnMockData) {
  var wg sync.WaitGroup
  for {
    select {
    case data := <-cd.send:
      wg.Add(1)
      go func() {
        cd.remote_recv <- data
        wg.Done()
      }()

    case data := <-cd.remote_send:
      wg.Add(1)
      go func() {
        cd.recv <- data
        wg.Done()
      }()

    case <-cd.shutdown:
      wg.Wait()
      return
    }
  }
}

func (hm *HostMock) Host(ping func([]byte) ([]byte, error), join func([]byte) error) {
  hm.net.host_mutex.Lock()
  defer hm.net.host_mutex.Unlock()
  hm.ping = ping
  hm.join = join
}

type networkMockRemoteHost struct {
  data []byte
  err  error
  id   int
}

func (hmrh networkMockRemoteHost) Data() []byte {
  return hmrh.data
}
func (hmrh networkMockRemoteHost) Error() error {
  return hmrh.err
}
func (hm *HostMock) Ping(data []byte) []RemoteHost {
  hm.net.host_mutex.Lock()
  defer hm.net.host_mutex.Unlock()
  var rhs []RemoteHost
  for i := range hm.net.hosts {
    if hm.net.hosts[i].ping == nil {
      continue
    }
    data, err := hm.net.hosts[i].ping(data)
    rh := networkMockRemoteHost{
      data: data,
      err:  err,
      id:   hm.net.hosts[i].id,
    }
    rhs = append(rhs, rh)
  }
  return rhs
}

func (hm *HostMock) Join(remote RemoteHost, data []byte) (Conn, error) {
  rh, ok := remote.(networkMockRemoteHost)
  if !ok {
    return nil, errors.New("Specified a remote host of an unknown type.")
  }
  if rh.id == hm.id {
    return nil, errors.New("Cannot connect a network to itself.")
  }
  for _, data := range hm.conn_data {
    if data.remote_id == rh.id {
      return nil, errors.New("Tried to connect to a network twice.")
    }
  }
  hm.net.host_mutex.Lock()
  defer hm.net.host_mutex.Unlock()
  for i := range hm.net.hosts {
    if hm.net.hosts[i].id == rh.id {
      for _, data := range hm.net.hosts[i].conn_data {
        if data.remote_id == hm.id {
          return nil, errors.New("Tried to connect to an already connected network.")
        }
      }
      err := hm.net.hosts[i].join(data)
      if err != nil {
        return nil, err
      }
      c1, c2 := makeConnMockPair(hm, hm.net.hosts[i])
      go func() {
        hm.net.hosts[i].new_conns <- c2
      }()
      return c1, nil
    }
  }

  return nil, errors.New("Couldn't find the remote host.")
}

func (hm *HostMock) NewConns() <-chan Conn {
  return hm.new_conns
}

func (hm *HostMock) ActiveConnections() int {
  return len(hm.conn_data)
}

func (hm *HostMock) Shutdown() {
  hm.Host(nil, nil)
  hm.net.host_mutex.Lock()
  defer hm.net.host_mutex.Unlock()
  for i := range hm.net.hosts {
    if hm.net.hosts[i] == hm {
      hm.net.hosts[i] = hm.net.hosts[len(hm.net.hosts)-1]
      hm.net.hosts = hm.net.hosts[0 : len(hm.net.hosts)-1]
      break
    }
  }
}
