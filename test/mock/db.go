package mock

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	mgo "labix.org/v2/mgo"

	"github.com/percona/pt-mongodb-summary/db"
	"github.com/percona/pt-mongodb-summary/proto"
)

/*
responses is a map of maps because I want to have several possible answers for each method
The idea is to be able to mock different responses for some methods
Example:

responses["CollectionNames"] -> ["dbname1"]: "file1.json",
                                ["dbname2"]: "file2.json",

		 ["SessionRun"]      -> [bson.M{"a command here": 1}]: "file3.json",

*/
type DB struct {
	host      string
	session   *mgo.Session
	responses map[string]map[interface{}]string
}

type mockResponses struct {
	Expect   interface{}
	Response string // json file having the response
}

func NewMongoMockConnector(host string) db.MongoConnector {
	db := &DB{
		host:      host,
		responses: make(map[string]map[interface{}]string),
	}
	return db
}

func (m *DB) Expect(method string, expect interface{}, file string, retValues ...interface{}) {
	if _, ok := m.responses[method]; !ok {
		m.responses[method] = make(map[interface{}]string)
	}
	m.responses[method][expect] = file
}

func (m *DB) BuildInfo() (mgo.BuildInfo, error) {
	return mgo.BuildInfo{
		Version:       "percona-test",
		VersionArray:  []int{1, 2, 3},
		GitVersion:    "1.0.1",
		SysInfo:       "sysinfo",
		Bits:          0,
		Debug:         false,
		MaxObjectSize: 1,
	}, nil
}

func (m *DB) Connect() error {
	return nil
}

func (m *DB) Close() {
	//
}

func (m *DB) CollectionNames(dbname string) ([]string, error) {
	return []string{"col1", "col2", "col3"}, nil
}

func (m *DB) ConnectionPoolStats() (interface{}, error) {
	var stats interface{}
	return stats, nil
}

func (m *DB) DatabaseNames() ([]string, error) {
	return []string{"db1", "db2"}, nil
}

func (m *DB) GetCmdLineOpts() (proto.CommandLineOptions, error) {
	clo := proto.CommandLineOptions{}
	err := m.returnExpect("GetCmdLineOpts", nil, &clo)
	return clo, err
}

func (m *DB) GetCurrentOp() (proto.CurrentOp, error) {
	co := proto.CurrentOp{}
	var err error

	err = m.returnExpect("GetCurrentOp", nil, &co)
	return co, err
}

func (m *DB) GetReplicaSetStatus() (proto.ReplicaSetStatus, error) {
	rss := proto.ReplicaSetStatus{}
	return rss, nil
}

func (m *DB) HostInfo() (proto.HostInfo, error) {
	hi := proto.HostInfo{}
	return hi, nil
}

func (m *DB) IsMaster() (proto.MasterDoc, error) {
	md := proto.MasterDoc{}
	return md, nil
}

func (m *DB) ReplicaSetGetStatus() (proto.ReplicaSetStatus, error) {
	rss := proto.ReplicaSetStatus{}
	return rss, nil
}

func (m *DB) RolesCount() (int, error) {
	return 1, nil
}

func (m *DB) ServerStatus() (proto.ServerStatus, error) {
	ss := proto.ServerStatus{}
	return ss, nil
}

func (m *DB) Session() *mgo.Session {
	return m.session
}

func (m *DB) SessionRun(cmd interface{}, result interface{}) error {
	return nil
}

func (m *DB) ShardConnectionPoolStats() (interface{}, error) {
	var stats interface{}
	return stats, nil
}

func (m *DB) UsersCount() (int, error) {
	return 1, nil
}

/*
   Internal methods
*/

func (m *DB) returnExpect(method string, expect interface{}, dest interface{}) error {
	response, ok := m.responses[method]
	if !ok {
		return fmt.Errorf("cannot find response for %s", method)
	}
	filename, ok := response[expect]
	if !ok {
		return fmt.Errorf("there is no response defined for '%#v'", expect)
	}

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(file, &dest)
	if err != nil {
		return err
	}
	return nil
}
