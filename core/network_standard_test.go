package core_test

import (
  "fmt"
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
  "runningwild/pnf/core"
  "time"
)

func NetworkStandardSpec(c gospec.Context) {
  c.Specify("Basic standard network functionality.", func() {
    port := int(core.RandomId()%10000 + 1000)
    host, err := core.MakeTcpUdpNetwork(port)
    c.Expect(err, Equals, error(nil))
    client, err := core.MakeTcpUdpNetwork(port)
    c.Expect(err, Equals, error(nil))
    println(host, client)

    ping := func(data []byte) ([]byte, error) {
      fmt.Printf("In ping func!\n")
      return []byte(fmt.Sprintf("Ping(%d)", string(data))), nil
    }

    join := func(data []byte) error {
      fmt.Printf("In join func!\n")
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
