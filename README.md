# Duramap

[![GoDoc](https://godoc.org/github.com/notduncansmith/duramap?status.svg)](https://godoc.org/github.com/notduncansmith/duramap)

Duramap wraps a `map[string]interface{}` with the safety of a [`sync.RWMutex`](https://golang.org/pkg/sync/#RWMutex) and the durability of [`bbolt`](https://github.com/etcd-io/bbolt). It is intended to be a reliable store of mutable, frequently-read, seldom-written K/V data.

In my own [unscientific](https://github.com/notduncansmith/duramap/blob/master/duramap.go#L43) [benchmarking](https://github.com/notduncansmith/duramap/blob/master/duramap.go#L68), it appears to shave about 22ms off the cost of accessing map items:

```sh
goos: darwin
goarch: amd64
pkg: github.com/notduncansmith/duramap
BenchmarkReadsDuramap-12    	      33	3638635082 ns/op
BenchmarkReadsBbolt-12      	      31	3874762095 ns/op
PASS
ok  	github.com/notduncansmith/duramap	240.373s
```

## Usage

See [GoDoc](https://godoc.org/github.com/notduncansmith/duramap) for full docs. Example:

```go
package duramap

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
)

func TestRoundtrip(t *testing.T) {
	dm, err := NewDuramap("./fixtures/test.db", "test")

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