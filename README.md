# Duramap

[![GoDoc](https://godoc.org/github.com/notduncansmith/duramap?status.svg)](https://godoc.org/github.com/notduncansmith/duramap)

Duramap wraps the speed of a `map[string]interface{}` with the safety of a [`sync.RWMutex`](https://golang.org/pkg/sync/#RWMutex) and the durability of [`bbolt`](https://github.com/etcd-io/bbolt). It is intended to be a reliable thread-safe store of mutable, frequently-read, seldom-written K/V data.

In my own unscientific benchmarking, the internal map appears to cut most of the cost (~18ms) of accessing K/V items through BoltDB directly, while adding some overhead (~0.6ms) to writes:

```sh
goos: darwin
goarch: amd64
pkg: github.com/notduncansmith/duramap
BenchmarkReadsDuramap/duramap-dowithmap-12      30733963	        38.8 ns/op
BenchmarkWritesDuramap/duramap-dowithmap-12           60	    18614541 ns/op
BenchmarkReadsBbolt/bbolt-12                          67	    18844352 ns/op
BenchmarkWritesBbolt/bbolt-w-12                       67	    18043206 ns/op
BenchmarkReadsRWMap/rwmutex-12                  75317965	        15.8 ns/op
BenchmarkWritesRWMap/rwmutex-12                 33796504	        34.7 ns/op
BenchmarkReadsMutableMap/mutable-12             32350231	        36.4 ns/op
BenchmarkWritesMutableMap/mutable-12            22673941	        51.8 ns/op
PASS
ok  	github.com/notduncansmith/duramap	11.857s
```

## Usage

See [GoDoc](https://godoc.org/github.com/notduncansmith/duramap) for full docs. Example:

```go
package mypackage

import (
	"testing"

	"github.com/notduncansmith/duramap"
)

func TestRoundtrip(t *testing.T) {
	dm, err := duramap.NewDuramap("./fixtures/test.db", "test")

	if err != nil {
		t.Errorf("Should be able to open database: %v", err)
		return
	}

	err = dm.Load()

	if err != nil {
		t.Errorf("Should be able to load map: %v", err)
		return
	}

	err = dm.UpdateMap(func(m GenericMap) GenericMap {
		m["foo"] = "bar"
		return m
	})

	if err != nil {
		t.Errorf("Should be able to save value: %v", err)
	}

	foo := dm.WithMap(func(m GenericMap) interface{} {
		return m["foo"]
	}).(string)

	if foo != "bar" {
		t.Error("Should be able to read saved value")
		return
	}
}
```

## License

[The MIT License](https://opensource.org/licenses/MIT):

Copyright 2019 Duncan Smith

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.