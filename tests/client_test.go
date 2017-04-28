package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/abci/server"
	"github.com/tendermint/merkleeyes/app"
	eyes "github.com/tendermint/merkleeyes/client"
	. "github.com/tendermint/tmlibs/common"
)

var tmspAddr = "tcp://127.0.0.1:46659"

func TestNonPersistent(t *testing.T) {
	testProcedure(t, tmspAddr, "", 0, false, true)
}

func TestPersistent(t *testing.T) {
	dbName := "testDb"
	os.RemoveAll(dbName) //remove the database if exists for any reason
	testProcedure(t, tmspAddr, dbName, 0, false, false)
	testProcedure(t, tmspAddr, dbName, 0, true, true)
	os.RemoveAll(dbName) //cleanup, remove database that was created by testProcedure
}

func testProcedure(t *testing.T, addr, dbName string, cache int, testPersistence, clearRecords bool) {

	checkErr := func(err error) {
		if err != nil {
			t.Fatal(err.Error())
			return
		}
	}

	// Create the app and start the listener
	mApp := app.NewMerkleEyesApp(dbName, cache)
	defer mApp.CloseDB()

	s, err := server.NewServer(addr, "socket", mApp)
	checkErr(err)

	s.Start()
	defer s.Stop()

	// Create client
	cli, err := eyes.NewClient(addr)
	checkErr(err)
	cli.Start()
	defer cli.Stop()

	if !testPersistence {
		// Empty
		commit(t, cli, "")
		get(t, cli, "foo", "")
		get(t, cli, "bar", "")
		// Set foo=FOO
		set(t, cli, "foo", "FOO")

		commit(t, cli, "882E25CAC42D1C6F45EDBBCEC66B08303DF9A38E")
		get(t, cli, "foo", "FOO")
		get(t, cli, "foa", "")
		get(t, cli, "foz", "")
		rem(t, cli, "foo")

		// Not empty until commit....
		get(t, cli, "foo", "FOO")
		commit(t, cli, "")
		get(t, cli, "foo", "")

		// Set foo1, foo2, foo3...
		set(t, cli, "foo1", "1")
		set(t, cli, "foo2", "2")
		set(t, cli, "foo3", "3")
		set(t, cli, "foo1", "4")
		// nothing commited yet...
		get(t, cli, "foo1", "")
		commit(t, cli, "548A1A7FE60D0D1414E65B385E9796D99E00B316")
		// now we got info
		get(t, cli, "foo1", "4")
		get(t, cli, "foo2", "2")
		get(t, cli, "foo3", "3")
	} else {
		get(t, cli, "foo1", "4")
		get(t, cli, "foo2", "2")
		get(t, cli, "foo3", "3")
	}

	if clearRecords {
		rem(t, cli, "foo3")
		rem(t, cli, "foo2")
		rem(t, cli, "foo1")
		// Empty
		commit(t, cli, "")
	}
}

func get(t *testing.T, cli *eyes.Client, key string, value string) {
	valExp := []byte(nil)
	if value != "" {
		valExp = []byte(value)
	}
	valGot := cli.Get([]byte(key))
	require.EqualValues(t, valExp, valGot)
}

func set(t *testing.T, cli *eyes.Client, key string, value string) {
	cli.Set([]byte(key), []byte(value))
}

func rem(t *testing.T, cli *eyes.Client, key string) {
	cli.Remove([]byte(key))
}

func commit(t *testing.T, cli *eyes.Client, hash string) {
	res := cli.CommitSync()
	require.False(t, res.IsErr(), res.Error())
	assert.Equal(t, hash, Fmt("%X", res.Data.Bytes()))
}
