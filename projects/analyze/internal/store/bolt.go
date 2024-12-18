package store

import (
	"bytes"
	"github.com/boltdb/bolt"
	"time"
)

type Wrapper struct {
	DB *bolt.DB `json:"db,omitempty"`
}

// New database and create buckets
func (r *Wrapper) New() (*Wrapper, error) {

	db, err := bolt.Open("./data.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	var buckets = []string{"posts", "ttl"}

	for _, k := range buckets {

		if err = db.Update(func(tx *bolt.Tx) error {
			_, err = tx.CreateBucketIfNotExists([]byte(k))
			if err != nil {
				return err
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	return &Wrapper{
		DB: db,
	}, nil
}

// Write to bucket with key and value
func (r *Wrapper) Write(bucket, key string, value []byte) error {
	if err := r.DB.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return bk.Put([]byte(key), value)
	}); err != nil {
		return err
	}
	return nil
}

// Read from bucket with key
func (r *Wrapper) Read(bucket, key string) ([]byte, error) {
	var result []byte
	if err := r.DB.Update(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(bucket))
		result = bk.Get([]byte(key))
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// Delete key from bucket
func (r *Wrapper) Delete(bucket, key string) error {
	if err := r.DB.Update(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(bucket))
		if err := bk.Delete([]byte(key)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (r *Wrapper) Sweep(maxAge time.Duration) error {
	var (
		keys [][]byte
		err  error
	)
	if keys, err = r.GetExpired(maxAge); err != nil || len(keys) == 0 {
		return err
	}
	return r.DB.Update(func(tx *bolt.Tx) (err error) {
		b := tx.Bucket([]byte("posts"))

		for _, key := range keys {
			if err = b.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Wrapper) GetExpired(maxAge time.Duration) ([][]byte, error) {
	var (
		keys    [][]byte
		ttlKeys [][]byte
	)
	err := r.DB.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("ttl")).Cursor()

		maxB := []byte(time.Now().UTC().Add(-maxAge).Format(time.RFC3339Nano))

		for k, v := c.First(); k != nil && bytes.Compare(k, maxB) <= 0; k, v = c.Next() {
			keys = append(keys, v)
			ttlKeys = append(ttlKeys, k)
		}
		return nil
	})
	err = r.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("ttl"))
		for _, key := range ttlKeys {
			if err = b.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
	return keys, nil
}

// Close database
func (r *Wrapper) Close() error {
	if err := r.DB.Close(); err != nil {
		return err
	}
	return nil
}
