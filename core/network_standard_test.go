package core_test

import (
  "bytes"
  "encoding/gob"
  "fmt"
  . "github.com/orfjackal/gospec/src/gospec"
  "github.com/orfjackal/gospec/src/gospec"
  "net"
  "runningwild/pnf/core"
  "time"
)

func NetworkStandardGobbingSpec(c gospec.Context) {
  c.Specify("Basic standard network gobbing.", func() {
    var payload core.TcpConnPayload
    payload.Data = make([]byte, 4096)
    for i := range payload.Data {
      payload.Data[i] = byte(i)
    }

    port := 2491
    laddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
    c.Expect(err, Equals, error(nil))
    if err != nil {
      return
    }
    listener, err := net.ListenTCP("tcp", laddr)
    c.Expect(err, Equals, error(nil))
    if err != nil {
      return
    }
    done := make(chan struct{})
    go func() {
      conn, err := listener.Accept()
      c.Expect(err, Equals, error(nil))
      if err != nil {
        return
      }
      var p2 core.TcpConnPayload
      dec := gob.NewDecoder(conn)
      err = dec.Decode(&p2)
      c.Expect(err, Equals, error(nil))
      c.Expect(string(payload.Data), Equals, string(p2.Data))
      done <- struct{}{}
    }()

    raddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", port))
    c.Expect(err, Equals, error(nil))
    if err != nil {
      return
    }
    conn, err := net.DialTCP("tcp", nil, raddr)
    c.Expect(err, Equals, error(nil))
    if err != nil {
      return
    }

    enc := gob.NewEncoder(conn)
    err = enc.Encode(payload)
    c.Expect(err, Equals, error(nil))
    <-done
  })
  c.Specify("Large payload over standard network.", func() {
    var payload core.TcpConnPayload
    payload.Bundle = new(core.FrameBundle)
    payload.Bundle.Frame = 12323
    payload.Bundle.Bundle = map[core.EngineId]core.AllEvents{}
    for i := core.EngineId(1); i < 300; i++ {
      ae := core.AllEvents{
        Engine: make([]core.EngineEvent, 300),
        Game:   make([]core.Event, 300),
      }
      for j := range ae.Game {
        ae.Game[j] = EventA{}
      }
      payload.Bundle.Bundle[i] = ae
    }

    buf := bytes.NewBuffer(nil)
    enc := gob.NewEncoder(buf)
    err := enc.Encode(payload)
    c.Expect(err, Equals, error(nil))
    var p2 core.TcpConnPayload
    dec := gob.NewDecoder(buf)
    err = dec.Decode(&p2)
    c.Expect(err, Equals, error(nil))
    c.Expect(payload.Bundle.Frame, Equals, p2.Bundle.Frame)
  })
}

func NetworkStandardSpec(c gospec.Context) {
  c.Specify("Basic standard network functionality.", func() {
    port := int(core.RandomId()%10000 + 1000)
    host, err := core.MakeTcpUdpNetwork(port)
    c.Expect(err, Equals, error(nil))
    client, err := core.MakeTcpUdpNetwork(port)
    c.Expect(err, Equals, error(nil))
    println(host, client)

    ping := func(data []byte) ([]byte, error) {
      return []byte(fmt.Sprintf("Ping(%d)", string(data))), nil
    }

    join := func(data []byte) error {
      return nil
    }

    host.Host(ping, join)
    time.Sleep(time.Millisecond * 100)
    rhs, err := client.Ping([]byte("MONKEYS"))
    c.Expect(err, Equals, error(nil))
    c.Expect(len(rhs), Equals, 1)
    if len(rhs) != 1 {
      return
    }

    conn, err := client.Join(rhs[0], rhs[0].Data())
    c.Expect(conn, Not(Equals), core.Conn(nil))
    c.Expect(err, Equals, error(nil))

    var new_conn core.Conn
    select {
    case new_conn = <-host.NewConns():
    default:
    }
    c.Expect(new_conn, Not(Equals), core.Conn(nil))

    new_conn.SendData([]byte("MONKEYS RULE!!!"))
    var recv_data []byte
    select {
    case recv_data = <-conn.RecvData():
    case <-time.After(time.Millisecond * 100):
    }
    c.Expect(string(recv_data), Equals, "MONKEYS RULE!!!")
    host.Shutdown()
  })
}
