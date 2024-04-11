boltdb
-------
![tests](https://github.com/devilcove/boltdb/actions/workflows/test.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/devilcove/boltdb?style=flat-square)](https://goreportcard.com/report/github.com/devilcove/boltdb)
[![Go Reference](https://pkg.go.dev/badge/github.com/devilcove/boltdb.svg)](https://pkg.go.dev/github.com/devilcove/boltdb)
[![Go Coverage](https://githubusercontent.com/wiki/devilcove/boltdb/coverage/coverage.svg)](https://raw.githack.com/wiki/devilcove/boltdb/coverage/coverage.html)

Go module `github.com/devilcove/boltdb` is a generic abstractions layer for basic crud operations on a `go.etcd.io/bbolt` key/value store

Installing
----------
To start using, install Go and run `go get`:
````
go get github.com/devilcove/boltdb@latest
````
Functions
---------
### Initialization
call initialize with the path to store and list of tables.  Store and tables will be created if they do not exist.
````
import "github.com/devilcove/bbolt"

if err := boltdb.Initialize(path, []string{"users", "networks"}); err != nil{
  return err
}
defer boltd.Close()
````
### Create/Update
pass the key/value pair along with table name
#### Save -- save key/value always (overwrite existing or create new)
#### Insert -- save only iff key does not exist
#### Update -- save only iff key exists
````
cont userTable = "users"

user := models.User {
  Username: "admin",
  Password: "encrypted password",
  IsAdmin: true,
}

if err := boltdb.Save(user, user.Username, userTable); err != nil {
  return err
}
````
 
### Read
return value of key in table
````
user, err := boltdb.Get[models.User]("admin", userTable)
if err != nil {
  return err
}
````
retrieve all values from table
````
users, err := boltdb.GetAll[models.User](userTable)
if err != nil {
  return err
}
````
### Delete
delete value of key in table
````
if err := boltdb.Delete[models.User]("admin", userTable); err != nil {
  return err
}
````
#### Advanced Usage
the db connection is made available if more advanced queries are needed
````
import (
  "encoding/json"
  "errors"

  "github.com/devilcove/boltdb"
  "go.etcd.io/bbolt"
)

func AdminExists() bool {
	var user models.User
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

````  
