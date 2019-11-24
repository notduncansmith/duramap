# Duramap

[![GoDoc](https://godoc.org/github.com/notduncansmith/duramap?status.svg)](https://godoc.org/github.com/notduncansmith/duramap) [![Build Status](https://travis-ci.com/notduncansmith/duramap.svg?branch=master)](https://travis-ci.com/notduncansmith/duramap) [![codecov](https://codecov.io/gh/notduncansmith/duramap/branch/master/graph/badge.svg)](https://codecov.io/gh/notduncansmith/duramap)

Duramap wraps the speed of a `map[string]interface{}` with the safety of a [`sync.RWMutex`](https://golang.org/pkg/sync/#RWMutex) and the durability of [`bbolt`](https://github.com/etcd-io/bbolt). It is intended to be a reliable thread-safe store of mutable, frequently-read, seldom-written K/V data (such as configuration that may be accessed on the hot path of an application).

The internal map cuts most of the cost (~18ms on my machine) of accessing K/V items through BoltDB directly, while the serialization of that map adds overhead to writes in line with map size (~0.5ms with a near-empty map, ~10ms with 10,000 map keys and 256-byte string values). See Benchmarks below for more details.

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

	if err = dm.Load(); err != nil {
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

## Benchmarks

```sh
goos: darwin
goarch: amd64
pkg: github.com/notduncansmith/duramap
BenchmarkReadsDuramap/int64-12  	23945770	        49.8 ns/op
BenchmarkReadsDuramap/str64b-12 	20652534	        50.1 ns/op
BenchmarkReadsDuramap/str128b-12         	21129648	        50.0 ns/op
BenchmarkReadsDuramap/str256b-12         	21397312	        51.0 ns/op
BenchmarkWritesDuramap/int64-12          	      62	  20138961 ns/op
BenchmarkWritesDuramap/str64b-12         	      56	  22581307 ns/op
BenchmarkWritesDuramap/str128b-12        	      52	  23350175 ns/op
BenchmarkWritesDuramap/str256b-12        	      42	  28058637 ns/op
BenchmarkReadsBbolt/str256b-12           	      66	  18385341 ns/op
BenchmarkWritesBbolt/str256b-12          	      64	  18086188 ns/op
BenchmarkReadsRWMap/rwmutex-12           	74938060	        15.8 ns/op
BenchmarkWritesRWMap/rwmutex-12          	33574027	        34.9 ns/op
BenchmarkReadsMutableMap/mutable-12      	31702456	        36.6 ns/op
BenchmarkWritesMutableMap/mutable-12     	22621276	        52.1 ns/op
```

## License

Released under [The MIT License](https://opensource.org/licenses/MIT) (see `LICENSE.txt`).

Copyright 2019 Duncan Smith