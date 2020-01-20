package duramap

import (
	"sync"

	"github.com/notduncansmith/mutable"
	mp "github.com/vmihailenco/msgpack"
	bolt "go.etcd.io/bbolt"
)

// GenericMap is a map of strings to empty interfaces
type GenericMap = map[string]interface{}

// Tx is a transaction that operates on a Duramap. It has a reference to the internal map.
type Tx struct {
	M      *GenericMap
	writes GenericMap
}

// Get returns the latest value either written in the Tx or stored in the map
func (tx *Tx) Get(k string) interface{} {
	written := tx.writes[k]
	if written != nil {
		return written
	}
	return (*tx.M)[k]
}

// Set writes a new value in the Tx
func (tx *Tx) Set(k string, v interface{}) {
	tx.writes[k] = v
}

// Duramap is a map of strings to ambiguous data structures, which is protected by a Read/Write mutex and backed by a bbolt database
type Duramap struct {
	mut  *sync.RWMutex
	db   *bolt.DB
	m    GenericMap
	Path string
	Name string
}

var dbs = map[string]*bolt.DB{}
var dbsRW = mutable.NewRW("dbs")

var dms = map[string]*Duramap{}
var dmsRW = mutable.NewRW("dms")

// NewDuramap returns a new Duramap backed by a bbolt database located at `path`
func NewDuramap(path, name string) (*Duramap, error) {
	dm := dmsRW.WithRLock(func() interface{} {
		return unsafeGetDM(path + ":" + name)
	}).(*Duramap)

	if dm != nil {
		return dm, nil
	}

	var err error
	var db *bolt.DB
	dbsRW.DoWithRWLock(func() {
		if dbs[path] != nil {
			db = dbs[path]
			return
		}
		db, err = bolt.Open(path, 0600, nil)
		if err == nil {
			dbs[path] = db
		}
	})

	if err != nil {
		return nil, err
	}

	mut := &sync.RWMutex{}

	dm = &Duramap{mut, db, GenericMap{}, path, name}
	dmsRW.DoWithRWLock(func() {
		unsafeAddDM(dm)
	})
	return dm, nil
}

// Load reads the stored Duramap value and sets it
func (dm *Duramap) Load() error {
	dm.db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte(dm.Name))
		return nil
	})

	return dm.db.View(func(tx *bolt.Tx) error {
		m := GenericMap{}
		b := tx.Bucket([]byte(dm.Name))
		c := b.Cursor()

		for k, bz := c.First(); k != nil; k, bz = c.Next() {
			v := map[string]interface{}{}
			if bz == nil || len(bz) == 0 {
				continue
			}
			if err := mp.Unmarshal(bz, &v); err != nil {
				return err
			}
			m[string(k)] = v["."]
		}

		dm.mut.Lock()
		defer dm.mut.Unlock()
		dm.m = m

		return nil
	})
}

// UpdateMap is like `DoWithMap` but the result of `f` is re-saved afterwards
func (dm *Duramap) UpdateMap(f func(tx *Tx) error) error {
	defer dm.mut.Unlock()
	dm.mut.Lock()
	tx := dm.tx()
	if err := f(tx); err != nil {
		return err
	}
	bzWrites := map[string][]byte{}
	for k, v := range tx.writes {
		bz, err := mp.Marshal(map[string]interface{}{".": v})
		if err != nil {
			return err
		}
		bzWrites[k] = bz
	}
	err := dm.db.Update(func(btx *bolt.Tx) error {
		for k, v := range bzWrites {
			b, _ := btx.CreateBucketIfNotExists([]byte(dm.Name))
			if err := b.Put([]byte(k), v); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	for k, v := range tx.writes {
		dm.m[k] = v
	}

	return nil
}

// WithMap returns the result of calling `f` with the internal map
func (dm *Duramap) WithMap(f func(m GenericMap) interface{}) interface{} {
	defer dm.mut.RUnlock()
	dm.mut.RLock()
	return f(dm.m)
}

// DoWithMap is like `WithMap` but does not return a result
func (dm *Duramap) DoWithMap(f func(m GenericMap)) {
	defer dm.mut.RUnlock()
	dm.mut.RLock()
	f(dm.m)
}

// Truncate will reset the contents of the Duramap to an empty GenericMap
func (dm *Duramap) Truncate() error {
	dm.mut.Lock()
	defer dm.mut.Unlock()
	err := dm.db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(dm.Name))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucket([]byte(dm.Name))
		return err
	})
	if err != nil {
		return err
	}
	dm.m = GenericMap{}
	return nil
}

func (dm *Duramap) tx() *Tx {
	return &Tx{&dm.m, GenericMap{}}
}

func unsafeAddDM(dm *Duramap) {
	k := dm.Path + ":" + dm.Name
	if dms[k] == nil {
		dms[k] = dm
	}
}

func unsafeGetDM(k string) *Duramap {
	return dms[k]
}
