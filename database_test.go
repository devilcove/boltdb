package boltdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var tables = []string{"users", "keys", "tables"}

type User struct {
	User string
	Pass string
}

type Other struct {
	Name string
	Foo  int
}

func TestInitialize(t *testing.T) {
	t.Log(tables)
	//tables := []string{"users", "keys", "networks"}
	t.Run("valid", func(t *testing.T) {
		err := Initialize("./test.db", tables)
		assert.Nil(t, err)
		err = Close()
		assert.Nil(t, err)
	})
	t.Run("pathDoesNotExist", func(t *testing.T) {
		err := Initialize("/tmp/thispathdoesnotexist/test.db", tables)
		assert.Contains(t, err.Error(), "no such file or directory")
	})
}

func TestClose(t *testing.T) {
	err := Initialize("./test.db", tables)
	assert.Nil(t, err)
	t.Run("open", func(t *testing.T) {
		err = Close()
		assert.Nil(t, err)
	})
	t.Run("closed", func(t *testing.T) {
		err := Close()
		assert.Nil(t, err)
	})
}

func TestSave(t *testing.T) {
	err := Initialize("./test.db", tables)
	assert.Nil(t, err)
	t.Run("noSuchTable", func(t *testing.T) {
		err := Save("testing", "key", "nosuchtable")
		assert.Equal(t, ErrInvalidTableName, err)
	})
	t.Run("invalidjson", func(t *testing.T) {
		function := func() {}
		value := struct {
			Function func()
		}{
			Function: function,
		}
		err := Save(value, "hello", "users")
		assert.NotNil(t, err)
	})
	t.Run("valid", func(t *testing.T) {
		err := Save("testing", "name", "users")
		assert.Nil(t, err)
	})
	err = Close()
	assert.Nil(t, err)
}

func TestGetOne(t *testing.T) {
	err := Initialize("./test.db", tables)
	assert.Nil(t, err)
	deleteTestEntries(t)
	t.Run("noSuchTable", func(t *testing.T) {
		value, err := Get[User]("first", "nosuchtable")
		assert.Equal(t, User{}, value)
		assert.Equal(t, ErrInvalidTableName, err)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := Get[User]("first", "users")
		assert.Equal(t, User{}, value)
		assert.Equal(t, ErrNoResults, err)
	})
	createTestEntries(t)
	t.Run("wrongkey", func(t *testing.T) {
		value, err := Get[User]("third", "users")
		assert.Equal(t, ErrNoResults, err)
		assert.Equal(t, User{}, value)
	})
	t.Run("wrongType", func(t *testing.T) {
		value, err := Get[Other]("first", "users")
		assert.Nil(t, err)
		assert.Equal(t, Other{}, value)
	})
	t.Run("valid", func(t *testing.T) {
		value, err := Get[User]("first", "users")
		assert.Nil(t, err)
		assert.Equal(t, "first", value.User)
		assert.Equal(t, "password", value.Pass)
	})
	deleteTestEntries(t)
	err = Close()
	assert.Nil(t, err)
}

func TestGetAll(t *testing.T) {
	err := Initialize("./test.db", tables)
	assert.Nil(t, err)
	deleteTestEntries(t)
	t.Run("noSuchTable", func(t *testing.T) {
		value, err := GetAll[User]("nosuchtable")
		assert.Equal(t, []User(nil), value)
		assert.Equal(t, ErrInvalidTableName, err)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := GetAll[User]("users")
		assert.Equal(t, []User(nil), value)
		assert.Nil(t, err)
	})
	createTestEntries(t)
	t.Run("valid", func(t *testing.T) {
		value, err := GetAll[User]("users")
		assert.Nil(t, err)
		assert.Equal(t, "first", value[0].User)
		assert.Equal(t, "password", value[0].Pass)
	})
	deleteTestEntries(t)
	err = Close()
	assert.Nil(t, err)
}

func TestDelete(t *testing.T) {
	err := Initialize("./test.db", tables)
	assert.Nil(t, err)
	t.Run("nonexistentTable", func(t *testing.T) {
		err := Delete[User]("first", "tabledoesnotexist")
		assert.Equal(t, ErrInvalidTableName, err)
	})
	t.Run("nosuchrecord", func(t *testing.T) {
		err := Delete[User]("first", "users")
		assert.Equal(t, ErrNoResults, err)
	})
	t.Run("valid", func(t *testing.T) {
		createTestEntries(t)
		err := Delete[User]("first", "users")
		assert.Nil(t, err)
	})
	deleteTestEntries(t)
	err = Close()
	assert.Nil(t, err)
}

func createTestEntries(t *testing.T) {
	t.Helper()
	users := []User{
		{
			User: "first",
			Pass: "password",
		},
		{
			User: "second",
			Pass: "testing",
		},
	}
	for _, user := range users {
		err := Save(user, user.User, "users")
		assert.Nil(t, err)
	}
}

func deleteTestEntries(t *testing.T) {
	t.Helper()
	values, err := GetAll[User]("users")
	assert.Nil(t, err)
	for _, value := range values {
		err := Delete[User](value.User, "users")
		assert.Nil(t, err)
	}
}
