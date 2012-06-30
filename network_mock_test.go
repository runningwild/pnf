package pnf_test

import (
  "fmt"
  "encoding/gob"
  "runningwild/pnf"
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
  "sort"
  "sync"
)

type EventA struct {
  Data int
}

func init() {
  gob.Register(EventA{})
}
func (e EventA) ApplyFast(pnf.Game)  {}
func (e EventA) Apply(pnf.Game)      {}
func (e EventA) ApplyFinal(pnf.Game) {}

type EventB struct {
  Data string
}

func init() {
  gob.Register(EventA{})
}
func (e EventB) ApplyFast(pnf.Game)  {}
func (e EventB) Apply(pnf.Game)      {}
func (e EventB) ApplyFinal(pnf.Game) {}

func NetworkMockSpec(c gospec.Context) {
  var network_mutex sync.Mutex
  c.Specify("NetworkMocks can connect to eachother and send Events.", func() {
    network_mutex.Lock()
    defer network_mutex.Unlock()
    nm1 := pnf.NewNetworkMock()
    nm2 := pnf.NewNetworkMock()
    defer nm1.Shutdown()
    defer nm2.Shutdown()
    on_ping := func(data []byte) ([]byte, error) {
      return []byte("Join us! " + string(data)), nil
    }
    on_join := func([]byte) ([]byte, error) {
      return []byte("You've joined us!"), nil
    }
    nm1.Host(on_ping, on_join)
    rhs := nm2.Ping([]byte("Monkeys"))
    c.Expect(len(rhs), Equals, 1)
    c.Expect(string(rhs[0].Data()), Equals, "Join us! Monkeys")
    res, err := nm2.Join(rhs[0], []byte("woo!"))
    c.Expect(err, Equals, error(nil))
    c.Expect(string(res), Equals, "You've joined us!")

    var eb1 pnf.EventBatch
    eb1.Opaque_data = 123
    eb1.Event = EventA{555}
    eb2 := eb1
    go nm2.Send(eb1)
    eb3 := <-nm1.Receive()
    c.Expect(eb3, Equals, eb2)
  })

  c.Specify("NetworkMocks can differentiate between multiple hosts.", func() {
    network_mutex.Lock()
    defer network_mutex.Unlock()
    hosts := make([]pnf.Network, 10)
    for i := range hosts {
      hosts[i] = pnf.NewNetworkMock()
      defer hosts[i].Shutdown()
      num := i
      on_ping := func(data []byte) ([]byte, error) {
        return []byte(fmt.Sprintf("Ping(%d)", num)), nil
      }
      on_join := func([]byte) ([]byte, error) {
        return []byte(fmt.Sprintf("Join(%d)", num)), nil
      }
      hosts[i].Host(on_ping, on_join)
    }
    rhs := hosts[0].Ping([]byte("Waffle"))
    c.Expect(len(rhs), Equals, 10)
    var resps []string
    for i := range rhs {
      resps = append(resps, string(rhs[i].Data()))
    }
    sort.Strings(resps)
    for i := range resps {
      c.Expect(resps[i], Equals, fmt.Sprintf("Ping(%d)", i))
    }
    res, err := hosts[0].Join(rhs[4], []byte("Pancake"))
    c.Expect(err, Equals, error(nil))
    c.Expect(string(res), Equals, fmt.Sprintf("Join(%d)", 4))
  })
}
