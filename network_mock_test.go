package pnf_test

import (
  "encoding/gob"
  "runningwild/pnf"
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
)

type EventA struct {
  Data int
}
func init() {
  gob.Register(EventA{})
}
func (e EventA) ApplyFast(pnf.Game) {}
func (e EventA) Apply(pnf.Game) {}
func (e EventA) ApplyFinal(pnf.Game) {}

type EventB struct {
  Data string
}
func init() {
  gob.Register(EventA{})
}
func (e EventB) ApplyFast(pnf.Game) {}
func (e EventB) Apply(pnf.Game) {}
func (e EventB) ApplyFinal(pnf.Game) {}


func NetworkMockSpec(c gospec.Context) {
  nm1 := pnf.NewNetworkMock()
  nm2 := pnf.NewNetworkMock()
  on_ping := func(data []byte) ([]byte, error) {
    return []byte("Join us! " + string(data)), nil
  }
  on_join := func([]byte) ([]byte, error) {
    return []byte("You've joined us!"), nil
  }
  c.Specify("Actions are loaded properly.", func() {
    nm1.Host([]byte("FUDGE!!"), on_ping, on_join)
    rhs := nm2.Ping([]byte("Monkeys"))
    c.Expect(len(rhs), Equals, 1)
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
}
