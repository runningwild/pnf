package container_test

import (
  "fmt"
  "runningwild/pnf/containers/window/uint64_string"
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
)

func Uint64StringWindowSpec(c gospec.Context) {
  var start uint64 = 1234
  var size uint64 = 10
  w := window.New(int(size), start)
  c.Specify("Basic getting and setting.", func() {
    for pos := w.Start(); pos < w.End(); pos++ {
      w.Set(pos, fmt.Sprintf("%d", pos))
    }
    c.Expect(w.End()-w.Start(), Equals, size)
    for pos := w.Start(); pos < w.End(); pos++ {
      c.Expect(w.Get(pos), Equals, fmt.Sprintf("%d", pos))
    }
  })
  c.Specify("Advancing", func() {
    for pos := w.Start(); pos < w.End(); pos++ {
      w.Set(pos, fmt.Sprintf("%d", pos))
    }
    w.Advance()
    w.Advance()
    w.Advance()
    c.Expect(w.Start(), Equals, start+3)
    c.Expect(w.End(), Equals, start+size+3)
    c.Expect(w.Get(w.Start()), Equals, fmt.Sprintf("%d", w.Start()))
  })
  c.Specify("Should panic when accessing out of bound elements.", func() {
    defer func() {
      recover()
    }()
    w.Get(start - 1)
    c.Expect("Failed to panic.", Equals, false)
  })
}
