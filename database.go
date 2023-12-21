package boltdb

import (
	"encoding/json"
	"errors"
	"time"

	"go.etcd.io/bbolt"
)

var (
	ErrNoResults        = errors.New("no results found")
	ErrInvalidTableName = errors.New("invalid table")
	db                  *bbolt.DB
)

func Initialize(file string, tables []string) error {
	var err error
	db, err = bbolt.Open(file, 0666, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	return createTables(tables)
}

func Close() error {
	return db.Close()
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

func Delete[T any](value T, key, table string) error {
	if _, err := Get(value, key, table); err != nil {
		return err
	}
	err := db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(table)).Delete([]byte(key))
	})
	return err
}
