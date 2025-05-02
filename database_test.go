package boltdb_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/devilcove/boltdb"
	"github.com/stretchr/testify/assert"
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
		assert.Nil(t, err)
		err = boltdb.Close()
		assert.Nil(t, err)
	})
	t.Run("pathDoesNotExist", func(t *testing.T) {
		err := boltdb.Initialize("/tmp/thispathdoesnotexist/test.db", tables)
		assert.Contains(t, err.Error(), "no such file or directory")
	})
}

func TestTables(t *testing.T) {
	err := testInit()
	assert.Nil(t, err)
	names := boltdb.Tables()
	for _, v := range names {
		assert.True(t, slices.Contains(tables, v))
	}
	for _, v := range tables {
		assert.True(t, slices.Contains(names, v))
	}
}
func TestClose(t *testing.T) {
	err := testInit()
	assert.Nil(t, err)
	t.Run("open", func(t *testing.T) {
		err = boltdb.Close()
		assert.Nil(t, err)
	})
	t.Run("closed", func(t *testing.T) {
		err := boltdb.Close()
		assert.Nil(t, err)
	})
}

func TestSave(t *testing.T) {
	err := testInit()
	assert.Nil(t, err)
	t.Run("noSuchTable", func(t *testing.T) {
		err := boltdb.Save("testing", "key", "nosuchtable")
		assert.Equal(t, boltdb.ErrInvalidTableName, err)
	})
	t.Run("invalidjson", func(t *testing.T) {
		function := func() {}
		value := struct {
			Function func()
		}{
			Function: function,
		}
		err := boltdb.Save(value, "hello", "users")
		assert.NotNil(t, err)
	})
	t.Run("valid", func(t *testing.T) {
		user := User{
			UserName: "testing",
		}
		err := boltdb.Save(user, user.UserName, "users")
		assert.Nil(t, err)
	})
	err = deleteTestEntries()
	assert.Nil(t, err)
	err = boltdb.Close()
	assert.Nil(t, err)
}

func TestInsert(t *testing.T) {
	err := testInit()
	assert.Nil(t, err)
	err = deleteTestEntries()
	assert.Nil(t, err)

	t.Run("valid", func(t *testing.T) {
		user := User{
			UserName: "testing",
		}
		err := boltdb.Insert(user, user.UserName, "users")
		assert.Nil(t, err)
	})
	t.Run("exists", func(t *testing.T) {
		user := User{
			UserName: "testing",
		}
		err := boltdb.Insert(user, user.UserName, "users")
		assert.True(t, errors.Is(err, boltdb.ErrExists))
	})
	//deleteTestEntries(t)
	err = boltdb.Close()
	assert.Nil(t, err)
}

func TestGetOne(t *testing.T) {
	//err := Initialize("./test.db", tables)
	err := testInit()
	assert.Nil(t, err)
	err = deleteTestEntries()
	assert.Nil(t, err)
	t.Log(err)
	t.Run("noSuchTable", func(t *testing.T) {
		value, err := boltdb.Get[User]("first", "nosuchtable")
		assert.Equal(t, User{}, value)
		assert.Equal(t, boltdb.ErrInvalidTableName, err)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := boltdb.Get[User]("first", "users")
		assert.Equal(t, User{}, value)
		assert.Equal(t, boltdb.ErrNoResults, err)
	})
	createTestEntries(t)
	t.Run("wrongkey", func(t *testing.T) {
		value, err := boltdb.Get[User]("third", "users")
		assert.Equal(t, boltdb.ErrNoResults, err)
		assert.Equal(t, User{}, value)
	})
	t.Run("wrongType", func(t *testing.T) {
		value, err := boltdb.Get[Other]("first", "users")
		assert.Nil(t, err)
		assert.Equal(t, Other{}, value)
	})
	t.Run("valid", func(t *testing.T) {
		value, err := boltdb.Get[User]("first", "users")
		assert.Nil(t, err)
		assert.Equal(t, "first", value.UserName)
		assert.Equal(t, "password", value.Password)
	})
}

func TestGetAll(t *testing.T) {
	err := testInit()
	assert.Nil(t, err)
	err = deleteTestEntries()
	assert.Nil(t, err)
	t.Run("noSuchTable", func(t *testing.T) {
		value, err := boltdb.GetAll[User]("nosuchtable")
		assert.Equal(t, []User(nil), value)
		assert.Equal(t, boltdb.ErrInvalidTableName, err)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := boltdb.GetAll[User]("users")
		assert.Equal(t, []User(nil), value)
		assert.Nil(t, err)
	})
	createTestEntries(t)
	t.Run("valid", func(t *testing.T) {
		value, err := boltdb.GetAll[User]("users")
		assert.Nil(t, err)
		assert.Equal(t, "first", value[0].UserName)
		assert.Equal(t, "password", value[0].Password)
	})
	//deleteTestEntries(t)
	err = boltdb.Close()
	assert.Nil(t, err)
}

func TestUpdate(t *testing.T) {
	err := testInit()
	assert.Nil(t, err)
	err = deleteTestEntries()
	assert.Nil(t, err)
	t.Run("does not exist", func(t *testing.T) {
		user := User{
			UserName: "testing",
		}
		err := boltdb.Update(user, user.UserName, "users")
		assert.True(t, errors.Is(err, boltdb.ErrExists))
	})
	t.Run("existing", func(t *testing.T) {
		user := User{
			UserName: "testing",
		}
		err := boltdb.Save(user, user.UserName, "users")
		assert.Nil(t, err)
		user2 := User{
			UserName: "test2",
			Password: "nopass",
		}
		err = boltdb.Update(user2, user.UserName, "users")
		assert.Nil(t, err)
		user, err = boltdb.Get[User](user.UserName, "users")
		assert.Nil(t, err)
		assert.Equal(t, user2.UserName, user.UserName)
	})
	//deleteTestEntries(t)
	err = boltdb.Close()
	assert.Nil(t, err)
}

func TestDelete(t *testing.T) {
	err := testInit()
	assert.Nil(t, err)
	err = deleteTestEntries()
	assert.Nil(t, err)
	t.Run("nonexistentTable", func(t *testing.T) {
		err := boltdb.Delete[User]("first", "tabledoesnotexist")
		assert.Equal(t, boltdb.ErrInvalidTableName, err)
	})
	t.Run("nosuchrecord", func(t *testing.T) {
		err := boltdb.Delete[User]("first", "users")
		assert.Equal(t, boltdb.ErrNoResults, err)
	})
	t.Run("valid", func(t *testing.T) {
		createTestEntries(t)
		err := boltdb.Delete[User]("first", "users")
		assert.Nil(t, err)
	})
	//deleteTestEntries(t)
	err = boltdb.Close()
	assert.Nil(t, err)
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
		assert.Nil(t, err)
	}
}

func deleteTestEntries() error {
	//t.Helper()
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
