package core_test

import (
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
  "github.com/runningwild/core"
)

func AuditorSpec(c gospec.Context) {
  c.Specify("Auditor stuff.", func() {
    a := new(core.Auditor)
    c.Expect(a, Not(Equals), (*core.Auditor)(nil))
  })
}
