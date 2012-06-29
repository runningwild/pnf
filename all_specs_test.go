package pnf_test

import (
  "github.com/orfjackal/gospec/src/gospec"
  "testing"
)

func TestAllSpecs(t *testing.T) {
  r := gospec.NewRunner()
  r.AddSpec(NetworkMockSpec)
  gospec.MainGoTest(r, t)
}
