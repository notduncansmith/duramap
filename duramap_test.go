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

	for n := 0; n < b.N; n++ {
		dm.DoWithMap(func(m GenericMap) {
			if m["foo"].(string) != "bar" {
				b.Error("Should be able to read saved value")
			}
		})
	}
}

func BenchmarkReadsBbolt(b *testing.B) {
	db, err := bolt.Open("./fixtures/bench_bbolt.db", 0600, nil)
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

	for n := 0; n < b.N; n++ {
		db.View(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists([]byte("dms"))
			if err != nil {
				return err
			}
			bar := bucket.Get([]byte("foo"))
			if string(bar) != "bar" {
				b.Error("Should be able to read saved value")
			}
			return nil
		})
	}
}
