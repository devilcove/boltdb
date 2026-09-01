package boltdb_test

import (
	"encoding/json"
	"fmt"

	"github.com/devilcove/boltdb"
	"go.etcd.io/bbolt"
)

type User struct {
	UserName string
	Password string
	IsAdmin  bool
}

const UserTable = "users"

func ExampleStore_Connection() {
	if AdminExists() {
		fmt.Println("admin exists")
	} else {
		fmt.Println("admin does not exist")
	}
	// Output: admin does not exist
}

func AdminExists() bool {
	var user User
	var found bool
	s := boltdb.Store{}
	db := s.Connection()
	if db == nil {
		return found
	}
	if err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(UserTable))
		if b == nil {
			return boltdb.ErrNoResults
		}
		_ = b.ForEach(func(k, v []byte) error {
			if err := json.Unmarshal(v, &user); err != nil {
				return err
			}
			if user.IsAdmin {
				found = true
			}
			return nil
		})
		return nil
	}); err != nil {
		return false
	}
	return found
}
