package duramap

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/notduncansmith/mutable"
	bolt "go.etcd.io/bbolt"
)

const foo = "foo"
const bar = "bar"
const baz = "baz"
const str64 = "abc4567890123456789012345678901234567890123456789012345678901234"

var str128 = str64 + str64
var str256 = str128 + str128

func TestRoundtrip(t *testing.T) {
	dm, err := NewDuramap("./fixtures/test.db", "test")
	defer dm.Close()

	if err != nil {
		t.Errorf("Should be able to open database: %v", err)
		return
	}

	if err = dm.Load(); err != nil {
		t.Errorf("Should be able to load map: %v", err)
		return
	}

	err = dm.UpdateMap(func(m GenericMap) GenericMap {
		m[foo] = bar
		return m
	})

	if err != nil {
		t.Errorf("Should be able to save value: %v", err)
	}

	foo := dm.WithMap(func(m GenericMap) interface{} {
		return m[foo]
	}).(string)

	if foo != bar {
		t.Error("Should be able to read saved value")
		return
	}

	dm.Truncate()
	dm.DoWithMap(func(m GenericMap) {
		if m[foo] != nil {
			t.Error("Should not be able to read saved value after .Truncate()")
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	wg := sync.WaitGroup{}
	access := func(i int) {
		defer wg.Done()
		wg.Add(1)
		dm, err := NewDuramap("./fixtures/concurrent_access.db", "concurrent")
		if err != nil {
			t.Errorf("Should be able to open database: %v", err)
		}
		defer dm.Close()
		if err = dm.Load(); err != nil {
			t.Errorf("Should be able to load map: %v", err)
			return
		}
		for n := 0; n < 10; n++ {
			err := dm.UpdateMap(func(m GenericMap) GenericMap {
				k := fmt.Sprintf("%v-%v", i, n)
				m[k] = n
				return m
			})
			if err != nil {
				t.Errorf("Unable to save value: %v", err)
			}
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
		}
		for n := 0; n < 10; n++ {
			dm.DoWithMap(func(m GenericMap) {
				k := fmt.Sprintf("%v-%v", i, n)
				if m[k] != n {
					t.Errorf("Should be able to read saved value")
				}
			})
		}
	}

	for m := 0; m < 12; m++ {
		go access(m)
	}
	wg.Wait()
}

func BenchmarkReadsDuramap(b *testing.B) {
	benchmarkReadsDuramap("int64", int64(123456789), b)
	benchmarkReadsDuramap("str64b", str64, b)
	benchmarkReadsDuramap("str128b", str128, b)
	benchmarkReadsDuramap("str256b", str256, b)
}

func BenchmarkWritesDuramap(b *testing.B) {
	benchmarkWritesDuramap("int64", int64(123456789), b)
	benchmarkWritesDuramap("str64b", str64, b)
	benchmarkWritesDuramap("str128b", str128, b)
	benchmarkWritesDuramap("str256b", str256, b)
}

func benchmarkReadsDuramap(label string, mapContents interface{}, b *testing.B) {
	dm, err := NewDuramap("./fixtures/bench_duramap_reads.db", "test_reads")
	defer dm.Truncate()

	if err != nil {
		b.Errorf("Should be able to open database: %v", err)
	}

	err = dm.UpdateMap(func(m GenericMap) GenericMap {
		m[foo] = bar
		for i := 0; i < 10000; i++ {
			m["thing-"+fmt.Sprintf("%v", i)] = mapContents
		}
		return m
	})

	if err != nil {
		b.Errorf("Should be able to save value: %v", err)
	}

	b.Run(label, func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			dm.DoWithMap(func(m GenericMap) {
				if m[foo] != bar {
					b.Error("Should be able to read saved value")
				}
			})
		}
	})
}

func benchmarkWritesDuramap(label string, mapContents interface{}, b *testing.B) {
	dm, err := NewDuramap("./fixtures/bench_duramap_writes.db", "test_writes")
	defer dm.Truncate()

	if err != nil {
		b.Errorf("Should be able to open database: %v", err)
	}

	err = dm.UpdateMap(func(m GenericMap) GenericMap {
		m[foo] = bar
		for i := 0; i < 10000; i++ {
			m["thing-"+fmt.Sprintf("%v", i)] = mapContents
		}
		return m
	})

	if err != nil {
		b.Errorf("Should be able to save value: %v", err)
	}

	b.Run(label, func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			err = dm.UpdateMap(func(m GenericMap) GenericMap {
				m[foo] = baz
				if m[foo] != baz {
					b.Error("Should be able to read saved value")
				}
				return m
			})

			if err != nil {
				b.Errorf("Should be able to save value: %v", err)
			}
		}
	})
}

func BenchmarkReadsBbolt(b *testing.B) {
	db, err := bolt.Open("./fixtures/bench_bbolt.db", 0600, nil)
	defer db.Close()
	if err != nil {
		b.Errorf("Should be able to open database: %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		for n := 0; n < 10000; n++ {
			err = bucket.Put([]byte("thing-"+fmt.Sprintf("%v", n)), []byte(str256))
		}
		return bucket.Put([]byte(foo), []byte(bar))
	})

	if err != nil {
		b.Errorf("Should be able to save value: %v", err)
	}

	b.Run("str256b", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			err = db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket(bucketName)
				val := bucket.Get([]byte(foo))
				if string(val) != bar {
					b.Errorf("Should be able to read saved value")
				}
				return nil
			})
			if err != nil {
				b.Errorf("Should be able to read saved value: %v", err)
			}
		}
	})
}

func BenchmarkWritesBbolt(b *testing.B) {
	db, err := bolt.Open("./fixtures/bench_bbolt.db", 0600, nil)
	defer db.Close()
	if err != nil {
		b.Errorf("Should be able to open database: %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		for n := 0; n < 10000; n++ {
			err = bucket.Put([]byte("thing-"+fmt.Sprintf("%v", n)), []byte(str256))
		}
		return bucket.Put([]byte(foo), []byte(bar))
	})

	if err != nil {
		b.Errorf("Should be able to save value: %v", err)
	}

	b.Run("str256b", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			err = db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket(bucketName)
				return bucket.Put([]byte(foo), []byte(baz))
			})
			if err != nil {
				b.Errorf("Should be able to read saved value: %v", err)
			}
		}
	})
}

func BenchmarkReadsRWMap(b *testing.B) {
	mut := sync.RWMutex{}
	gm := GenericMap{foo: bar}
	b.Run("rwmutex", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			mut.RLock()
			if gm[foo].(string) != bar {
				b.Error("Should be able to read saved value")
			}
			mut.RUnlock()
		}
	})
}

func BenchmarkWritesRWMap(b *testing.B) {
	mut := sync.RWMutex{}
	gm := GenericMap{foo: bar}
	b.Run("rwmutex", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			mut.Lock()
			gm[foo] = baz
			if gm[foo].(string) != baz {
				b.Error("Should be able to read saved value")
			}
			mut.Unlock()
		}
	})
}

func BenchmarkReadsMutableMap(b *testing.B) {
	m := mutable.NewRW("benchmark")
	gm := GenericMap{foo: bar}

	b.Run("mutable", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			m.DoWithRLock(func() {
				if gm[foo].(string) != bar {
					b.Error("Should be able to read saved value")
				}
			})
		}
	})
}

func BenchmarkWritesMutableMap(b *testing.B) {
	m := mutable.NewRW("benchmark")
	gm := GenericMap{foo: bar}

	b.Run("mutable", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			m.DoWithRWLock(func() {
				gm[foo] = baz
				if gm[foo].(string) != baz {
					b.Error("Should be able to read saved value")
				}
			})
		}
	})
}
