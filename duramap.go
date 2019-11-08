package duramap

import (
	"bytes"
	"sync"

	"github.com/notduncansmith/mutable"
	mp "github.com/vmihailenco/msgpack"
	bolt "go.etcd.io/bbolt"
)

// GenericMap is a map of strings to empty interfaces
type GenericMap = map[string]interface{}

// Duramap is a map of strings to ambiguous data structures, which is protected by a Read/Write mutex and backed by a bbolt database
type Duramap struct {
	mut  *sync.RWMutex
	db   *bolt.DB
	d    *mp.Decoder
	bz   []byte
	m    GenericMap
	Path string
	Name string
}

var dms = map[string]*Duramap{}
var dmsRW = mutable.NewRW("dms")
var bucketName = []byte("dms")

// NewDuramap returns a new Duramap backed by a bbolt database located at `path`
func NewDuramap(path, name string) (*Duramap, error) {
	dm := dmsRW.WithRLock(func() interface{} {
		return unsafeGetDM(name)
	}).(*Duramap)

	if dm != nil {
		return dm, nil
	}

	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	mut := &sync.RWMutex{}
	bz := []byte{}
	d := mp.NewDecoder(bytes.NewBuffer(bz))

	dm = &Duramap{mut, db, d, bz, GenericMap{}, path, name}
	dmsRW.DoWithRWLock(func() {
		unsafeAddDM(dm)
	})
	return dm, nil
}

// Load reads the stored Duramap value and sets it
func (dm *Duramap) Load() error {
	dm.mut.RLock()
	if dm.m != nil {
		dm.mut.RUnlock()
		return nil
	}
	dm.mut.RUnlock()

	return dm.db.Update(func(tx *bolt.Tx) error {
		m := GenericMap{}
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}

		bz := b.Get([]byte(dm.Name))
		if bz == nil {
			bz, err = mp.Marshal(m)
			if err != nil {
				return err
			}
			defer dm.mut.Unlock()
			dm.mut.Lock()
			err = b.Put([]byte(dm.Name), bz)
			if err != nil {
				return err
			}
			dm.unsafeSet(m, bz)
		}

		mperr := mp.Unmarshal(bz, &m)
		if mperr != nil {
			return mperr
		}

		defer dm.mut.Unlock()
		dm.mut.Lock()
		dm.unsafeSet(m, bz)

		return nil
	})
}

// UpdateMap is like `DoWithMap` but the result of `f` is re-saved afterwards
func (dm *Duramap) UpdateMap(f func(m GenericMap) GenericMap) error {
	defer dm.mut.Unlock()
	dm.mut.Lock()
	if err := dm.unsafeSetMap(f(dm.m)); err != nil {
		return err
	}
	return dm.unsafeStoreBytes()
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

// UpdateBytes is like `UpdateMap` but with bytes
func (dm *Duramap) UpdateBytes(f func(bz []byte) []byte) error {
	defer dm.mut.Unlock()
	dm.mut.Lock()
	if err := dm.unsafeSetBytes(f(dm.bz)); err != nil {
		return err
	}

	return dm.unsafeStoreBytes()
}

// WithBytes returns the result of calling `f` with the internal bytes
func (dm *Duramap) WithBytes(f func(bz []byte) interface{}) interface{} {
	defer dm.mut.RUnlock()
	dm.mut.RLock()
	return f(dm.bz)
}

// DoWithBytes is like `WithBytes` but does not return a result
func (dm *Duramap) DoWithBytes(f func(bz []byte)) {
	defer dm.mut.RUnlock()
	dm.mut.RLock()
	f(dm.bz)
}

// Close will close the connection to the database and remove the Duramap from the list of Duramaps
func (dm *Duramap) Close() error {
	err := dmsRW.WithRWLock(func() interface{} {
		if err := dm.db.Close(); err != nil {
			return err
		}
		unsafeRemoveDM(dm.Name)
		return nil
	})
	if err != nil {
		return err.(error)
	}
	return nil
}

// Truncate will reset the contents of the Duramap to an empty GenericMap
func (dm *Duramap) Truncate() error {
	return dm.UpdateMap(func(m GenericMap) GenericMap {
		return GenericMap{}
	})
}

// DoWithQuery extracts values matching a path `q` from the map and calls `f` with the results
func (dm *Duramap) DoWithQuery(q string, f func([]interface{})) error {
	defer dm.mut.RUnlock()
	dm.mut.RLock()
	results, err := dm.d.Query(q)
	if err != nil {
		return err
	}
	f(results)
	return nil
}

func (dm *Duramap) unsafeSet(m GenericMap, bz []byte) {
	dm.bz = bz
	dm.m = m
	dm.d = mp.NewDecoder(bytes.NewBuffer(dm.bz))
}

func (dm *Duramap) unsafeSetMap(m GenericMap) error {
	bz, err := mp.Marshal(m)

	if err != nil {
		return err
	}
	dm.unsafeSet(m, bz)
	return nil
}

func (dm *Duramap) unsafeSetBytes(bz []byte) (err error) {
	dm.m = GenericMap{}
	err = mp.Unmarshal(bz, &dm.m)

	if err != nil {
		return err
	}

	dm.unsafeSet(dm.m, bz)

	return nil
}

func (dm *Duramap) unsafeStoreMap() error {
	var err error
	dm.bz, err = mp.Marshal(dm.m)
	if err != nil {
		return err
	}
	return dm.unsafeStoreBytes()
}

func (dm *Duramap) unsafeStoreBytes() error {
	return dm.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		b.Put([]byte(dm.Name), dm.bz)
		return err
	})
}

func unsafeAddDM(dm *Duramap) {
	if dms[dm.Name] == nil {
		dms[dm.Name] = dm
	}
}

func unsafeGetDM(name string) *Duramap {
	return dms[name]
}

func unsafeRemoveDM(name string) {
	dms[name] = nil
}
