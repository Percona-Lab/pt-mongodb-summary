package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/percona/pt-mongodb-summary/proto"
	"github.com/percona/pt-mongodb-summary/test"
	//"github.com/percona/pt-mongodb-summary/test"
	"labix.org/v2/mgo" // mock
	"labix.org/v2/mgo/bson"
)

func TestGetHostnames(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mgo.MOCK().SetController(ctrl)

	session := &mgo.Session{}

	mockShardsInfo := proto.ShardsInfo{
		Shards: []proto.Shard{
			proto.Shard{
				ID:   "r1",
				Host: "r1/localhost:17001,localhost:17002,localhost:17003",
			},
			proto.Shard{
				ID:   "r2",
				Host: "r2/localhost:18001,localhost:18002,localhost:18003",
			},
		},
		OK: 1,
	}

	mgo.EXPECT().Dial(gomock.Any()).Return(session, nil)
	session.EXPECT().Run("listShards", gomock.Any()).SetArg(1, mockShardsInfo)
	session.EXPECT().Close()

	expect := []string{"localhost", "localhost:17001", "localhost:18001"}
	rss, err := getHostnames("localhost")
	if err != nil {
		t.Errorf("getHostnames: %v", err)
	}
	if !reflect.DeepEqual(rss, expect) {
		t.Errorf("getHostnames: got %+v, expected: %+v\n", rss, expect)
	}
}

func TestGetReplicasetMembers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mgo.MOCK().SetController(ctrl)

	session := &mgo.Session{}

	mockrss := proto.ReplicaSetStatus{
		Date:    "",
		MyState: 1,
		Term:    0,
		HeartbeatIntervalMillis: 0,
		Members: []proto.Members{
			proto.Members{
				Optime:        nil,
				OptimeDate:    "",
				InfoMessage:   "",
				Id:            0,
				Name:          "localhost:17001",
				Health:        1,
				StateStr:      "PRIMARY",
				Uptime:        113287,
				ConfigVersion: 1,
				Self:          true,
				State:         1,
				ElectionTime:  6340960613392449537,
				ElectionDate:  "",
				Set:           ""},
			proto.Members{
				Optime:        nil,
				OptimeDate:    "",
				InfoMessage:   "",
				Id:            1,
				Name:          "localhost:17002",
				Health:        1,
				StateStr:      "SECONDARY",
				Uptime:        113031,
				ConfigVersion: 1,
				Self:          false,
				State:         2,
				ElectionTime:  0,
				ElectionDate:  "",
				Set:           ""},
			proto.Members{
				Optime:        nil,
				OptimeDate:    "",
				InfoMessage:   "",
				Id:            2,
				Name:          "localhost:17003",
				Health:        1,
				StateStr:      "SECONDARY",
				Uptime:        113031,
				ConfigVersion: 1,
				Self:          false,
				State:         2,
				ElectionTime:  0,
				ElectionDate:  "",
				Set:           ""}},
		Ok:  1,
		Set: "r1",
	}
	expect := []proto.Members{
		proto.Members{
			Optime:        nil,
			OptimeDate:    "",
			InfoMessage:   "",
			Id:            0,
			Name:          "localhost:17001",
			Health:        1,
			StateStr:      "PRIMARY",
			Uptime:        113287,
			ConfigVersion: 1,
			Self:          true,
			State:         1,
			ElectionTime:  6340960613392449537,
			ElectionDate:  "",
			Set:           "r1"},
		proto.Members{Optime: (*proto.Optime)(nil),
			OptimeDate:    "",
			InfoMessage:   "",
			Id:            1,
			Name:          "localhost:17002",
			Health:        1,
			StateStr:      "SECONDARY",
			Uptime:        113031,
			ConfigVersion: 1,
			Self:          false,
			State:         2,
			ElectionTime:  0,
			ElectionDate:  "",
			Set:           "r1"},
		proto.Members{Optime: (*proto.Optime)(nil),
			OptimeDate:    "",
			InfoMessage:   "",
			Id:            2,
			Name:          "localhost:17003",
			Health:        1,
			StateStr:      "SECONDARY",
			Uptime:        113031,
			ConfigVersion: 1,
			Self:          false,
			State:         2,
			ElectionTime:  0,
			ElectionDate:  "",
			Set:           "r1",
		}}

	mgo.EXPECT().Dial(gomock.Any()).Return(session, nil)
	session.EXPECT().Run(bson.M{"replSetGetStatus": 1}, gomock.Any()).SetArg(1, mockrss)
	session.EXPECT().Close()

	rss, err := getReplicasetMembers([]string{"localhost"})
	if err != nil {
		t.Errorf("getReplicasetMembers: %v", err)
	}
	if !reflect.DeepEqual(rss, expect) {
		t.Errorf("getReplicasetMembers: got %+v, expected: %+v\n", rss, expect)
	}

}

func TestGetTemplateData(t *testing.T) {

	d := os.Getenv("BASEDIR")
	if d == "" {
		log.Printf("cannot get BASEDIR env var")
		return
	}
	d1, _ := os.Getwd()
	ioutil.WriteFile("/tmp/dat1", []byte(d1), 0644)

	shardsInfo := proto.ShardsInfo{}
	test.LoadJson(d+"/test/sample/shardsinfo.json", &shardsInfo)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mgo.MOCK().SetController(ctrl)

	session := &mgo.Session{}

	mgo.EXPECT().Dial(gomock.Any()).Return(session, nil)
	session.EXPECT().Run("listShards", gomock.Any()).SetArg(1, shardsInfo)
	session.EXPECT().Close()

	mgo.EXPECT().Dial(gomock.Any()).Return(session, nil)

	var bi mgo.BuildInfo
	test.LoadJson(d+"/test/sample/buildinfo.json", &bi)
	session.EXPECT().BuildInfo().Return(bi, nil)

	var md proto.MasterDoc
	test.LoadJson(d+"/test/sample/ismaster.json", &md)
	session.EXPECT().Run("isMaster", gomock.Any()).SetArg(1, md)

	// getReplicasetMembers
	rss0 := proto.ReplicaSetStatus{}
	rss1 := proto.ReplicaSetStatus{}
	rss2 := proto.ReplicaSetStatus{}
	test.LoadJson(d+"/test/sample/replsetgetstatus_00.json", &rss0)
	test.LoadJson(d+"/test/sample/replsetgetstatus_01.json", &rss1)
	test.LoadJson(d+"/test/sample/replsetgetstatus_02.json", &rss2)
	mgo.EXPECT().Dial(gomock.Any()).Return(session, nil)
	session.EXPECT().Run(bson.M{"replSetGetStatus": 1}, gomock.Any()).SetArg(1, rss0)
	session.EXPECT().Close()
	mgo.EXPECT().Dial(gomock.Any()).Return(session, nil)
	session.EXPECT().Run(bson.M{"replSetGetStatus": 1}, gomock.Any()).SetArg(1, rss1)
	session.EXPECT().Close()
	mgo.EXPECT().Dial(gomock.Any()).Return(session, nil)
	session.EXPECT().Run(bson.M{"replSetGetStatus": 1}, gomock.Any()).SetArg(1, rss2)
	session.EXPECT().Close()

	// serverStatus
	database := &mgo.Database{}
	//usersCol := &mgo.Collection{}
	//rolesCol := &mgo.Collection{}
	ss := proto.ServerStatus{}
	test.LoadJson(d+"/test/sample/serverstatus.json", &ss)
	session.EXPECT().DB("admin").Return(database)
	database.EXPECT().Run(bson.D{{"serverStatus", 1}, {"recordStats", 1}}, gomock.Any()).SetArg(1, ss)

	// get host info
	hi := proto.HostInfo{}
	test.LoadJson(d+"/test/sample/hostinfo.json", &hi)
	session.EXPECT().Run(bson.M{"hostInfo": 1}, gomock.Any()).SetArg(1, hi)

	// get security settings
	cmdopts := proto.CommandLineOptions{}
	test.LoadJson(d+"/test/sample/cmdopts.json", &cmdopts)
	session.EXPECT().DB("admin").Return(database)
	database.EXPECT().Run(bson.D{{"getCmdLineOpts", 1}, {"recordStats", 1}}, gomock.Any()).SetArg(1, cmdopts)

	usersCol := &mgo.Collection{}
	rolesCol := &mgo.Collection{}

	session.EXPECT().DB("admin").Return(database)
	database.EXPECT().C("system.users").Return(usersCol)
	usersCol.EXPECT().Count().Return(1, nil)

	session.EXPECT().DB("admin").Return(database)
	database.EXPECT().C("system.roles").Return(rolesCol)
	rolesCol.EXPECT().Count().Return(2, nil)

	//
	session.EXPECT().Close()

	td, err := getTemplateData("localhost")
	if err != nil {
		t.Errorf("cannot get template data: %s", err)
	}
	if td.NodeType == "" {
		t.Errorf("something was wrong. cannot get template data")
	}

}

func TestSecurityOpts(t *testing.T) {
	cmdopts := []proto.CommandLineOptions{
		// 1
		proto.CommandLineOptions{
			Parsed: proto.Parsed{
				Net: proto.Net{
					SSL: proto.SSL{
						Mode: "",
					},
				},
			},
			Security: proto.Security{
				KeyFile:       "",
				Authorization: "",
			},
		},
		// 2
		proto.CommandLineOptions{
			Parsed: proto.Parsed{
				Net: proto.Net{
					SSL: proto.SSL{
						Mode: "",
					},
				},
			},
			Security: proto.Security{
				KeyFile:       "a file",
				Authorization: "",
			},
		},
		// 3
		proto.CommandLineOptions{
			Parsed: proto.Parsed{
				Net: proto.Net{
					SSL: proto.SSL{
						Mode: "",
					},
				},
			},
			Security: proto.Security{
				KeyFile:       "",
				Authorization: "something here",
			},
		},
		// 4
		proto.CommandLineOptions{
			Parsed: proto.Parsed{
				Net: proto.Net{
					SSL: proto.SSL{
						Mode: "super secure",
					},
				},
			},
			Security: proto.Security{
				KeyFile:       "",
				Authorization: "",
			},
		},
	}

	expect := []*security{
		// 1
		&security{
			Users: 1,
			Roles: 2,
			Auth:  "disabled",
			SSL:   "disabled",
		},
		// 2
		&security{
			Users: 1,
			Roles: 2,
			Auth:  "enabled",
			SSL:   "disabled",
		},
		// 3
		&security{
			Users: 1,
			Roles: 2,
			Auth:  "enabled",
			SSL:   "disabled",
		},
		// 4
		&security{
			Users: 1,
			Roles: 2,
			Auth:  "disabled",
			SSL:   "super secure",
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mgo.MOCK().SetController(ctrl)

	session := &mgo.Session{}
	database := &mgo.Database{}
	usersCol := &mgo.Collection{}
	rolesCol := &mgo.Collection{}

	for i, cmd := range cmdopts {
		session.EXPECT().DB("admin").Return(database)
		database.EXPECT().Run(bson.D{{"getCmdLineOpts", 1}, {"recordStats", 1}}, gomock.Any()).SetArg(1, cmd)

		session.EXPECT().DB("admin").Return(database)
		database.EXPECT().C("system.users").Return(usersCol)
		usersCol.EXPECT().Count().Return(1, nil)

		session.EXPECT().DB("admin").Return(database)
		database.EXPECT().C("system.roles").Return(rolesCol)
		rolesCol.EXPECT().Count().Return(2, nil)

		got, err := getSecuritySettings(session)

		if err != nil {
			t.Errorf("cannot get sec settings: %v", err)
		}
		if !reflect.DeepEqual(got, expect[i]) {
			t.Errorf("got: %+v, expect: %+v\n", got, expect[i])
		}
	}
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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mgo.MOCK().SetController(ctrl)

	session := &mgo.Session{}
	for _, m := range md {
		session.EXPECT().Run("isMaster", gomock.Any()).SetArg(1, m.in)
		nodeType, err := getNodeType(session)
		if err != nil {
			t.Errorf("cannot get node type: %+v, error: %s\n", m.in, err)
		}
		if nodeType != m.out {
			t.Errorf("invalid node type. got %s, expect: %s\n", nodeType, m.out)
		}
	}
	session.EXPECT().Run("isMaster", gomock.Any()).Return(fmt.Errorf("some fake error"))
	nodeType, err := getNodeType(session)
	if err == nil {
		t.Errorf("error expected, got nil")
	}
	if nodeType != "" {
		t.Errorf("expected blank node type, got %s", nodeType)
	}

}
