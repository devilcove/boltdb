package boltdb

import (
	"encoding/json"
	"errors"
	"time"

	"go.etcd.io/bbolt"
)

// Generic error results
var (
	ErrNoResults        = errors.New("no results found") // ErrNoResults indicates query found no results
	ErrInvalidTableName = errors.New("invalid table")    // ErrInvalidTableName indicates that specified table does not exist
	ErrNoConnection     = errors.New("no db connection") //
	db                  *bbolt.DB
)

// Initialize sets up bbolt db using file path and creates tables if required
func Initialize(file string, tables []string) error {
	var err error
	db, err = bbolt.Open(file, 0666, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	return createTables(tables)
}

// Close closes the database
func Close() error {
	return db.Close()
}

// Connection returns the connection to the store for more advanced queries by caller
func Connection() *bbolt.DB {
	return db
}

func createTables(tables []string) error {
	var errs error
	for _, table := range tables {
		if err := createTable(table); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

func createTable(name string) error {
	if err := db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// Save saves a value under key in the specified table
func Save(value any, key, table string) error {
	marshalled, err := json.Marshal(&value)
	if err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(table))
		if b == nil {
			return ErrInvalidTableName
		}
		return b.Put([]byte(key), marshalled)
	})
}

// Get retrieves a value for key in specified table
func Get[T any](value T, key, table string) (T, error) {
	err := db.View(func(tx *bbolt.Tx) error {
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

// GetAll retrieves all values from table
func GetAll[T any](value T, table string) ([]T, error) {
	var values []T
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(table))
		if b == nil {
			return ErrInvalidTableName
		}
		_ = b.ForEach(func(k, v []byte) error {
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

// Delete deletes the entry in table corresponding to key
func Delete[T any](value T, key, table string) error {
	if _, err := Get(value, key, table); err != nil {
		return err
	}
	err := db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(table)).Delete([]byte(key))
	})
	return err
}
