package duramap

import (
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"os"
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

func withDM(name string, t *testing.T, encrypted bool, fn func(dm *Duramap)) {
	os.Remove("./fixtures/" + name + ".db")
	defer os.Remove("./fixtures/" + name + ".db")
	key, err := generateEncryptionSecret()

	if !encrypted {
		key = nil
		err = nil
	}

	if err != nil {
		t.Errorf("Unable to generate encryption key: %v", err)
		return
	}

	dm, err := NewDuramap("./fixtures/"+name+".db", name, key)

	if err != nil {
		t.Errorf("Should be able to open database: %v", err)
	}

	if err := dm.Load(); err != nil {
		t.Errorf("Should be able to load map: %v", err)
		return
	}

	fn(dm)
}

func updateMap(t *testing.T, dm *Duramap, fn func(tx *Tx) error) {
	err := dm.UpdateMap(fn)

	if err != nil {
		t.Errorf("Should be able to save value: %v", err)
	}
}

func expectMap(t *testing.T, dm *Duramap, gm GenericMap) {
	m := dm.WithMap(func(m GenericMap) interface{} {
		if m == nil {
			t.Errorf("Expected map %v, got nil", gm)
		}
		return m
	}).(GenericMap)

	for k, v := range gm {
		if m[k] != v {
			t.Errorf("Expected m[%v] == %v, got %v", k, gm[k], v)
		}
	}

	dm.DoWithMap(func(m GenericMap) {
		for k := range m {
			if gm[k] == nil {
				t.Errorf("Unexpected key %v", k)
			}
		}
	})
}

func TestRoundtrip(t *testing.T) {
	withDM("roundtrip", t, false, func(dm *Duramap) {
		updateMap(t, dm, func(tx *Tx) error {
			tx.Set(foo, bar)
			if tx.Get(foo) != bar {
				t.Errorf("Should be able to read saved value %v %v", tx, bar)
			}
			return nil
		})
		updateMap(t, dm, func(tx *Tx) error {
			if tx.Get(foo) != bar {
				t.Errorf("Should be able to read saved value %v %v", *(tx.M), tx.Get(foo))
			}
			return nil
		})
		expectMap(t, dm, GenericMap{foo: bar})
		dm.Truncate()
		expectMap(t, dm, GenericMap{})
	})
}

func TestRoundtripEncrypted(t *testing.T) {
	withDM("roundtrip_encrypted", t, true, func(dm *Duramap) {
		updateMap(t, dm, func(tx *Tx) error {
			tx.Set(foo, bar)
			if tx.Get(foo) != bar {
				t.Error("Should be able to read saved value")
			}
			return nil
		})
		updateMap(t, dm, func(tx *Tx) error {
			if tx.Get(foo) != bar {
				t.Error("Should be able to read saved value")
			}
			return nil
		})
		expectMap(t, dm, GenericMap{foo: bar})
		dm.Truncate()
		expectMap(t, dm, GenericMap{})
	})
}

func TestConcurrentAccess(t *testing.T) {
	wg := sync.WaitGroup{}
	key, err := generateEncryptionSecret()
	if err != nil {
		t.Errorf("Unable to generate encryption secret: %v", err)
	}
	defer os.Remove("./fixtures/concurrent_access.db")

	access := func(i int, w *sync.WaitGroup) {
		defer w.Done()

		dm, err := NewDuramap("./fixtures/concurrent_access.db", "concurrent", key)
		if err != nil {
			t.Errorf("Should be able to open database: %v", err)
			return
		}
		if err = dm.Load(); err != nil {
			t.Errorf("Should be able to load map: %v", err)
			return
		}
		for n := 0; n < 9; n++ {
			err := dm.UpdateMap(func(tx *Tx) error {
				k := fmt.Sprintf("%v-%v", i, n)
				tx.Set(k, n)
				return nil
			})
			if err != nil {
				t.Errorf("Should be able to save value: %v", err)
				return
			}
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)))
		}
		for n := 0; n < 9; n++ {
			dm.DoWithMap(func(m GenericMap) {
				k := fmt.Sprintf("%v-%v", i, n)
				if m[k] != n {
					t.Errorf("Should be able to read saved value")
				}
			})
		}
	}

	for m := 0; m < 3; m++ {
		wg.Add(1)
		go access(m, &wg)
	}

	wg.Wait()
}

func TestLoadEmpty(t *testing.T) {
	withDM("load_empty", t, true, func(dm *Duramap) {
		expectMap(t, dm, GenericMap{})
	})
}

func TestLoadRepeated(t *testing.T) {
	defer os.Remove("./fixtures/repeated.db")
	_, err := NewDuramap("./fixtures/repeated.db", "repeated", nil)
	if err != nil {
		t.Errorf("Should be able to open database: %v", err)
	}

	_, err = NewDuramap("./fixtures/repeated.db", "repeated", nil)

	if err != nil {
		t.Errorf("Should be able to open database: %v", err)
	}
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
	dm, err := NewDuramap("./fixtures/bench_duramap_reads.db", "test_reads", nil)
	defer dm.Truncate()

	if err != nil {
		b.Errorf("Should be able to open database: %v", err)
	}

	err = dm.UpdateMap(func(tx *Tx) error {
		tx.Set(foo, bar)
		for i := 0; i < 10000; i++ {
			tx.Set("thing-"+fmt.Sprintf("%v", i), mapContents)
		}
		return nil
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
	dm, err := NewDuramap("./fixtures/bench_duramap_writes.db", "test_writes", nil)
	defer dm.Truncate()

	if err != nil {
		b.Errorf("Should be able to open database: %v", err)
	}

	err = dm.UpdateMap(func(tx *Tx) error {
		tx.Set(foo, bar)
		for i := 0; i < 10000; i++ {
			tx.Set("thing-"+fmt.Sprintf("%v", i), mapContents)
		}
		return nil
	})

	if err != nil {
		b.Errorf("Should be able to save value: %v", err)
	}

	b.Run(label, func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			err = dm.UpdateMap(func(tx *Tx) error {
				tx.Set(foo, baz)
				if tx.Get(foo) != baz {
					b.Error("Should be able to read saved value")
				}
				return nil
			})

			if err != nil {
				b.Errorf("Should be able to save value: %v", err)
			}
		}
	})
}

func BenchmarkReadsBbolt(b *testing.B) {
	db, err := bolt.Open("./fixtures/bench_bbolt.db", 0600, nil)
	bucketName := []byte("bbolt")
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
	bucketName := []byte("bbolt")
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

func generateEncryptionSecret() (EncryptionSecret, error) {
	var secretKey [32]byte
	if _, err := io.ReadFull(crand.Reader, secretKey[:]); err != nil {
		return nil, err
	}
	return &secretKey, nil
}
