package mock

import (
	"encoding/json"
	"io/ioutil"

	"github.com/percona/pt-mongodb-summary/db"
	"github.com/percona/pt-mongodb-summary/proto"

	mgo "gopkg.in/mgo.v2"
)

type DB struct {
	host        string
	session     *mgo.Session
	answerFiles []string
	pos         int // answer file index
}

func NewMongoMockConnector(host string) db.MongoConnector {
	db := &DB{
		host:        host,
		answerFiles: []string{},
		pos:         -1,
	}
	return db
}

func (m *DB) Connect() error {
	return nil
}
func (m *DB) Close() {
	//
}

func (m *DB) GetCurrentOp() (proto.CurrentOp, error) {
	co := proto.CurrentOp{}

	err := m.unmarshalNext(&co)
	if err != nil {
		return co, err
	}
	return co, nil
}

func (m *DB) BuildInfo() (mgo.BuildInfo, error) {
	return mgo.BuildInfo{
		Version:        "percona-test",
		VersionArray:   []int{1, 2, 3},
		GitVersion:     "1.0.1",
		OpenSSLVersion: "",
		Bits:           0,
		Debug:          false,
		MaxObjectSize:  1,
	}, nil
}

func (m *DB) DatabaseNames() ([]string, error) {
	return []string{"db1", "db2"}, nil
}

func (m *DB) GetReplicaSetStatus() proto.ReplicaSetStatus {
	return nil
}

func (m *DB) AddResponseFile(file string) {
	m.answerFiles = append(m.answerFiles, file)
}

func (m *DB) unmarshalNext(dest interface{}) error {
	if m.pos+1 < len(m.answerFiles) {
		m.pos++
	}
	return m.unmarshal(m.answerFiles[m.pos], &dest)
}

func (m *DB) unmarshal(filename string, dest interface{}) error {
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
