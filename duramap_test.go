package duramap

import (
	"sync"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/notduncansmith/mutable"
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

func BenchmarkReadsDuramap(b *testing.B) {
	dm, err := NewDuramap("./fixtures/bench_duramap.db", "test")

	if err != nil {
		b.Errorf("Should be able to open database: %v", err)
	}

	err = dm.UpdateMap(func(m GenericMap) GenericMap {
		m["foo"] = "bar"
		return m
	})

	if err != nil {
		b.Errorf("Should be able to save value: %v", err)
	}

	b.Run("duramap-dowithmap", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			dm.DoWithMap(func(m GenericMap) {
				if m["foo"].(string) != "bar" {
					b.Error("Should be able to read saved value")
				}
			})
		}
	})
}

func BenchmarkWritesDuramap(b *testing.B) {
	dm, err := NewDuramap("./fixtures/bench_duramap.db", "test")

	if err != nil {
		b.Errorf("Should be able to open database: %v", err)
	}

	b.Run("duramap-dowithmap", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			err = dm.UpdateMap(func(m GenericMap) GenericMap {
				m["foo"] = "baz"
				if m["foo"].(string) != "baz" {
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
		bucket, err := tx.CreateBucketIfNotExists([]byte("dms"))
		if err != nil {
			return err
		}
		return bucket.Put([]byte("foo"), []byte("bar"))
	})

	if err != nil {
		b.Errorf("Should be able to save value: %v", err)
	}

	b.Run("bbolt", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			err = db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte("dms"))
				bar := bucket.Get([]byte("foo"))
				if string(bar) != "bar" {
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
		bucket, err := tx.CreateBucketIfNotExists([]byte("dms"))
		if err != nil {
			return err
		}
		return bucket.Put([]byte("foo"), []byte("bar"))
	})

	if err != nil {
		b.Errorf("Should be able to save value: %v", err)
	}

	b.Run("bbolt-w", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			err = db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte("dms"))
				return bucket.Put([]byte("foo"), []byte("baz"))
			})
			if err != nil {
				b.Errorf("Should be able to read saved value: %v", err)
			}
		}
	})
}

func BenchmarkReadsRWMap(b *testing.B) {
	mut := sync.RWMutex{}
	gm := GenericMap{"foo": "bar"}
	b.Run("rwmutex", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			mut.RLock()
			if gm["foo"].(string) != "bar" {
				b.Error("Should be able to read saved value")
			}
			mut.RUnlock()
		}
	})
}

func BenchmarkWritesRWMap(b *testing.B) {
	mut := sync.RWMutex{}
	gm := GenericMap{"foo": "bar"}
	b.Run("rwmutex", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			mut.Lock()
			gm["foo"] = "baz"
			if gm["foo"].(string) != "baz" {
				b.Error("Should be able to read saved value")
			}
			mut.Unlock()
		}
	})
}

func BenchmarkReadsMutableMap(b *testing.B) {
	m := mutable.NewRW("benchmark")
	gm := GenericMap{"foo": "bar"}

	b.Run("mutable", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			m.DoWithRLock(func() {
				if gm["foo"].(string) != "bar" {
					b.Error("Should be able to read saved value")
				}
			})
		}
	})
}

func BenchmarkWritesMutableMap(b *testing.B) {
	m := mutable.NewRW("benchmark")
	gm := GenericMap{"foo": "bar"}

	b.Run("mutable", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			m.DoWithRWLock(func() {
				gm["foo"] = "baz"
				if gm["foo"].(string) != "baz" {
					b.Error("Should be able to read saved value")
				}
			})
		}
	})
}
