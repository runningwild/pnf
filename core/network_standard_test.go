package core_test

import (
  "github.com/orfjackal/gospec/src/gospec"
  . "github.com/orfjackal/gospec/src/gospec"
  "runningwild/pnf/core"
)

func NetworkStandardSpec(c gospec.Context) {
  c.Specify("Basic standard network functionality.", func() {
    network, err := core.MakeTcpUdpNetwork(1234)
    c.Expect(err, Equals, error(nil))
    network.Ping([]byte("FUDGECAKEMONKEYS"))
    network.Shutdown()
  })
}
