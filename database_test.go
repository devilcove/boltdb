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

type Other struct {
	Bar string
	Foo int
}

const testUser = "testing"

var tables = []string{"users", "keys", "tables"}

func TestInitialize(t *testing.T) {
	t.Log(tables)
	t.Run("valid", func(t *testing.T) {
		s, err := testInit()
		should.BeNil(t, err)
		err = s.Close()
		should.BeNil(t, err)
	})
	t.Run("pathDoesNotExist", func(t *testing.T) {
		var pathError *fs.PathError
		_, err := boltdb.Initialize("/tmp/thispathdoesnotexist/test.db", tables)
		should.BeTrue(t, errors.As(err, &pathError))
	})
}

func TestTables(t *testing.T) {
	s, err := testInit()
	should.BeNil(t, err)
	names := s.Tables()
	for _, v := range names {
		should.BeTrue(t, slices.Contains(tables, v))
		should.Contain(t, tables, v)
	}
	for _, v := range tables {
		should.BeTrue(t, slices.Contains(names, v))
	}
}

func TestClose(t *testing.T) {
	s, err := testInit()
	should.BeNil(t, err)
	t.Run("open", func(t *testing.T) {
		err = s.Close()
		should.BeNil(t, err)
	})
	t.Run("closed", func(t *testing.T) {
		err := s.Close()
		should.BeNil(t, err)
	})
}

func TestSave(t *testing.T) {
	s, err := testInit()
	should.BeNil(t, err)
	t.Run("noSuchTable", func(t *testing.T) {
		err := s.Save("testing", "key", "nosuchtable")
		should.BeEqual(t, err, boltdb.ErrInvalidTableName)
	})
	t.Run("invalidjson", func(t *testing.T) {
		function := func() {}
		value := struct {
			Function func()
		}{
			Function: function,
		}
		err := s.Save(value, "hello", "users")
		should.NotBeNil(t, err)
	})
	t.Run("valid", func(t *testing.T) {
		user := User{
			UserName: testUser,
		}
		err := s.Save(user, user.UserName, "users")
		should.BeNil(t, err)
	})
	err = deleteTestEntries(s)
	should.BeNil(t, err)
	err = s.Close()
	should.BeNil(t, err)
}

func TestInsert(t *testing.T) {
	s, err := testInit()
	should.BeNil(t, err)
	err = deleteTestEntries(s)
	should.BeNil(t, err)

	t.Run("valid", func(t *testing.T) {
		user := User{
			UserName: testUser,
		}
		err := s.Insert(user, user.UserName, "users")
		should.BeNil(t, err)
	})
	t.Run("exists", func(t *testing.T) {
		user := User{
			UserName: testUser,
		}
		err := s.Insert(user, user.UserName, "users")
		should.BeTrue(t, errors.Is(err, boltdb.ErrExists))
	})
	// deleteTestEntries(t)
	err = s.Close()
	should.BeNil(t, err)
}

func TestGetOne(t *testing.T) {
	// err := Initialize("./test.db", tables)
	s, err := testInit()
	should.BeNil(t, err)
	err = deleteTestEntries(s)
	should.BeNil(t, err)
	t.Log(err)
	t.Run("noSuchTable", func(t *testing.T) {
		value, err := boltdb.Get[User](s, "first", "nosuchtable")
		should.BeEqual(t, value, User{})
		should.BeEqual(t, err, boltdb.ErrInvalidTableName)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := boltdb.Get[User](s, "first", "users")
		should.BeEqual(t, value, User{})
		should.BeEqual(t, err, boltdb.ErrNoResults)
	})
	createTestEntries(t, s)
	t.Run("wrongkey", func(t *testing.T) {
		value, err := boltdb.Get[User](s, "third", "users")
		should.BeEqual(t, err, boltdb.ErrNoResults)
		should.BeEqual(t, value, User{})
	})
	t.Run("wrongType", func(t *testing.T) {
		value, err := boltdb.Get[Other](s, "first", "users")
		should.BeNil(t, err)
		should.BeEqual(t, value, Other{})
	})
	t.Run("valid", func(t *testing.T) {
		value, err := boltdb.Get[User](s, "first", "users")
		should.BeNil(t, err)
		should.BeEqual(t, value.UserName, "first")
		should.BeEqual(t, value.Password, "password")
	})
}

func TestGetAll(t *testing.T) {
	s, err := testInit()
	should.BeNil(t, err)
	err = deleteTestEntries(s)
	should.BeNil(t, err)
	t.Run("noSuchTable", func(t *testing.T) {
		value, err := boltdb.GetAll[User](s, "nosuchtable")
		should.BeEmpty(t, value)
		should.BeEqual(t, err, boltdb.ErrInvalidTableName)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := boltdb.GetAll[User](s, "users")
		should.BeEmpty(t, value)
		should.BeNil(t, err)
	})
	createTestEntries(t, s)
	t.Run("valid", func(t *testing.T) {
		value, err := boltdb.GetAll[User](s, "users")
		should.BeNil(t, err)
		should.BeEqual(t, value[0].UserName, "first")
		should.BeEqual(t, value[0].Password, "password")
	})
	// deleteTestEntries(t)
	err = s.Close()
	should.BeNil(t, err)
}

func TestUpdate(t *testing.T) {
	s, err := testInit()
	should.BeNil(t, err)
	err = deleteTestEntries(s)
	should.BeNil(t, err)
	t.Run("does not exist", func(t *testing.T) {
		user := User{
			UserName: testUser,
		}
		err := s.Update(user, user.UserName, "users")
		should.BeTrue(t, errors.Is(err, boltdb.ErrExists))
	})
	t.Run("existing", func(t *testing.T) {
		user := User{
			UserName: testUser,
		}
		err := s.Save(user, user.UserName, "users")
		should.BeNil(t, err)
		user2 := User{
			UserName: "test2",
			Password: "nopass",
		}
		err = s.Update(user2, user.UserName, "users")
		should.BeNil(t, err)
		user, err = boltdb.Get[User](s, user.UserName, "users")
		should.BeNil(t, err)
		should.BeEqual(t, user.UserName, user2.UserName)
	})
	// deleteTestEntries(t)
	err = s.Close()
	should.BeNil(t, err)
}

func TestDelete(t *testing.T) {
	s, err := testInit()
	should.BeNil(t, err)
	err = deleteTestEntries(s)
	should.BeNil(t, err)
	t.Run("nonexistentTable", func(t *testing.T) {
		err := boltdb.Delete[User](s, "first", "tabledoesnotexist")
		should.BeEqual(t, err, boltdb.ErrInvalidTableName)
	})
	t.Run("nosuchrecord", func(t *testing.T) {
		err := boltdb.Delete[User](s, "first", "users")
		should.BeEqual(t, err, boltdb.ErrNoResults)
	})
	t.Run("valid", func(t *testing.T) {
		createTestEntries(t, s)
		err := boltdb.Delete[User](s, "first", "users")
		should.BeNil(t, err)
	})
	// deleteTestEntries(t)
	err = s.Close()
	should.BeNil(t, err)
}

func createTestEntries(t *testing.T, s *boltdb.Store) {
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
		err := s.Save(user, user.UserName, "users")
		should.BeNil(t, err)
	}
}

func deleteTestEntries(s *boltdb.Store) error {
	// t.Helper()
	values, err := boltdb.GetAll[User](s, "users")
	if err != nil {
		if strings.Contains(err.Error(), "no results") {
			return nil
		}
		return err
	}
	for _, value := range values {
		if err := boltdb.Delete[User](s, value.UserName, "users"); err != nil {
			if strings.Contains(err.Error(), "no results") {
				return nil
			}
			return err
		}
	}
	return nil
}

func testInit() (*boltdb.Store, error) {
	if err := os.Remove("./test.db"); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return &boltdb.Store{}, err
		}
	}
	return boltdb.Initialize("./test.db", tables)
}
