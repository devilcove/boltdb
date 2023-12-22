package boltdb_test

import (
	"encoding/json"
	"fmt"

	"github.com/devilcove/boltdb"
	"go.etcd.io/bbolt"
)

const UserTable = "users"

type User struct {
	UserName string
	Password string
	IsAdmin  bool
}

func ExampleConnection() {
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
	db := boltdb.Connection()
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
