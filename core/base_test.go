package core_test

import (
  "runningwild/pnf/core"
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
)

type foo struct {
  A, B int
  Bar  struct {
    C, D     string
    Wingding struct {
      E, F float64
    }
  }
}

func BaseSpec(c gospec.Context) {
  c.Specify("QuickGob", func() {
    var f foo
    f.A = 1
    f.B = 2
    f.Bar.C = "asdf"
    f.Bar.D = "monkey"
    f.Bar.Wingding.E = 1.234
    f.Bar.Wingding.F = -343.3
    var f2 foo
    data, err := core.QuickGobEncode(f)
    c.Expect(err, Equals, error(nil))
    if err != nil {
      return
    }
    err = core.QuickGobDecode(&f2, data)
    c.Expect(err, Equals, error(nil))
    if err != nil {
      return
    }
    c.Expect(f2, Equals, f)
  })
}
