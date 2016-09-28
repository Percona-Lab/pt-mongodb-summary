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
	conn.(*mock.DB).Expect("GetCurrentOp", nil, rootDir+"/test/sample/currentop.json", nil)

	expect := int64(3)
	got, err := countCurrentOps(conn)
	if err != nil {
		t.Errorf("cannot get current ops: %s", err)
	}
	if got != expect {
		t.Errorf("invalid current ops count. Expected %d, got %d", expect, got)
	}
}

func TestGetSecuritySettings(t *testing.T) {

	conn := mock.NewMongoMockConnector("some fake host")
	conn.Connect()
	defer conn.Close()

	conn.(*mock.DB).Expect("GetCmdLineOpts", nil, rootDir+"/test/sample/cmdopts.json", nil)

	s, err := getSecuritySettings(conn)

	if err != nil {
		t.Errorf("error getting security settings: %s", err.Error())
	}

	if s.Users != 1 {
		t.Error("invalid users count")
	}

	if s.Roles != 1 {
		t.Error("invalid roles count")
	}

	if s.Auth != "enabled" {
		t.Error("auth should be enabled")
	}

	if s.SSL != "requireSSL" {
		t.Errorf("invalid SSL settings. Got %s, expected %s", s.SSL, "requireSSL")
	}

}
