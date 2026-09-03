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

var testBuckets = []boltdb.Path{
	userBucket, keysBucket, tablesBucket, nestedBucket,
}

var (
	userBucket        = boltdb.Path{"users"}
	invalidBucket     = boltdb.Path{"invalidtablename"}
	nestedBucket      = boltdb.Path{"nested", "one", "two"}
	tablesBucket      = boltdb.Path{"tables"}
	keysBucket        = boltdb.Path{"keys"}
	emptyBucket       = boltdb.Path{}
	emptyNestedBucket = boltdb.Path{"foo", "", "bar"}
	userKeyFirst      = boltdb.Path{"users", "first"}
	userKeySecond     = boltdb.Path{"users", "second"}
	userKeyNested     = boltdb.Path{"nested", "one", "key"}
)

func TestInitialize(t *testing.T) {
	t.Log(testBuckets)
	t.Run("valid", func(t *testing.T) {
		s := testInit(t)
		should.NotBeError(t, s.Close())
	})
	t.Run("open", func(t *testing.T) {
		s, err := boltdb.Open("test.db")
		should.NotBeError(t, err)
		_, err = boltdb.Open("test.db")
		should.BeErrorIs(t, err, berror.ErrTimeout)
		should.NotBeError(t, s.Close())
	})
	t.Run("pathDoesNotExist", func(t *testing.T) {
		var pathError *fs.PathError
		_, err := boltdb.Initialize("/tmp/thispathdoesnotexist/test.db", testBuckets)
		should.BeTrue(t, errors.As(err, &pathError))
	})
}

func TestCreateBucket(t *testing.T) {
	s := testInit(t)
	t.Run("invalid path", func(t *testing.T) {
		should.BeErrorIs(t, s.CreateBucket(emptyBucket), boltdb.ErrInvalidPath)
	})
	t.Run("invalid nested", func(t *testing.T) {
		should.BeErrorIs(t, s.CreateBucket(emptyNestedBucket), berror.ErrBucketNameRequired)
	})
	t.Run("valid", func(t *testing.T) {
		should.NotBeError(t, s.CreateBucket(userBucket))
	})
	t.Run("valid", func(t *testing.T) {
		should.NotBeError(t, s.CreateBucket(nestedBucket))
	})
}

func TestCopyBucket(t *testing.T) {
	s := testInit(t)
	t.Run("non-existent src path", func(t *testing.T) {
		should.BeErrorIs(t, s.CopyBucket(invalidBucket, keysBucket), boltdb.ErrInvalidPath)
	})
	t.Run("empty src path", func(t *testing.T) {
		should.BeErrorIs(t, s.CopyBucket(emptyBucket, keysBucket), boltdb.ErrInvalidPath)
	})
	t.Run("empty dest path", func(t *testing.T) {
		should.BeErrorIs(t, s.CopyBucket(userBucket, emptyNestedBucket),
			berror.ErrBucketNameRequired)
	})
	t.Run("root to root", func(t *testing.T) {
		should.NotBeError(t, s.CopyBucket(userBucket, keysBucket))
	})
	t.Run("root to nested", func(t *testing.T) {
		should.NotBeError(t, s.CopyBucket(userBucket, nestedBucket))
	})
	t.Run("nested to root", func(t *testing.T) {
		should.NotBeError(t, s.Insert("hello world", "key", nestedBucket))
		should.NotBeError(t, s.CopyBucket(nestedBucket, keysBucket))
	})
}

func TestCopyKey(t *testing.T) {
	s := testInit(t)
	createTestEntries(t, s)
	t.Run("invalid src path", func(t *testing.T) {
		should.BeErrorIs(t, s.CopyKey(emptyBucket, keysBucket), boltdb.ErrInvalidPath)
	})
	t.Run("invalid dest path", func(t *testing.T) {
		should.BeErrorIs(t, s.CopyKey(userKeyFirst, boltdb.Path{"nested", "missing", "two"}),
			boltdb.ErrInvalidPath)
	})
	t.Run("copy to bucket", func(t *testing.T) {
		should.BeErrorIs(t, s.CopyKey(userKeyFirst, nestedBucket), berror.ErrIncompatibleValue)
	})
	t.Run("copy to self", func(t *testing.T) {
		should.BeErrorIs(t, s.CopyKey(userKeyFirst, userKeyFirst), boltdb.ErrExists)
	})
	t.Run("copy to existing", func(t *testing.T) {
		should.BeErrorIs(t, s.CopyKey(userKeyFirst, userKeySecond), boltdb.ErrExists)
	})
	t.Run("copy bucket", func(t *testing.T) {
		should.BeErrorIs(t, s.CopyKey(nestedBucket, userKeyFirst), boltdb.ErrNoResults)
	})
	t.Run("valid", func(t *testing.T) {
		should.NotBeError(t, s.CopyKey(userKeyFirst, userKeyNested))
		// key
	})
}

func TestClose(t *testing.T) {
	s := testInit(t)
	t.Run("open", func(t *testing.T) {
		should.NotBeError(t, s.Close())
	})
	t.Run("closed", func(t *testing.T) {
		err := s.Close()
		should.BeNil(t, err)
	})
}

func TestDeleteBucket(t *testing.T) {
	s := testInit(t)
	t.Run("empty path", func(t *testing.T) {
		should.BeErrorIs(t, s.DeleteBucket(emptyBucket), boltdb.ErrInvalidPath)
	})
	t.Run("invalid path", func(t *testing.T) {
		should.BeErrorIs(t, s.DeleteBucket(emptyNestedBucket), boltdb.ErrInvalidPath)
	})
	t.Run("non-existent bucket", func(t *testing.T) {
		should.BeErrorIs(t, s.DeleteBucket(invalidBucket), berror.ErrBucketNotFound)
	})
	t.Run("root bucket", func(t *testing.T) {
		should.NotBeError(t, s.DeleteBucket(userBucket))
	})
	t.Run("nexted bucket", func(t *testing.T) {
		should.NotBeError(t, s.DeleteBucket(nestedBucket))
	})
}

func TestEmptyBucket(t *testing.T) {
	s := testInit(t)
	t.Run("empty path", func(t *testing.T) {
		should.BeErrorIs(t, s.EmptyBucket(emptyBucket), boltdb.ErrInvalidPath)
	})
	t.Run("non-existent bucket", func(t *testing.T) {
		should.BeErrorIs(t, s.EmptyBucket(invalidBucket), boltdb.ErrInvalidPath)
	})
	t.Run("root bucket", func(t *testing.T) {
		should.NotBeError(t, s.EmptyBucket(userBucket))
	})
	t.Run("nested bucket", func(t *testing.T) {
		bucket := boltdb.Path{"nested", "one"}
		should.NotBeError(t, s.Insert("value", "key", bucket))
		should.NotBeError(t, s.EmptyBucket(bucket))
	})
}

func TestMoveBucket(t *testing.T) {
	s := testInit(t)
	createTestEntries(t, s)

	t.Run("non-existent src path", func(t *testing.T) {
		should.BeErrorIs(t, s.MoveBucket(invalidBucket, keysBucket), boltdb.ErrInvalidPath)
	})
	t.Run("empty src path", func(t *testing.T) {
		should.BeErrorIs(t, s.MoveBucket(emptyBucket, keysBucket), boltdb.ErrInvalidPath)
	})
	t.Run("root to root", func(t *testing.T) {
		should.NotBeError(t, s.MoveBucket(userBucket, keysBucket))
	})
	t.Run("empty dest path", func(t *testing.T) {
		should.BeErrorIs(t, s.MoveBucket(userBucket, emptyNestedBucket), boltdb.ErrInvalidPath)
	})
	t.Run("root to nested", func(t *testing.T) {
		should.NotBeError(t, s.MoveBucket(tablesBucket, nestedBucket))
	})
	t.Run("nested to root", func(t *testing.T) {
		should.NotBeError(t, s.Insert("hello world", "key", nestedBucket))
		should.NotBeError(t, s.MoveBucket(nestedBucket, keysBucket))
	})
	t.Run("transaction error", func(t *testing.T) {
		should.NotBeError(t, s.Close())
		should.BeErrorIs(t, s.MoveBucket(keysBucket, nestedBucket), berror.ErrDatabaseNotOpen)
	})
}

func TestMoveKey(t *testing.T) {
	s := testInit(t)
	createTestEntries(t, s)
	t.Run("non-existent src path", func(t *testing.T) {
		should.BeErrorIs(t, s.MoveKey(invalidBucket, keysBucket), boltdb.ErrInvalidPath)
	})
	t.Run("empty src path", func(t *testing.T) {
		should.BeErrorIs(t, s.MoveKey(emptyBucket, keysBucket), boltdb.ErrInvalidPath)
	})
	t.Run("src is bucket", func(t *testing.T) {
		should.BeErrorIs(t, s.MoveKey(userBucket, userKeyNested), boltdb.ErrInvalidPath)
	})
	t.Run("to bucket", func(t *testing.T) {
		should.BeErrorIs(t, s.MoveKey(userKeyFirst, nestedBucket), berror.ErrIncompatibleValue)
	})
	t.Run("empty dest path", func(t *testing.T) {
		should.BeErrorIs(t, s.MoveKey(userKeyFirst, emptyNestedBucket), boltdb.ErrInvalidPath)
	})
	t.Run("to self", func(t *testing.T) {
		should.BeErrorIs(t, s.MoveKey(userKeyFirst, userKeyFirst), boltdb.ErrExists)
	})
	t.Run("to existing key", func(t *testing.T) {
		should.BeErrorIs(t, s.MoveKey(userKeyFirst, userKeySecond), boltdb.ErrExists)
	})
	t.Run("root to nested", func(t *testing.T) {
		should.NotBeError(t, s.MoveKey(userKeySecond, userKeyNested))
	})
	t.Run("nested to root", func(t *testing.T) {
		should.NotBeError(t, s.Insert("value", "hello world", nestedBucket))
		should.NotBeError(t, s.MoveKey(append(nestedBucket, "hello world"), userKeySecond))
	})
	t.Run("transaction error", func(t *testing.T) {
		should.NotBeError(t, s.Close())
		should.BeErrorIs(t, s.MoveKey(userKeyFirst, nestedBucket), berror.ErrDatabaseNotOpen)
	})
}

func TestRenameBucket(t *testing.T) {
	s := testInit(t)
	t.Run("no bucket", func(t *testing.T) {
		should.BeErrorIs(t, s.RenameBucket(emptyBucket, "newBucket"), boltdb.ErrInvalidPath)
	})
	// Investigate and Fix.
	// t.Run("key", func(t *testing.T) {
	// createTestEntries(t, s)
	// should.BeErrorIs(t, s.RenameBucket(userKeyFirst, "newBucket"), boltdb.ErrInvalidPath)
	// })
	t.Run("valid", func(t *testing.T) {
		should.NotBeError(t, s.RenameBucket(userBucket, "hello"))
	})
}

func TestRenameKey(t *testing.T) {
	s := testInit(t)
	createTestEntries(t, s)
	t.Run("key does not exist", func(t *testing.T) {
		should.BeErrorIs(t, s.RenameKey(boltdb.Path{"users", "third"}, "new name"),
			boltdb.ErrNoResults)
	})
	t.Run("bucket does not exist", func(t *testing.T) {
		should.BeErrorIs(t, s.RenameKey(boltdb.Path{"missing", "key"}, "new name"),
			boltdb.ErrInvalidPath)
	})
	t.Run("invalid path", func(t *testing.T) {
		should.BeErrorIs(t, s.RenameKey(tablesBucket, "new name"), boltdb.ErrInvalidPath)
	})
	t.Run("vaid", func(t *testing.T) {
		should.NotBeError(t, s.RenameKey(userKeyFirst, "new name"))
	})
}

func TestSave(t *testing.T) {
	user := User{
		UserName: testUser,
	}
	s := testInit(t)
	t.Run("noSuchTable", func(t *testing.T) {
		should.BeErrorIs(t, s.Save("testing", "key", invalidBucket), boltdb.ErrInvalidPath)
	})
	t.Run("invalidjson", func(t *testing.T) {
		function := func() {}
		value := struct {
			Function func()
		}{
			Function: function,
		}
		should.BeError(t, s.Save(value, "hello", userBucket))
	})
	t.Run("emptyTable", func(t *testing.T) {
		should.BeErrorIs(t, s.Save("value", "key", emptyBucket), boltdb.ErrInvalidPath)
	})
	t.Run("emptyNestedTable", func(t *testing.T) {
		should.BeErrorIs(t, s.Save("value", "key", emptyNestedBucket), boltdb.ErrInvalidPath)
	})
	t.Run("dest key name is bucket", func(t *testing.T) {
		should.BeErrorIs(t, s.Save("value", "two", boltdb.Path{"nested", "one"}),
			berror.ErrIncompatibleValue)
	})
	t.Run("valid", func(t *testing.T) {
		should.NotBeError(t, s.Save(user, user.UserName, userBucket))
	})
	t.Run("nested", func(t *testing.T) {
		should.NotBeError(t, s.Save(user, user.UserName, nestedBucket))
	})
	should.NotBeError(t, deleteTestEntries(s))
	should.NotBeError(t, s.Close())
}

func TestInsert(t *testing.T) {
	user := User{
		UserName: testUser,
	}
	s := testInit(t)
	should.NotBeError(t, deleteTestEntries(s))

	t.Run("valid", func(t *testing.T) {
		should.NotBeError(t, s.Insert(user, user.UserName, userBucket))
	})
	t.Run("name with space", func(t *testing.T) {
		should.NotBeError(t, s.Insert("value", "hello world", userBucket))
	})
	t.Run("exists", func(t *testing.T) {
		should.BeErrorIs(t, s.Insert(user, user.UserName, userBucket), boltdb.ErrExists)
	})
	t.Run("invalid path", func(t *testing.T) {
		should.BeErrorIs(t, s.Insert(user, user.UserName, invalidBucket), boltdb.ErrInvalidPath)
	})
	// deleteTestEntries(t)
	should.NotBeError(t, s.Close())
}

func TestGetOne(t *testing.T) {
	// err := Initialize("./test.db", tables)
	s := testInit(t)
	should.NotBeError(t, deleteTestEntries(s))
	t.Run("noSuchTable", func(t *testing.T) {
		value, err := s.Get[User]("first", invalidBucket)
		should.BeEqual(t, value, User{})
		should.BeErrorIs(t, err, boltdb.ErrInvalidPath)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := s.Get[User]("first", userBucket)
		should.BeEqual(t, value, User{})
		should.BeErrorIs(t, err, boltdb.ErrNoResults)
	})
	createTestEntries(t, s)
	t.Run("wrongkey", func(t *testing.T) {
		value, err := s.Get[User]("third", userBucket)
		should.BeErrorIs(t, err, boltdb.ErrNoResults)
		should.BeEqual(t, value, User{})
	})
	t.Run("wrongType", func(t *testing.T) {
		value, err := s.Get[Other]("first", userBucket)
		should.NotBeError(t, err)
		should.BeEqual(t, value, Other{})
	})
	t.Run("valid", func(t *testing.T) {
		value, err := s.Get[User]("first", userBucket)
		should.NotBeError(t, err)
		should.BeEqual(t, value.UserName, "first")
		should.BeEqual(t, value.Password, "password")
	})
}

func TestGetAll(t *testing.T) {
	s := testInit(t)
	should.NotBeError(t, deleteTestEntries(s))
	t.Run("noSuchTable", func(t *testing.T) {
		value, err := s.GetAll[User](invalidBucket)
		should.BeEmpty(t, value)
		should.BeErrorIs(t, err, boltdb.ErrInvalidPath)
	})
	t.Run("noValues", func(t *testing.T) {
		value, err := s.GetAll[User](userBucket)
		should.BeEmpty(t, value)
		should.NotBeError(t, err)
	})
	createTestEntries(t, s)
	t.Run("wrongtype", func(t *testing.T) {
		value, err := s.GetAll[string](userBucket)
		should.BeError(t, err)
		should.BeEmpty(t, value)
	})
	t.Run("valid", func(t *testing.T) {
		value, err := s.GetAll[User](userBucket)
		should.NotBeError(t, err)
		should.BeEqual(t, value[0].UserName, "first")
		should.BeEqual(t, value[0].Password, "password")
	})
	// deleteTestEntries(t)
	should.NotBeError(t, s.Close())
}

func TestUpdate(t *testing.T) {
	s := testInit(t)
	should.NotBeError(t, deleteTestEntries(s))
	t.Run("key does not exist", func(t *testing.T) {
		err := s.Update("new value", "missing", userBucket)
		should.BeErrorIs(t, err, boltdb.ErrNoResults)
	})
	t.Run("bucket does not exist", func(t *testing.T) {
		should.BeErrorIs(t, s.Update("new value", "keyname", emptyBucket), boltdb.ErrInvalidPath)
	})
	createTestEntries(t, s)
	t.Run("existing", func(t *testing.T) {
		should.NotBeError(t, s.Update("new value", "second", userBucket))
	})
}

func TestDelete(t *testing.T) {
	s := testInit(t)
	should.NotBeError(t, deleteTestEntries(s))
	t.Run("nonexistentTable", func(t *testing.T) {
		should.BeErrorIs(t, s.Delete("first", invalidBucket), boltdb.ErrInvalidPath)
	})
	t.Run("nosuchrecord", func(t *testing.T) {
		should.BeErrorIs(t, s.Delete("first", userBucket), boltdb.ErrNoResults)
	})
	t.Run("valid", func(t *testing.T) {
		createTestEntries(t, s)
		should.NotBeError(t, s.Delete("first", userBucket))
	})
	t.Run("nested", func(t *testing.T) {
		should.NotBeError(t, s.Save("value", "key", nestedBucket))
		should.NotBeError(t, s.Delete("key", nestedBucket))
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
		should.NotBeError(t, s.Save(user, user.UserName, userBucket))
	}
}

func deleteTestEntries(s *boltdb.Store) error {
	values, err := s.GetAll[User](userBucket)
	if err != nil {
		if errors.Is(err, boltdb.ErrNoResults) {
			return nil
		}
		return err
	}
	for _, value := range values {
		if err := s.Delete(value.UserName, userBucket); err != nil {
			if errors.Is(err, boltdb.ErrNoResults) {
				return nil
			}
			return err
		}
	}
	return nil
}

func testInit(t *testing.T) *boltdb.Store {
	t.Helper()
	if err := os.Remove("./test.db"); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			t.FailNow()
		}
	}
	store, err := boltdb.Initialize("./test.db", testBuckets)
	if err != nil {
		t.FailNow()
	}
	return store
}
