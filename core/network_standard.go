package core

import (
  "bytes"
  "encoding/gob"
  "errors"
  "fmt"
  "net"
  "time"
)

type networkTcpUdp struct {
  requests  chan interface{}
  new_conns chan Conn
  port      int
  ping      func([]byte) ([]byte, error)
  join      func([]byte) error
}

type hostRequest struct {
  ping func([]byte) ([]byte, error)
  join func([]byte) error
}

type pingRequest struct {
  response chan pingResponse
  data     []byte
}
type pingResponse struct {
  hosts []RemoteHost
  err   error
}

type joinRequest struct {
  response chan joinResponse
  remote   standardRemoteHost
  data     []byte
}
type joinResponse struct {
  conn Conn
  err  error
}

// Binds to udp and tcp ports as specified.
func MakeTcpUdpNetwork(port int) (Network, error) {
  var n networkTcpUdp
  n.port = port
  n.requests = make(chan interface{})
  n.new_conns = make(chan Conn)
  go n.routine()
  return &n, nil
}

// Listens on a udp port, if it receives any data it passes it to the ping function, then if
// that was successful it responds with the specified data.
func (n *networkTcpUdp) launchPingRoutine(die chan struct{}) error {
  laddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", n.port))
  if err != nil {
    return err
  }
  listener, err := net.ListenUDP("udp", laddr)
  if err != nil {
    return err
  }
  go func() {
    defer listener.Close()

    // TODO: Should either document the size of this buffer or make it configurable.
    buf := make([]byte, 1024)
    for {
      err = listener.SetDeadline(time.Now().Add(time.Second))
      if err != nil {
        go func() {
          <-die
        }()
        return
      }
      size, raddr, err := listener.ReadFromUDP(buf)
      if err != nil {
        if err.(net.Error).Timeout() {
          continue
        } else {
          go func() {
            <-die
          }()
          return
        }
      }
      select {
      case <-die:
        return
      default:
      }
      resp, err := n.ping(buf[0:size])

      // When testing locally we need a very slight delay so that we have time
      // to start listening for the response.
      time.Sleep(time.Millisecond * 10)

      if err == nil {
        // We'll block on this, but it's udp, so we probably won't hang.
        resp_conn, err := net.DialUDP("udp", nil, raddr)
        if err == nil {
          _, err = resp_conn.Write(resp)
          resp_conn.Close()
        }
      }
    }
  }()

  return nil
}

// Listens on a tcp port
func (n *networkTcpUdp) launchJoinRoutine(die chan struct{}) error {
  laddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", n.port))
  if err != nil {
    return errors.New(fmt.Sprintf("Unable to resolve local tcp addr: %v", err))
  }
  listener, err := net.ListenTCP("tcp", laddr)
  if err != nil {
    return errors.New(fmt.Sprintf("Unable to listen for joins: %v", err))
  }

  go func() {
    <-die
    listener.Close()
  }()

  go func() {
    for {
      raw_con, err := listener.Accept()
      if err != nil {
        panic("FUCK")
        return
      }
      go func() {
        buf := make([]byte, 1024)
        num, err := raw_con.Read(buf)
        if err != nil {
          return
        }
        err = n.join(buf[0:num])
        raw_con.SetWriteDeadline(time.Now().Add(time.Second))
        if err != nil {
          raw_con.Write([]byte(fmt.Sprintf("FAIL: %v", err)))
          return
        }
        _, err = raw_con.Write([]byte("SUCCESS"))
        if err != nil {
          return
        }
        raw_con.SetDeadline(time.Time{})
        conn := makeTcpConn(raw_con.(*net.TCPConn))
        n.new_conns <- conn
      }()
    }
  }()
  return nil
}

func (n *networkTcpUdp) routine() {
  var kill chan struct{}
  for _req := range n.requests {
    switch req := _req.(type) {
    case hostRequest:
      if kill != nil {
        kill <- struct{}{}
        kill <- struct{}{}
      }
      if req.ping == nil || req.join == nil {
        n.ping = nil
        n.join = nil
        kill = nil
      } else {
        n.ping = req.ping
        n.join = req.join
        kill = make(chan struct{})
        err := n.launchPingRoutine(kill)
        if err != nil {
          kill = nil
          continue
        }
        err = n.launchJoinRoutine(kill)
        if err != nil {
          kill <- struct{}{}
          kill = nil
        }
      }

    case pingRequest:
      req.response <- n.handlePingRequest(req)

    case joinRequest:
      req.response <- n.handleJoinRequest(req)
    }
  }
}

type standardRemoteHost struct {
  data []byte
  ip   string
  port int
}

func (rh standardRemoteHost) Data() []byte {
  return rh.data
}
func (rh standardRemoteHost) Error() error {
  return nil
}

func (n *networkTcpUdp) handlePingRequest(req pingRequest) (resp pingResponse) {
  // Now we'll broadcast a simple ping packet and then listen for one second.
  raddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", n.port))
  if err != nil {
    resp.err = errors.New(fmt.Sprintf("Unable to resolve udp raddr: %v\n", err))
    return
  }

  conn, err := net.DialUDP("udp", nil, raddr)
  if err != nil {
    resp.err = errors.New(fmt.Sprintf("Unable to dial: %v\n", err))
    return
  }

  _, err = conn.Write(req.data)
  if err != nil {
    resp.err = errors.New(fmt.Sprintf("Unable to broadcast: %v\n", err))
    return
  }
  laddr := conn.LocalAddr().(*net.UDPAddr)
  conn.Close()

  conn, err = net.ListenUDP("udp", laddr)
  if err != nil {
    resp.err = errors.New(fmt.Sprintf("Unable to listen: %v\n", err))
    return
  }
  defer conn.Close()

  // TODO: Maybe this 1 second should be configurable
  err = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
  if err != nil {
    resp.err = errors.New(fmt.Sprintf("Unable to set read deadline: %v\n", err))
    return
  }

  data := make([]byte, 2048)
  for {
    n, addr, err := conn.ReadFromUDP(data)
    if err != nil {
      return
    }
    var rh standardRemoteHost
    rh.data = make([]byte, n)
    copy(rh.data, data)
    rh.port = addr.Port
    rh.ip = addr.IP.String()
    resp.hosts = append(resp.hosts, rh)
  }
  return
}

func (n *networkTcpUdp) handleJoinRequest(req joinRequest) (resp joinResponse) {
  raddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", req.remote.ip, n.port))
  if err != nil {
    resp.err = errors.New(fmt.Sprintf("Unable to resolve remote tcp addr: %v", err))
    return
  }
  conn, err := net.DialTCP("tcp", nil, raddr)
  if err != nil {
    resp.err = errors.New(fmt.Sprintf("Unable to dial: %v", err))
    return
  }

  buf := make([]byte, 1024)
  conn.SetDeadline(time.Now().Add(time.Second))

  _, err = conn.Write(req.data)
  if err != nil {
    resp.err = errors.New(fmt.Sprintf("Unable to write: %v", err))
    return
  }

  num, err := conn.Read(buf)
  if err != nil {
    resp.err = errors.New(fmt.Sprintf("Unable to read: %v", err))
    return
  }

  if string(buf[0:4]) == "FAIL" {
    resp.err = errors.New(fmt.Sprintf("Unable to join: %s", buf[0:num]))
    return
  }
  conn.SetDeadline(time.Time{})
  resp.conn = makeTcpConn(conn)
  return
}

func (n *networkTcpUdp) Host(ping func([]byte) ([]byte, error), join func([]byte) error) {
  n.requests <- hostRequest{ping, join}
}

func (n *networkTcpUdp) Ping(data []byte) ([]RemoteHost, error) {
  c := make(chan pingResponse)
  n.requests <- pingRequest{c, data}
  response := <-c
  return response.hosts, response.err
}

func (n *networkTcpUdp) Join(remote RemoteHost, data []byte) (Conn, error) {
  c := make(chan joinResponse)
  srh, ok := remote.(standardRemoteHost)
  if !ok {
    return nil, errors.New("Invalid RemoteHost")
  }
  n.requests <- joinRequest{c, srh, data}
  response := <-c
  return response.conn, response.err
}

func (n *networkTcpUdp) NewConns() <-chan Conn {
  return n.new_conns
}

func (n *networkTcpUdp) ActiveConnections() int {
  return 0
}

func (n *networkTcpUdp) Shutdown() {
  close(n.requests)
}

type tcpConn struct {
  raw  *net.TCPConn
  data struct {
    from_net chan []byte
    to_pnf   chan []byte
  }
  bundle struct {
    from_net chan FrameBundle
    to_pnf   chan FrameBundle
  }
  send struct {
    from_pnf chan TcpConnPayload
    to_net   chan TcpConnPayload
  }
  kill chan struct{}
}

func makeTcpConn(raw *net.TCPConn) *tcpConn {
  var c tcpConn
  c.raw = raw
  c.data.from_net = make(chan []byte, 100)
  c.data.to_pnf = make(chan []byte, 100)
  c.bundle.from_net = make(chan FrameBundle, 100)
  c.bundle.to_pnf = make(chan FrameBundle, 100)
  c.send.from_pnf = make(chan TcpConnPayload)
  c.send.to_net = make(chan TcpConnPayload)

  c.kill = make(chan struct{})
  go c.readRoutine()
  go c.writeRoutine()
  go c.sendRoutine()
  go c.recvDataRoutine()
  go c.recvBundleRoutine()
  return &c
}

func (c *tcpConn) terminate() {
  // Send signals along c.kill
}

func check(err error) {
  if err != nil {
    panic(err)
  }
}

type TcpConnPayload struct {
  Data   []byte
  Bundle *FrameBundle
}

// func (cp *TcpConnPayload) GobDecode(data []byte) error {
//   buf := bytes.NewBuffer(data)
//   dec := gob.NewDecoder(buf)
//   var length uint32
//   err := dec.Decode(&length)
//   if err != nil {
//     return err
//   }
//   cp.Data = make([]byte, int(length))
//   _, err = buf.Read(cp.Data)
//   if err != nil {
//     return err
//   }
//   var ok bool
//   err = dec.Decode(&ok)
//   if err != nil {
//     return err
//   }
//   if !ok {
//     return nil
//   }
//   err = dec.Decode(&cp.Bundle)
//   if err != nil {
//     return err
//   }
//   return nil
// }

// func (cp *TcpConnPayload) GobEncode() ([]byte, error) {
//   buf := bytes.NewBuffer(nil)
//   enc := gob.NewEncoder(buf)
//   err := enc.Encode(uint32(len(cp.Data)))
//   if err != nil {
//     return nil, err
//   }
//   _, err = buf.Write(cp.Data)
//   if err != nil {
//     return nil, err
//   }
//   err = enc.Encode(cp.Bundle != nil)
//   if err != nil {
//     return nil, err
//   }
//   if cp.Bundle != nil {
//     err = enc.Encode(cp.Bundle)
//     if err != nil {
//       return nil, err
//     }
//   }
//   return buf.Bytes(), nil
// }

// func (c *tcpConn) readRoutine() {
//   dec := gob.NewDecoder(c.raw)
//   for {
//     var payload TcpConnPayload
//     err := dec.Decode(&payload)
//     if err != nil {
//       c.terminate()
//       fmt.Printf("Error in readRoutine: %v\n", err)
//       return
//     }
//     if payload.Bundle != nil {
//       c.bundle.from_net <- *payload.Bundle
//     } else {
//       c.data.from_net <- payload.Data
//     }
//   }
// }

func (c *tcpConn) readRoutine() {
  db := bytes.NewBuffer(nil)
  dec := gob.NewDecoder(db)
  buf := make([]byte, 4096*256)
  for {
    tbuf := buf[:]
    var payload TcpConnPayload
    n, err := c.raw.Read(tbuf)
    if err != nil {
      panic(err.Error())
    }
    tbuf = tbuf[:n]
    db.Write(tbuf)
    err = dec.Decode(&payload)
    if err != nil {
      c.terminate()
      fmt.Printf("Error in readRoutine: %v\n", err)
      return
    }
    if payload.Bundle != nil {
      c.bundle.from_net <- *payload.Bundle
    } else {
      c.data.from_net <- payload.Data
    }
  }
}

// func (c *tcpConn) writeRoutine() {
//   enc := gob.NewEncoder(c.raw)
//   for payload := range c.send.to_net {
//     err := enc.Encode(payload)
//     if err != nil {
//       c.terminate()
//       fmt.Printf("Error in writeRoutine: %v\n", err)
//       return
//     }
//     fmt.Printf("Encoded payload: %v\n", payload)
//   }
// }

func (c *tcpConn) writeRoutine() {
  buf := bytes.NewBuffer(nil)
  enc := gob.NewEncoder(buf)
  for payload := range c.send.to_net {
    err := enc.Encode(payload)
    if err != nil {
      c.terminate()
      fmt.Printf("Error in writeRoutine: %v\n", err)
      return
    }
    _, err = c.raw.Write(buf.Bytes())
    buf.Reset()
    if err != nil {
      c.terminate()
      fmt.Printf("Error in writeRoutine: %v\n", err)
      return
    }
  }
}

func (c *tcpConn) sendRoutine() {
  var queue []TcpConnPayload
  var out chan TcpConnPayload
  var payload TcpConnPayload
  for {
    if len(queue) > 0 {
      out = c.send.to_net
      payload = queue[0]
    } else {
      out = nil
    }
    select {
    case payload = <-c.send.from_pnf:
      queue = append(queue, payload)
    case out <- payload:
      queue = queue[1:]
    }
  }
}

// Buffers infinitely, so that we don't rely on the capacity of any channel.
func (c *tcpConn) recvDataRoutine() {
  var queue [][]byte
  var out chan []byte
  var datum []byte
  for {
    if len(queue) > 0 {
      out = c.data.to_pnf
      datum = queue[0]
    } else {
      out = nil
    }
    select {
    case data := <-c.data.from_net:
      queue = append(queue, data)
    case out <- datum:
      queue = queue[1:]
    case <-c.kill:
      return
    }
  }
}

// Exactly like recvDataRoutine(), but for the FrameBundles
func (c *tcpConn) recvBundleRoutine() {
  var queue []FrameBundle
  var out chan FrameBundle
  var datum FrameBundle
  for {
    if len(queue) > 0 {
      out = c.bundle.to_pnf
      datum = queue[0]
    } else {
      out = nil
    }
    select {
    case bundle := <-c.bundle.from_net:
      queue = append(queue, bundle)
    case out <- datum:
      queue = queue[1:]
    case <-c.kill:
      return
    }
  }
}

func (c *tcpConn) SendData(data []byte) {
  c.send.from_pnf <- TcpConnPayload{Data: data}
}
func (c *tcpConn) RecvData() <-chan []byte {
  return c.data.to_pnf
}
func (c *tcpConn) SendFrameBundle(bundle FrameBundle) {
  c.send.from_pnf <- TcpConnPayload{Bundle: &bundle}
}
func (c *tcpConn) RecvFrameBundle() <-chan FrameBundle {
  return c.bundle.to_pnf
}
func (c *tcpConn) Id() int {
  return 0
}
func (c *tcpConn) Close() error {
  return nil
}
