package test

import (
  "io"
  "bufio"
  "testing"
)

func Writer(prefix string, t *testing.T) io.Writer {
  reader, writer := io.Pipe()

  go func(t *testing.T, reader io.Reader) {
    scanner := bufio.NewScanner(reader)
    for scanner.Scan() {
      t.Logf("%s%s", prefix, scanner.Text())
    }
    if scannererr := scanner.Err(); scannererr != nil {
      t.Error(scannererr)
    }
  }(t, reader)

  return writer
}
