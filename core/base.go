package core

import (
  "bytes"
  "crypto/rand"
  "math/big"
  "encoding/gob"
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

func QuickGobEncode(a interface{}) ([]byte, error) {
  buf := bytes.NewBuffer(nil)
  err := gob.NewEncoder(buf).Encode(a)
  return buf.Bytes(), err
}

func QuickGobDecode(a interface{}, data []byte) error {
  return gob.NewDecoder(bytes.NewBuffer(data)).Decode(a)
}
