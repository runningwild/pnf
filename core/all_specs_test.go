package core_test

import (
  "github.com/orfjackal/gospec/src/gospec"
  "testing"
)

func TestAllSpecs(t *testing.T) {
  r := gospec.NewRunner()
  r.AddSpec(NetworkMockSpec)
  r.AddSpec(BundlerSpec)
  gospec.MainGoTest(r, t)
}
