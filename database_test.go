package boltdb_test

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Kairum-Labs/should"
	"github.com/devilcove/boltdb"
)

var tables = []string{"users", "keys", "tables"}

type Other struct {
	Bar string
	Foo int
}

func TestInitialize(t *testing.T) {
	t.Log(tables)
	t.Run("valid", func(t *testing.T) {
		err := testInit()
		should.BeNil(t, err)
		err = boltdb.Close()
		should.BeNil(t, err)
	})
	t.Run("pathDoesNotExist", func(t *testing.T) {
		var pathError *fs.PathError
		err := boltdb.Initialize("/tmp/thispathdoesnotexist/test.db", tables)
		should.BeTrue(t, errors.As(err, &pathError))
	})
}

func TestTables(t *testing.T) {
	err := testInit()
	should.BeNil(t, err)
	names := boltdb.Tables()
	for _, v := range names {
		should.BeTrue(t, slices.Contains(tables, v))
		should.Contain(t, tables, v)
	}
	for _, v := range tables {
		should.BeTrue(t, slices.Contains(names, v))
	}
}

func TestClose(t *testing.T) {
	err := testInit()
	should.BeNil(t, err)
	t.Run("open", func(t *testing.T) {
		err = boltdb.Close()
		should.BeNil(t, err)
	})
	t.Run("closed", func(t *testing.T) {
		err := boltdb.Close()
		should.BeNil(t, err)
	})
}

func TestSave(t *testing.T) {
	err := testInit()
	should.BeNil(t, err)
	t.Run("noSuchTable", func(t *testing.T) {
		err := boltdb.Save("testing", "key", "nosuchtable")
		should.BeEqual(t, err, boltdb.ErrInvalidTableName)
	})
	t.Run("invalidjson", func(t *testing.T) {
		function := func() {}
		value := struct {
			Function func()
		}{
			Function: function,
		}
		err := boltdb.Save(value, "hello", "users")
		should.NotBeNil(t, err)
	})
	t.Run("valid", func(t *testing.T) {
		user := User{
			UserName: "testing",
		}
		err := boltdb.Save(user, user.UserName, "users")
		should.BeNil(t, err)
	})
	err = deleteTestEntries()
	should.BeNil(t, err)
	err = boltdb.Close()
	should.BeNil(t, err)
}

func TestInsert(t *testing.T) {
	err := testInit()
	should.BeNil(t, err)
	err = deleteTestEntries()
	should.BeNil(t, err)

	t.Run("valid", func(t *testing.T) {
		user := User{
			UserName: "testing",
		}
		err := boltdb.Insert(user, user.UserName, "users")
		should.BeNil(t, err)
	})
	t.Run("exists", func(t *testing.T) {
		user := User{
			UserName: "testing",
		}
		err := boltdb.Insert(user, user.UserName, "users")
		should.BeTrue(t, errors.Is(err, boltdb.ErrExists))
	})
	// deleteTestEntries(t)
	err = boltdb.Close()
	should.BeNil(t, err)
}

func TestGetOne(t *testing.T) {
	// err := Initialize("./test.db", tables)
	err := testInit()
	should.BeNil(t, err)
	err = deleteTestEntries()
	should.BeNil(t, err)
	t.Log(err)
	t.Run("noSuchTable", func(t *testing.T) {
		value, err := boltdb.Get[User]("first", "nosuchtable")
		should.BeEqual(t, value, User{})
		should.BeEqual(t, err, boltdb.ErrInvalidTableName)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := boltdb.Get[User]("first", "users")
		should.BeEqual(t, value, User{})
		should.BeEqual(t, err, boltdb.ErrNoResults)
	})
	createTestEntries(t)
	t.Run("wrongkey", func(t *testing.T) {
		value, err := boltdb.Get[User]("third", "users")
		should.BeEqual(t, err, boltdb.ErrNoResults)
		should.BeEqual(t, value, User{})
	})
	t.Run("wrongType", func(t *testing.T) {
		value, err := boltdb.Get[Other]("first", "users")
		should.BeNil(t, err)
		should.BeEqual(t, value, Other{})
	})
	t.Run("valid", func(t *testing.T) {
		value, err := boltdb.Get[User]("first", "users")
		should.BeNil(t, err)
		should.BeEqual(t, value.UserName, "first")
		should.BeEqual(t, value.Password, "password")
	})
}

func TestGetAll(t *testing.T) {
	err := testInit()
	should.BeNil(t, err)
	err = deleteTestEntries()
	should.BeNil(t, err)
	t.Run("noSuchTable", func(t *testing.T) {
		value, err := boltdb.GetAll[User]("nosuchtable")
		should.BeEmpty(t, value)
		should.BeEqual(t, err, boltdb.ErrInvalidTableName)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := boltdb.GetAll[User]("users")
		should.BeEmpty(t, value)
		should.BeNil(t, err)
	})
	createTestEntries(t)
	t.Run("valid", func(t *testing.T) {
		value, err := boltdb.GetAll[User]("users")
		should.BeNil(t, err)
		should.BeEqual(t, value[0].UserName, "first")
		should.BeEqual(t, value[0].Password, "password")
	})
	// deleteTestEntries(t)
	err = boltdb.Close()
	should.BeNil(t, err)
}

func TestUpdate(t *testing.T) {
	err := testInit()
	should.BeNil(t, err)
	err = deleteTestEntries()
	should.BeNil(t, err)
	t.Run("does not exist", func(t *testing.T) {
		user := User{
			UserName: "testing",
		}
		err := boltdb.Update(user, user.UserName, "users")
		should.BeTrue(t, errors.Is(err, boltdb.ErrExists))
	})
	t.Run("existing", func(t *testing.T) {
		user := User{
			UserName: "testing",
		}
		err := boltdb.Save(user, user.UserName, "users")
		should.BeNil(t, err)
		user2 := User{
			UserName: "test2",
			Password: "nopass",
		}
		err = boltdb.Update(user2, user.UserName, "users")
		should.BeNil(t, err)
		user, err = boltdb.Get[User](user.UserName, "users")
		should.BeNil(t, err)
		should.BeEqual(t, user.UserName, user2.UserName)
	})
	// deleteTestEntries(t)
	err = boltdb.Close()
	should.BeNil(t, err)
}

func TestDelete(t *testing.T) {
	err := testInit()
	should.BeNil(t, err)
	err = deleteTestEntries()
	should.BeNil(t, err)
	t.Run("nonexistentTable", func(t *testing.T) {
		err := boltdb.Delete[User]("first", "tabledoesnotexist")
		should.BeEqual(t, err, boltdb.ErrInvalidTableName)
	})
	t.Run("nosuchrecord", func(t *testing.T) {
		err := boltdb.Delete[User]("first", "users")
		should.BeEqual(t, err, boltdb.ErrNoResults)
	})
	t.Run("valid", func(t *testing.T) {
		createTestEntries(t)
		err := boltdb.Delete[User]("first", "users")
		should.BeNil(t, err)
	})
	// deleteTestEntries(t)
	err = boltdb.Close()
	should.BeNil(t, err)
}

func createTestEntries(t *testing.T) {
	t.Helper()
	users := []User{
		{
			UserName: "first",
			Password: "password",
		},
		{
			UserName: "second",
			Password: "testing",
		},
	}
	for _, user := range users {
		err := boltdb.Save(user, user.UserName, "users")
		should.BeNil(t, err)
	}
}

func deleteTestEntries() error {
	// t.Helper()
	values, err := boltdb.GetAll[User]("users")
	if err != nil {
		if strings.Contains(err.Error(), "no results") {
			return nil
		}
		return err
	}
	for _, value := range values {
		if err := boltdb.Delete[User](value.UserName, "users"); err != nil {
			if strings.Contains(err.Error(), "no results") {
				return nil
			}
			return err
		}
	}
	return nil
}

func testInit() error {
	if err := os.Remove("./test.db"); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := boltdb.Initialize("./test.db", tables); err != nil {
		return err
	}
	return nil
}
