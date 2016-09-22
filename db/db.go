package db

import (
	"github.com/percona/pt-mongodb-summary/proto"
	"github.com/pkg/errors"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type DB struct {
	host    string
	session *mgo.Session
}

type MongoSessionConnector interface {
	Run(interface{}, interface{}) error
}

type MongoConnector interface {
	BuildInfo() (mgo.BuildInfo, error)
	Close()
	CollectionNames(dbname string) ([]string, error)
	Connect() error
	DatabaseNames() ([]string, error)
	GetCmdLineOpts() (proto.CommandLineOptions, error)
	GetCurrentOp() (proto.CurrentOp, error)
	HostInfo() (proto.HostInfo, error)
	IsMaster() (proto.MasterDoc, error)
	ReplicaSetGetStatus() (proto.ReplicaSetStatus, error)
	ServerStatus() (proto.ServerStatus, error)
	Session() *mgo.Session
	SessionRun(cmd interface{}, result interface{}) error

	// TODO not really implemented. Types shouldn't be interface{}
	ConnectionPoolStats() (interface{}, error)
	ShardConnectionPoolStats() (interface{}, error)
}

func NewMongoConnector(host string) MongoConnector {
	db := &DB{
		host: host,
	}
	return db
}

func (m *DB) BuildInfo() (mgo.BuildInfo, error) {
	return m.session.BuildInfo()
}

func (m *DB) Close() {
	m.session.Close()
}

func (m *DB) CollectionNames(dbname string) ([]string, error) {
	collectionNames, err := m.session.DB(dbname).CollectionNames()
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get collection names for db %s", dbname)
	}
	return collectionNames, nil
}

func (m *DB) Connect() error {
	var err error
	m.session, err = mgo.Dial(m.host)
	if err != nil {
		return err
	}
	return nil
}

func (m *DB) DatabaseNames() ([]string, error) {
	return m.session.DatabaseNames()
}

func (m *DB) GetCmdLineOpts() (proto.CommandLineOptions, error) {
	clo := proto.CommandLineOptions{}
	err := m.session.DB("admin").Run(bson.D{{"getCmdLineOpts", 1}, {"recordStats", 1}}, &clo)
	if err != nil {
		return clo, errors.Wrap(err, "cannot get command line options")
	}
	return clo, nil
}

func (m *DB) GetCurrentOp() (proto.CurrentOp, error) {
	co := proto.CurrentOp{}

	err := m.session.DB("admin").C("$cmd.sys.inprog").Find(nil).One(&co)
	if err != nil {
		return co, err
	}
	return co, nil
}

func (m *DB) HostInfo() (proto.HostInfo, error) {
	hi := proto.HostInfo{}
	err := m.session.Run(bson.M{"hostInfo": 1}, &hi)
	if err != nil {
		return hi, errors.Wrap(err, "cannot get host info")
	}
	return hi, nil
}

func (m *DB) IsMaster() (proto.MasterDoc, error) {
	md := proto.MasterDoc{}
	err := m.session.Run("isMaster", &md)
	if err != nil {
		return md, errors.Wrap(err, "cannot get isMaster")
	}
	return md, nil
}

func (m *DB) ReplicaSetGetStatus() (proto.ReplicaSetStatus, error) {
	rss := proto.ReplicaSetStatus{}
	err := m.session.Run(bson.M{"replSetGetStatus": 1}, &rss)
	if err != nil {
		return rss, errors.Wrap(err, "cannot get ReplicaSetStatus")
	}
	return rss, nil
}

func (m *DB) ServerStatus() (proto.ServerStatus, error) {
	ss := proto.ServerStatus{}
	err := m.session.DB("admin").Run(bson.D{{"serverStatus", 1}, {"recordStats", 0}}, &ss)
	if err != nil {
		return ss, errors.Wrap(err, "cannot get server status")
	}
	return ss, nil
}

func (m *DB) Session() *mgo.Session {
	return m.session
}

func (m *DB) SessionRun(cmd interface{}, result interface{}) error {
	return m.session.Run(cmd, result)
}

func (m *DB) ConnectionPoolStats() (interface{}, error) {
	var stats interface{}
	err := m.session.Run(bson.M{"connPoolStats": 1}, &stats)
	return stats, err
}

func (m *DB) ShardConnectionPoolStats() (interface{}, error) {
	var stats interface{}
	err := m.session.Run(bson.M{"shardConnPoolStats": 1}, &stats)
	return stats, err
}
