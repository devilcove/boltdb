package boltdb

import (
	"encoding/json"
	"errors"
	"time"

	"go.etcd.io/bbolt"
)

// Store represents a bbolt db store.
type Store struct {
	db *bbolt.DB
}

// Generic error results.
var (
	// ErrNoResults indicates query found no results.
	ErrNoResults = errors.New("no results found")
	// ErrInvalidPath indicates that specified bucket does not exist.
	ErrInvalidPath = errors.New("invalid path")
	// ErrNoConnection indicates that database is not open.
	ErrNoConnection = errors.New("no db connection")
	// ErrExists indicates that a key exists.
	ErrExists = errors.New("key existss")
)

// Initialize sets up bbolt db using file path and creates tables if required.
func Initialize(file string, buckets []Path) (*Store, error) {
	db, err := bbolt.Open(file, 0o666, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return &Store{}, err
	}
	s := &Store{db: db}
	return s, s.createBuckets(buckets)
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Connection returns the connection to the store for more advanced queries by caller.
func (s *Store) Connection() *bbolt.DB {
	return s.db
}

// Insert saves a value only if key does not exist.
func (s *Store) Insert(value any, key string, table Path) error {
	_, err := s.Get[any](key, table)
	if err == nil {
		return ErrExists
	}
	if errors.Is(err, ErrNoResults) {
		return s.Save(value, key, table)
	}
	return err
}

// Save saves a generic value under key in the specified bucket.
func (s *Store) Save(value any, key string, path Path) error {
	marshalled, err := json.Marshal(&value)
	if err != nil {
		return err
	}
	return s.SaveRaw(marshalled, []byte(key), path)
}

// SaveRaw saves a byte value in a bucket.
func (s *Store) SaveRaw(value, key []byte, path Path) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := getBucket(tx, path)
		if b == nil {
			return ErrInvalidPath
		}
		return b.Put(key, value)
	})
}

// Tables returns array of table names.
func (s *Store) Tables() []string {
	tables := []string{}
	s.db.View(func(tx *bbolt.Tx) error { //nolint:errcheck,gosec
		bucket := tx.Inspect()
		for _, v := range bucket.Children {
			tables = append(tables, v.Name)
		}
		return nil
	})
	return tables
}

// Update save a value only if key already exists.
func (s *Store) Update(value any, key string, bucket Path) error {
	_, err := s.Get[any](key, bucket)
	if err == nil {
		return s.Save(value, key, bucket)
	}
	return ErrExists
}

// Get retrieves a value for key in specified bucket.
func (s *Store) Get[T any](key string, path Path) (T, error) {
	var value T
	bytes, err := s.GetRaw([]byte(key), path)
	if err != nil {
		return value, err
	}

	err = json.Unmarshal(bytes, &value)
	return value, err
}

// GetRaw retrieves a value for key in a specified bucket.
func (s *Store) GetRaw(key []byte, path Path) ([]byte, error) {
	value := []byte{}
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := getBucket(tx, path)
		if bucket == nil {
			return nil
		}
		value = bucket.Get(key)
		if value == nil {
			return ErrNoResults
		}
		return nil
	})
	return value, err
}

// GetAll retrieves all values from bucket.
func (s *Store) GetAll[T any](path Path) ([]T, error) {
	var values []T
	var value T
	raw, err := s.GetAllRaw(path)
	if err != nil {
		return values, err
	}
	for _, r := range raw {
		if err := json.Unmarshal(r, &value); err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (s *Store) GetAllRaw(path Path) ([][]byte, error) {
	var result [][]byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := getBucket(tx, path)
		if b == nil {
			return ErrInvalidPath
		}
		_ = b.ForEach(func(k, v []byte) error {
			result = append(result, v)
			return nil
		})
		return nil
	})
	return result, err
}

// Delete deletes a key in a bucket.
func (s *Store) Delete[T any](key string, path Path) error {
	k := []byte(key)
	// verify key exists

	if _, err := s.GetRaw(k, path); err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := getBucket(tx, path)
		// no need to check for nil bucket as GetRow above would err
		return b.Delete(k)
	})
}

func (s *Store) createBucket(name Path) error {
	path := toDBPath(name)
	if len(path) == 0 {
		return ErrInvalidPath
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(path[0])
		if err != nil {
			return err
		}
		for _, p := range path[1:] {
			b, err = b.CreateBucketIfNotExists(p)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) createBuckets(tables []Path) error {
	var errs error
	for _, table := range tables {
		if err := s.createBucket(table); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

func getBucket(tx *bbolt.Tx, path Path) *bbolt.Bucket {
	if len(path) == 0 {
		return nil
	}
	dbpath := toDBPath(path)
	bucket := tx.Bucket(dbpath[0])
	if bucket == nil {
		return bucket
	}
	for _, p := range dbpath[1:] {
		bucket = bucket.Bucket(p)
		if bucket == nil {
			return bucket
		}
	}
	return bucket
}
