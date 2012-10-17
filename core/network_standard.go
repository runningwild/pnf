package core

import (
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
  remote   RemoteHost
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
// that was successful it response with the specified data.
func (n *networkTcpUdp) launchPingRoutine(die chan struct{}) error {
  laddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", n.port))
  if err != nil {
    return err
  }
  listener, err := net.ListenUDP("udp", laddr)
  if err != nil {
    return err
  }
  err = listener.SetDeadline(time.Now().Add(time.Second))
  if err != nil {
    return err
  }

  go func() {
    // TODO: Should either document the size of this buffer or make it configurable.
    buf := make([]byte, 1024)
    for {
      size, raddr, err := listener.ReadFromUDP(buf)
      select {
      case <-die:
        return
      default:
      }
      resp, err := n.ping(buf[0:size])
      if err == nil {
        // We'll block on this, but it's udp, so we probably won't hang.
        laddr, err := net.ResolveUDPAddr("udp", ":")
        if err == nil {
          resp_conn, err := net.DialUDP("udp", laddr, raddr)
          if err == nil {
            resp_conn.Write(resp)
          }
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
    return err
  }
  listener, err := net.ListenTCP("tcp", laddr)
  if err != nil {
    return err
  }
  listener.Accept()
  return err
  for {
    select {
    case <-die:
      return nil
    default:
    }
  }
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

func (n *networkTcpUdp) handlePingRequest(req pingRequest) (resp pingResponse) {
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
  defer conn.Close()
  _, err = conn.Write(req.data)
  if err != nil {
    resp.err = errors.New(fmt.Sprintf("Unable to broadcast: %v\n", err))
    return
  }
  return
}

func (n *networkTcpUdp) handleJoinRequest(req joinRequest) (resp joinResponse) {
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
  n.requests <- joinRequest{c, remote, data}
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
