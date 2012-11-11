package core_test

import (
  "encoding/gob"
  "github.com/orfjackal/gospec/src/gospec"
  "testing"
)

type EventA struct {
  Data int
}

func init() {
  gob.Register(EventA{})
}
func (e EventA) ApplyFirst(interface{}) {}
func (e EventA) Apply(g interface{}) {
  g.(*TestGame).A += e.Data
}
func (e EventA) ApplyFinal(interface{}) {}

type EventB struct {
  Data string
}

func init() {
  gob.Register(EventB{})
}
func (e EventB) ApplyFirst(interface{}) {}
func (e EventB) Apply(g interface{}) {
  g.(*TestGame).B = e.Data
}
func (e EventB) ApplyFinal(interface{}) {}

func init() {
  gob.Register(&TestGame{})
}

type TestGame struct {
  A      int
  B      string
  Thinks int
}

func (g *TestGame) ThinkFirst() {}
func (g *TestGame) ThinkFinal() {}
func (g *TestGame) Think() {
  g.Thinks++
}
func (g *TestGame) Copy() interface{} {
  g2 := *g
  return &g2
}
func (g *TestGame) OverwriteWith(_g2 interface{}) {
  g2 := _g2.(*TestGame)
  if g == g2 {
    panic("SHIT")
  }
  *g = *g2
}

func TestAllSpecs(t *testing.T) {
  r := gospec.NewRunner()
  r.AddSpec(NetworkMockSpec)
  r.AddSpec(NetworkStandardSpec)
  r.AddSpec(NetworkStandardGobbingSpec)
  r.AddSpec(EngineUdpTcpSpec)
  r.AddSpec(BundlerSpec)
  r.AddSpec(UpdaterSpec)
  r.AddSpec(CommunicatorSpec)
  r.AddSpec(AuditorSpec)
  r.AddSpec(BaseSpec)
  r.AddSpec(EngineSpec)
  gospec.MainGoTest(r, t)
}
