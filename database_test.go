package boltdb_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/Kairum-Labs/should"
	"github.com/devilcove/boltdb"
	berror "go.etcd.io/bbolt/errors"
)

type Other struct {
	Bar string
	Foo int
}

const testUser = "testing"

var tables = []boltdb.Path{userTable, keysTable, tablesTable, nestedTable}

var (
	userTable        = boltdb.Path{"users"}
	invalidTable     = boltdb.Path{"invalidtablename"}
	nestedTable      = boltdb.Path{"nested", "one"}
	tablesTable      = boltdb.Path{"tables"}
	keysTable        = boltdb.Path{"keys"}
	emptyTable       = boltdb.Path{}
	emptyNestedTable = boltdb.Path{"nested", "", "bar"}
)

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

func TestCreateBucket(t *testing.T) {
	s, err := testInit()
	should.NotBeError(t, err)
	t.Run("invalid path", func(t *testing.T) {
		should.BeErrorIs(t, s.CreateBucket(emptyTable), boltdb.ErrInvalidPath)
	})
	t.Run("invalid nested", func(t *testing.T) {
		should.BeErrorIs(t, s.CreateBucket(emptyNestedTable), berror.ErrBucketNameRequired)
	})
	t.Run("valid", func(t *testing.T) {
		should.NotBeError(t, s.CreateBucket(userTable))
	})
	t.Run("valid", func(t *testing.T) {
		should.NotBeError(t, s.CreateBucket(nestedTable))
	})
}

// func TestTables(t *testing.T) {
// 	s, err := testInit()
// 	should.BeNil(t, err)
// 	names := s.Tables()
// 	for _, v := range names {
// 		should.BeTrue(t, slices.Contains(tables, v))
// 		should.Contain(t, tables, v)
// 	}
// 	for _, v := range tables {
// 		should.BeTrue(t, slices.Contains(names, v))
// 	}
// }

func TestClose(t *testing.T) {
	s, err := testInit()
	should.NotBeError(t, err)
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
	user := User{
		UserName: testUser,
	}
	s, err := testInit()
	should.BeNil(t, err)
	t.Run("noSuchTable", func(t *testing.T) {
		should.BeErrorIs(t, s.Save("testing", "key", invalidTable), boltdb.ErrInvalidPath)
	})
	t.Run("invalidjson", func(t *testing.T) {
		function := func() {}
		value := struct {
			Function func()
		}{
			Function: function,
		}
		should.BeError(t, s.Save(value, "hello", userTable))
	})
	t.Run("emptyTable", func(t *testing.T) {
		should.BeErrorIs(t, s.Save("value", "key", emptyTable), boltdb.ErrInvalidPath)
	})
	t.Run("emptyNestedTable", func(t *testing.T) {
		should.BeErrorIs(t, s.Save("value", "key", emptyNestedTable), boltdb.ErrInvalidPath)
	})
	t.Run("valid", func(t *testing.T) {
		should.NotBeError(t, s.Save(user, user.UserName, userTable))
	})
	t.Run("nested", func(t *testing.T) {
		should.NotBeError(t, s.Save(user, user.UserName, nestedTable))
	})
	err = deleteTestEntries(s)
	should.BeNil(t, err)
	err = s.Close()
	should.BeNil(t, err)
}

func TestInsert(t *testing.T) {
	user := User{
		UserName: testUser,
	}
	s, err := testInit()
	should.NotBeError(t, err)
	should.NotBeError(t, deleteTestEntries(s))

	t.Run("valid", func(t *testing.T) {
		should.NotBeError(t, s.Insert(user, user.UserName, userTable))
	})
	t.Run("exists", func(t *testing.T) {
		should.BeErrorIs(t, s.Insert(user, user.UserName, userTable), boltdb.ErrExists)
	})
	t.Run("invalid path", func(t *testing.T) {
		should.BeErrorIs(t, s.Insert(user, user.UserName, invalidTable), boltdb.ErrInvalidPath)
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
		value, err := s.Get[User]("first", invalidTable)
		should.BeEqual(t, value, User{})
		should.BeErrorIs(t, err, boltdb.ErrInvalidPath)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := s.Get[User]("first", userTable)
		should.BeEqual(t, value, User{})
		should.BeErrorIs(t, err, boltdb.ErrNoResults)
	})
	createTestEntries(t, s)
	t.Run("wrongkey", func(t *testing.T) {
		value, err := s.Get[User]("third", userTable)
		should.BeErrorIs(t, err, boltdb.ErrNoResults)
		should.BeEqual(t, value, User{})
	})
	t.Run("wrongType", func(t *testing.T) {
		value, err := s.Get[Other]("first", userTable)
		should.NotBeError(t, err)
		should.BeEqual(t, value, Other{})
	})
	t.Run("valid", func(t *testing.T) {
		value, err := s.Get[User]("first", userTable)
		should.NotBeError(t, err)
		should.BeEqual(t, value.UserName, "first")
		should.BeEqual(t, value.Password, "password")
	})
}

func TestGetAll(t *testing.T) {
	s, err := testInit()
	should.BeNil(t, err)
	err = deleteTestEntries(s)
	should.NotBeError(t, err)
	t.Run("noSuchTable", func(t *testing.T) {
		value, err := s.GetAll[User](invalidTable)
		should.BeEmpty(t, value)
		should.BeErrorIs(t, err, boltdb.ErrInvalidPath)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := s.GetAll[User](userTable)
		should.BeEmpty(t, value)
		should.NotBeError(t, err)
	})
	createTestEntries(t, s)
	t.Run("wrongtype", func(t *testing.T) {
		value, err := s.GetAll[string](userTable)
		should.BeError(t, err)
		should.BeEmpty(t, value)
	})
	t.Run("valid", func(t *testing.T) {
		value, err := s.GetAll[User](userTable)
		should.NotBeError(t, err)
		should.BeEqual(t, value[0].UserName, "first")
		should.BeEqual(t, value[0].Password, "password")
	})
	// deleteTestEntries(t)
	should.NotBeError(t, s.Close())
}

func TestUpdate(t *testing.T) {
	s, err := testInit()
	should.NotBeError(t, err)
	should.NotBeError(t, deleteTestEntries(s))
	t.Run("does not exist", func(t *testing.T) {
		user := User{
			UserName: testUser,
		}
		err := s.Update(user, user.UserName, userTable)
		should.BeErrorIs(t, err, boltdb.ErrExists)
	})
	t.Run("existing", func(t *testing.T) {
		user := User{
			UserName: testUser,
		}
		should.NotBeError(t, s.Save(user, user.UserName, userTable))
		user2 := User{
			UserName: "test2",
			Password: "nopass",
		}
		should.NotBeError(t, s.Update(user2, user.UserName, userTable))
		user, err = s.Get[User](user.UserName, userTable)
		should.NotBeError(t, err)
		should.BeEqual(t, user.UserName, user2.UserName)
	})
	// deleteTestEntries(t)
	should.NotBeError(t, s.Close())
}

func TestDelete(t *testing.T) {
	s, err := testInit()
	should.NotBeError(t, err)
	should.NotBeError(t, deleteTestEntries(s))
	t.Run("nonexistentTable", func(t *testing.T) {
		should.BeErrorIs(t, s.Delete[User]("first", invalidTable), boltdb.ErrInvalidPath)
	})
	t.Run("nosuchrecord", func(t *testing.T) {
		should.BeErrorIs(t, s.Delete[User]("first", userTable), boltdb.ErrNoResults)
	})
	t.Run("valid", func(t *testing.T) {
		createTestEntries(t, s)
		should.NotBeError(t, s.Delete[User]("first", userTable))
	})
	// deleteTestEntries(t)
	should.NotBeError(t, s.Close())
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
		should.NotBeError(t, s.Save(user, user.UserName, userTable))
	}
}

func deleteTestEntries(s *boltdb.Store) error {
	// t.Helper()
	values, err := s.GetAll[User](userTable)
	if err != nil {
		if errors.Is(err, boltdb.ErrNoResults) {
			return nil
		}
		return err
	}
	for _, value := range values {
		if err := s.Delete[User](value.UserName, userTable); err != nil {
			if errors.Is(err, boltdb.ErrNoResults) {
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
