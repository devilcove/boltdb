package boltdb

import (
	"encoding/json"
	"errors"
	"slices"
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
	ErrExists = errors.New("key exists")
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

// CopyBucket copies a bucket from src path to dest path.
func (s *Store) CopyBucket(src, dest Path) error {
	if len(src) == 0 || len(dest) == 0 {
		return ErrInvalidPath
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		origin := tx.Bucket([]byte(src.last()))
		if len(src) > 1 {
			origin = parentBucket(tx, src)
		}
		if origin == nil {
			return ErrInvalidPath
		}
		return copyBucket(tx, origin, dest)
	})
}

// CopyKey copies a key from src parent bucket to dest bucket.
func (s *Store) CopyKey(sKey, dKey Path) error {
	if len(sKey) < 2 || len(dKey) < 2 {
		return ErrInvalidPath
	}
	name := []byte(sKey.last())
	dname := []byte(dKey.last())
	dbucket := toDBPath(dKey)[:len(dKey)-1]
	return s.db.Update(func(tx *bbolt.Tx) error {
		value, err := getRaw(tx, name, sKey)
		if err != nil {
			return err
		}
		k, err := getRaw(tx, name, dKey)
		if k != nil {
			return ErrExists
		}
		if err != nil && !errors.Is(err, ErrNoResults) {
			return err
		}
		parent, err := createBucket(tx, dbucket)
		// parent := getBucket(tx, dKey)
		if err != nil {
			return err
		}
		return parent.Put(dname, value)
	})
}

// CreateBucket creates a new bucket at given path.
func (s *Store) CreateBucket(path Path) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		_, err := createBucket(tx, toDBPath(path))
		return err
	})
}

// Delete deletes a key in a bucket.
func (s *Store) Delete(key string, parent Path) error {
	k := []byte(key)
	// verify key exists

	if _, err := s.GetRaw(k, parent); err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := getBucket(tx, parent)
		// no need to check for nil bucket as GetRow above would err
		return b.Delete(k)
	})
}

// DeleteBucket deletes the bucket at path.
func (s *Store) DeleteBucket(path Path) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return deleteBucket(tx, path)
	})
}

// EmptyBucket deletes all of a buckets children.
func (s *Store) EmptyBucket(path Path) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := getBucket(tx, path)
		if bucket == nil {
			return ErrInvalidPath
		}
		return bucket.ForEach(func(k, v []byte) error {
			if v == nil {
				return bucket.DeleteBucket(k)
			}
			return bucket.Delete(k)
		})
	})
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

// GetRaw retrieves the value of a key in specified bucket.
func (s *Store) GetRaw(key []byte, path Path) ([]byte, error) {
	value := []byte{}
	err := s.db.View(func(tx *bbolt.Tx) error {
		var err error
		value, err = getRaw(tx, key, path)
		return err
	})
	return value, err
}

func getRaw(tx *bbolt.Tx, key []byte, path Path) ([]byte, error) {
	var (
		value []byte
		err   error
	)
	bucket := getBucket(tx, path)
	if bucket == nil {
		return value, ErrInvalidPath
	}
	value = bucket.Get(key)
	if value == nil {
		err = ErrNoResults
	}
	return value, err
}

// Insert saves a value only if key does not exist.
func (s *Store) Insert(value any, key string, bucket Path) error {
	_, err := s.Get[any](key, bucket)
	if err == nil {
		return ErrExists
	}
	if errors.Is(err, ErrNoResults) {
		return s.Save(value, key, bucket)
	}
	return err
}

// MoveBucket copies bucket to new location and deletes original.
func (s *Store) MoveBucket(src, dest Path) error {
	var bucket *bbolt.Bucket
	if len(src) == 0 || len(dest) == 0 {
		return ErrInvalidPath
	}
	tx, err := s.Connection().Begin(true)
	if err != nil {
		return err
	}

	bucket = getBucket(tx, src)
	if bucket == nil {
		return errors.Join(tx.Rollback(), ErrInvalidPath)
	}
	if err := copyBucket(tx, bucket, dest); err != nil {
		return errors.Join(err, tx.Rollback())
	}

	if err := deleteBucket(tx, src); err != nil {
		return errors.Join(err, tx.Rollback())
	}

	return tx.Commit()
}

// MoveKey copies a key to new destination and deletes original.
func (s *Store) MoveKey(src, dest Path) error {
	if len(src) < 2 || len(dest) < 2 {
		return ErrInvalidPath
	}
	srcKeyName := toDBPath(src).last()
	destKeyName := toDBPath(dest).last()
	tx, err := s.Connection().Begin(true)
	if err != nil {
		return err
	}
	value, err := getRaw(tx, srcKeyName, src[:len(src)-1])
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	parent := parentBucket(tx, dest)
	if parent == nil {
		return errors.Join(ErrInvalidPath, tx.Rollback())
	}
	if v := parent.Get(destKeyName); v != nil {
		return errors.Join(ErrExists, tx.Rollback())
	}
	if err := parent.Put(destKeyName, value); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	parent = parentBucket(tx, src)
	if err := parent.Delete(srcKeyName); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

// RenameBucket renames a bucket.
func (s *Store) RenameBucket(path Path, name string) error {
	if len(path) == 0 {
		return ErrInvalidPath
	}
	newPath := slices.Clone(path)
	newPath[len(newPath)-1] = name
	return s.MoveBucket(path, newPath)
}

// RenameKey renames a Key.
func (s *Store) RenameKey(path Path, name string) error {
	if len(path) < 2 {
		return ErrInvalidPath
	}
	newPath := slices.Clone(path)
	newPath[len(newPath)-1] = name
	return s.MoveKey(path, newPath)
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
	if _, err := s.Get[any](key, bucket); err != nil {
		return err
	}
	return s.Save(value, key, bucket)
}

func copyBucket(tx *bbolt.Tx, bucket *bbolt.Bucket, dest Path) error {
	replica, err := createBucket(tx, toDBPath(dest))
	if err != nil {
		return err
	}
	return bucket.ForEach(func(k, v []byte) error {
		if v == nil {
			dest = append(dest, string(k))
			nested := bucket.Bucket(k)
			return copyBucket(tx, nested, dest)
		}
		return replica.Put(k, v)
	})
}

func createBucket(tx *bbolt.Tx, path path) (*bbolt.Bucket, error) {
	if len(path) == 0 {
		return nil, ErrInvalidPath
	}
	b, err := tx.CreateBucketIfNotExists(path[0])
	if err != nil {
		return nil, err
	}
	for _, p := range path[1:] {
		b, err = b.CreateBucketIfNotExists(p)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (s *Store) createBuckets(buckets []Path) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		var errs error
		for _, p := range buckets {
			if _, err := createBucket(tx, toDBPath(p)); err != nil {
				errs = errors.Join(errs, err)
			}
		}
		return errs
	})
}

func deleteBucket(tx *bbolt.Tx, path Path) error {
	if len(path) == 0 {
		return ErrInvalidPath
	}
	if len(path) == 1 {
		return tx.DeleteBucket([]byte(path[0]))
	}
	parent := parentBucket(tx, path)
	if parent == nil {
		return ErrInvalidPath
	}
	return parent.DeleteBucket([]byte(path.last()))
}

func getBucket(tx *bbolt.Tx, path Path) *bbolt.Bucket {
	if len(path) == 0 {
		return nil
	}
	dbpath := toDBPath(path)
	bucket := tx.Bucket(dbpath[0])
	if bucket != nil {
		return bucket
	}
	for _, p := range dbpath[1:] {
		if len(p) == 0 {
			return nil
		}
		bucket = bucket.Bucket(p)
		if bucket == nil {
			return bucket
		}
	}
	return bucket
}

func parentBucket(tx *bbolt.Tx, path Path) *bbolt.Bucket {
	b := tx.Bucket([]byte(path[0]))
	if len(path) == 1 || b == nil {
		return nil
	}
	for _, p := range path[1 : len(path)-1] {
		b = b.Bucket([]byte(p))
	}
	return b
}
