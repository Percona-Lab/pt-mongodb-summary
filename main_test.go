package main

import (
	"os"
	"testing"

	"github.com/go-test/test"
	"github.com/percona/pt-mongodb-summary/db"
	"github.com/percona/pt-mongodb-summary/proto"
	"github.com/percona/pt-mongodb-summary/test/mock"
)

var conn db.MongoConnector
var rootDir string

func TestMain(m *testing.M) {

	rootDir = test.RootDir()
	conn = mock.NewMongoMockConnector("some hostname")
	code := m.Run()
	os.Exit(code)

}

func TestGetNodeType(t *testing.T) {
	md := []struct {
		in  proto.MasterDoc
		out string
	}{
		{proto.MasterDoc{SetName: "name"}, "replset"},
		{proto.MasterDoc{Msg: "isdbgrid"}, "mongos"},
		{proto.MasterDoc{Msg: "a msg"}, "mongod"},
	}

	for _, i := range md {
		nodeType := GetNodeType(i.in)
		if nodeType != i.out {
			t.Errorf("invalid node type. got %s, expected %s\n", nodeType, i.out)
		}
	}
}

func TestGetCurrentOps(t *testing.T) {
	conn.(*mock.DB).Expect("GetCurrentOp", nil, rootDir+"/test/sample/currentop.json")

	expect := int64(3)
	got, err := countCurrentOps(conn)
	if err != nil {
		t.Error("cannot get current ops: %s", err)
	}
	if got != expect {
		t.Errorf("invalid current ops count. Expected %d, got %d", expect, got)
	}
}
