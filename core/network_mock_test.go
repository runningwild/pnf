package core_test

import (
  "fmt"
  "encoding/gob"
  "runningwild/pnf/core"
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
func (e EventA) ApplyFast(core.Game)  {}
func (e EventA) Apply(core.Game)      {}
func (e EventA) ApplyFinal(core.Game) {}

type EventB struct {
  Data string
}

func init() {
  gob.Register(EventA{})
}
func (e EventB) ApplyFast(core.Game)  {}
func (e EventB) Apply(core.Game)      {}
func (e EventB) ApplyFinal(core.Game) {}

var network_mutex sync.Mutex

func NetworkMockSpec(c gospec.Context) {
  c.Specify("NetworkMocks can connect to eachother and send Events.", func() {
    var nm1, nm2 core.Network
    // Repeating the joining to make failing more likely if things aren't
    // synced up properly.
    for i := 0; i < 50; i++ {
      func() {
        network_mutex.Lock()
        defer network_mutex.Unlock()
        nm1 = core.NewNetworkMock()
        nm2 = core.NewNetworkMock()
        on_ping := func(data []byte) ([]byte, error) {
          return []byte("Join us! " + string(data)), nil
        }
        on_join := func([]byte) ([]byte, error) {
          return []byte("You've joined us!"), nil
        }
        nm1.Host(on_ping, on_join)
        rhs := nm2.Ping([]byte("Monkeys"))
        c.Assume(len(rhs), Equals, 1)
        c.Specify("Can connect", func() {
          c.Expect(string(rhs[0].Data()), Equals, "Join us! Monkeys")
          res, err := nm2.Join(rhs[0], []byte("woo!"))
          fmt.Printf("On Join: res(%s), err(%v)\n", res, err)
          c.Expect(string(res), Equals, "You've joined us!")
          c.Assume(err, Equals, error(nil))
          c.Specify("Can send data", func() {
            var eb1 core.EventBatch
            eb1.Opaque_data = 123
            eb1.Event = EventA{555}
            eb2 := eb1
            go nm2.Send(eb1)
            eb3 := <-nm1.Receive()
            c.Expect(eb3, Equals, eb2)
          })
        })
        fmt.Printf("Shutdown...\n")
        nm1.Shutdown()
        nm2.Shutdown()
        rhs = nm2.Ping([]byte("Monkeys"))
        fmt.Printf("Shutdown complete: %d\n", len(rhs))
        c.Expect(len(rhs), Equals, 0)
      }()
    }
  })

  c.Specify("NetworkMocks can differentiate between multiple hosts.", func() {
    network_mutex.Lock()
    defer network_mutex.Unlock()
    N := 1000
    hosts := make([]core.Network, N)
    pad := func(n int, length int) string {
      s := fmt.Sprintf("%d", n)
      for len(s) < length {
        s = "0" + s
      }
      return s
    }
    for i := range hosts {
      hosts[i] = core.NewNetworkMock()
      defer hosts[i].Shutdown()
      num := i
      on_ping := func(data []byte) ([]byte, error) {
        return []byte(fmt.Sprintf("Ping(%s)", pad(num, 10))), nil
      }
      on_join := func([]byte) ([]byte, error) {
        return []byte(fmt.Sprintf("Join(%s)", pad(num, 10))), nil
      }
      hosts[i].Host(on_ping, on_join)
    }
    rhs := hosts[0].Ping([]byte("Waffle"))
    c.Expect(len(rhs), Equals, N)
    var resps []string
    for i := range rhs {
      resps = append(resps, string(rhs[i].Data()))
    }
    sort.Strings(resps)
    for i := range resps {
      c.Expect(resps[i], Equals, fmt.Sprintf("Ping(%s)", pad(i, 10)))
    }
    res, err := hosts[0].Join(rhs[4], []byte("Pancake"))
    c.Expect(err, Equals, error(nil))
    c.Expect(string(res), Equals, fmt.Sprintf("Join(%s)", pad(4, 10)))
  })
}
