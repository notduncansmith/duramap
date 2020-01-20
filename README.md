# Duramap

[![GoDoc](https://godoc.org/github.com/notduncansmith/duramap?status.svg)](https://godoc.org/github.com/notduncansmith/duramap) [![Build Status](https://travis-ci.com/notduncansmith/duramap.svg?branch=master)](https://travis-ci.com/notduncansmith/duramap) [![codecov](https://codecov.io/gh/notduncansmith/duramap/branch/master/graph/badge.svg)](https://codecov.io/gh/notduncansmith/duramap)

Duramap wraps the speed of a `map[string]interface{}` with the safety of a [`sync.RWMutex`](https://golang.org/pkg/sync/#RWMutex) and the durability of [`bbolt`](https://github.com/etcd-io/bbolt) (effectively, it is an always-fully-loaded write-through cache). It is intended to be a reliable thread-safe store of mutable data with fast read requirements.

The internal map cuts most of the cost (~18ms on my machine) of accessing K/V items through BoltDB directly, while serialization of map values adds minimal overhead to writes (values are encoded with [`vmihailenco/msgpack`](https://github.com/vmihailenco/msgpack)). See Benchmarks below for more details.

### New and improved!

The old Duramap used to serialize the entire map contents with MsgPack and store this in a single key, with mutation happening directly to the map passed to `UpdateMap`. Now, the new Duramap now serializes individual key values and writes each to their own corresponding key in bbolt, saving drastically on write overhead for non-tiny maps. Updates now happen through a new `Tx` struct.

## Usage

See [GoDoc](https://godoc.org/github.com/notduncansmith/duramap) for full docs. Example:

```go
package mypackage

import (
	"testing"

	"github.com/notduncansmith/duramap"
)

func TestRoundtrip(t *testing.T) {
	dm, err := duramap.NewDuramap("./example.db", "example")

	if err != nil {
		t.Errorf("Should be able to open database: %v", err)
		return
	}

	if err = dm.Load(); err != nil {
		t.Errorf("Should be able to load map: %v", err)
		return
	}

	err = dm.UpdateMap(func(tx *Tx) error {
		tx.Set("foo", "bar")
		if tx.Get("foo") != "bar" {
			t.Error("Should be able to read saved value")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Should be able to save value: %v", err)
	}

	foo := dm.WithMap(func(m GenericMap) interface{} {
		// this is the internal map, do not mutate it!
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
BenchmarkReadsDuramap/int64-12  	17755630	        57.6 ns/op
BenchmarkReadsDuramap/str64b-12 	18237058	        56.9 ns/op
BenchmarkReadsDuramap/str128b-12         	19252372	        56.2 ns/op
BenchmarkReadsDuramap/str256b-12         	20408672	        54.7 ns/op
BenchmarkWritesDuramap/int64-12          	      63	  19229244 ns/op
BenchmarkWritesDuramap/str64b-12         	      66	  18821763 ns/op
BenchmarkWritesDuramap/str128b-12        	      63	  19051516 ns/op
BenchmarkWritesDuramap/str256b-12        	      63	  18178327 ns/op
BenchmarkReadsBbolt/str256b-12           	      64	  18297425 ns/op
BenchmarkWritesBbolt/str256b-12          	      63	  19110888 ns/op
BenchmarkReadsRWMap/rwmutex-12           	75495019	        15.7 ns/op
BenchmarkWritesRWMap/rwmutex-12          	34192036	        34.3 ns/op
BenchmarkReadsMutableMap/mutable-12      	29584789	        39.0 ns/op
BenchmarkWritesMutableMap/mutable-12     	21703112	        54.3 ns/op
```

## License

Released under [The MIT License](https://opensource.org/licenses/MIT) (see `LICENSE.txt`).

Copyright 2019 Duncan Smith