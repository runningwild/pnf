package pnf

import (
  "crypto/rand"
  "math/big"
)

// Creates a random id that will be unique among all other engines with high
// probability.
func RandomId() int64 {
  b := big.NewInt(1 << 62)
  v, err := rand.Int(rand.Reader, b)
  if err != nil {
    // uh-oh
    panic(err)
  }
  return v.Int64()
}
