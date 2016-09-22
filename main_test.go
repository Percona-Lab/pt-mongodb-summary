package main

import (
	"testing"

	"github.com/percona/pt-mongodb-summary/proto"

	mgo "gopkg.in/mgo.v2"
)

var session *mgo.Session

//func TestMain(m *testing.M) {
//	var err error
//	host := "localhost:17002"
//	session, err = mgo.Dial(host)
//	if err != nil {
//		panic(err)
//	}
//	defer session.Close()
//
//	v := m.Run()
//	os.Exit(v)
//}

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
