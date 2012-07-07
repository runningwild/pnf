package core_test

import (
  "encoding/gob"
  "runningwild/pnf/core"
  "github.com/orfjackal/gospec/src/gospec"
  "testing"
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

func TestAllSpecs(t *testing.T) {
  r := gospec.NewRunner()
  r.AddSpec(NetworkMockSpec)
  r.AddSpec(BundlerSpec)
  gospec.MainGoTest(r, t)
}
