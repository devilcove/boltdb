package boltdb

import (
	"encoding/json"
	"errors"
	"time"

	"go.etcd.io/bbolt"
)

type Store struct {
	db *bbolt.DB
}

// Generic error results.
var (
	// ErrNoResults indicates query found no results.
	ErrNoResults = errors.New("no results found")
	// ErrInvalidTableName indicates that specified table does not exist.
	ErrInvalidTableName = errors.New("invalid table")
	// ErrNoConnection indicates that database is not open.
	ErrNoConnection = errors.New("no db connection")
	// ErrExists indicates that a key exists.
	ErrExists = errors.New("key existss")
)

// Initialize sets up bbolt db using file path and creates tables if required.
func Initialize(file string, tables []string) (*Store, error) {
	db, err := bbolt.Open(file, 0o666, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return &Store{}, err
	}
	s := &Store{db: db}
	return s, s.createTables(tables)
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
func (s *Store) Insert(value any, key, table string) error {
	_, err := Get[any](s, key, table)
	if err == nil {
		return ErrExists
	}
	if errors.Is(err, ErrNoResults) {
		return s.Save(value, key, table)
	}
	return err
}

// Save saves a value under key in the specified table.
func (s *Store) Save(value any, key, table string) error {
	marshalled, err := json.Marshal(&value)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(table))
		if b == nil {
			return ErrInvalidTableName
		}
		return b.Put([]byte(key), marshalled)
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
func (s *Store) Update(value any, key, table string) error {
	_, err := Get[any](s, key, table)
	if err == nil {
		return s.Save(value, key, table)
	}
	return ErrExists
}

// Get retrieves a value for key in specified table.
func Get[T any](s *Store, key, table string) (T, error) {
	var value T
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(table))
		if b == nil {
			return ErrInvalidTableName
		}
		v := b.Get([]byte(key))
		if v == nil {
			return ErrNoResults
		}
		if err := json.Unmarshal(v, &value); err != nil {
			return err
		}
		return nil
	})
	return value, err
}

// GetAll retrieves all values from table.
func GetAll[T any](s *Store, table string) ([]T, error) {
	var values []T
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(table))
		if b == nil {
			return ErrInvalidTableName
		}
		_ = b.ForEach(func(k, v []byte) error {
			var value T
			if err := json.Unmarshal(v, &value); err != nil {
				return err
			}
			values = append(values, value)
			return nil
		})
		return nil
	})
	return values, err
}

// Delete deletes the entry in table corresponding to key.
func Delete[T any](s *Store, key, table string) error {
	// verify table exists
	if _, err := Get[T](s, key, table); err != nil {
		return err
	}
	err := s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(table)).Delete([]byte(key))
	})
	return err
}

func (s *Store) createTable(name string) error {
	if err := s.db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *Store) createTables(tables []string) error {
	var errs error
	for _, table := range tables {
		if err := s.createTable(table); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}
