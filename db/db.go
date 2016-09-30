package db

import (
	"fmt"

	"github.com/percona/pt-mongodb-summary/proto"
	"github.com/pkg/errors"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type DB struct {
	host      string
	connected bool
	session   *mgo.Session
}

type OplogEntry struct {
	Name    string
	Options struct {
		Capped      bool
		Size        int64
		AutoIndexId bool
	}
}

var NOT_CONNECTED = errors.New("not connected")

type MongoConnector interface {
	BuildInfo() (mgo.BuildInfo, error)
	Close()
	CollectionNames(dbname string) ([]string, error)
	Connect() error
	DatabaseNames() ([]string, error)
	DbRun(string, interface{}, interface{}) error
	FindOne(dbname string, collection string, query interface{}, sort []string, result interface{}) error
	GetCmdLineOpts() (proto.CommandLineOptions, error)
	GetCurrentOp() (proto.CurrentOp, error)
	GetOplogCollection() (string, error)
	GetOplogEntry(string) (*OplogEntry, error)
	HostInfo() (proto.HostInfo, error)
	IsMaster() (proto.MasterDoc, error)
	ReplicaSetGetStatus() (proto.ReplicaSetStatus, error)
	RolesCount() (int, error)
	ServerStatus() (proto.ServerStatus, error)
	Session() *mgo.Session
	SessionRun(interface{}, interface{}) error
	UsersCount() (int, error)

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
	if m.session != nil {
		m.session.Close()
	}
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

func (m *DB) FindOne(dbname string, collection string, query interface{}, sort []string, result interface{}) error {
	db := m.session.DB(dbname)
	col := db.C(collection)

	err := col.Find(query).Sort(sort...).One(result)
	if err != nil {
		return err
	}
	return nil
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

func (m *DB) GetOplogCollection() (string, error) {
	oplog := "oplog.rs"

	db := m.session.DB("local")
	nsCol := db.C("system.namespaces")

	var res interface{}
	if err := nsCol.Find(bson.M{"name": "local." + oplog}).One(&res); err == nil {
		return oplog, nil
	}

	oplog = "oplog.$main"
	if err := nsCol.Find(bson.M{"name": "local." + oplog}).One(&res); err == nil {
		return oplog, nil
	}

	return "", fmt.Errorf("neither master/slave nor replica set replication detected")
}

func (m *DB) GetOplogEntry(oplogCol string) (*OplogEntry, error) {
	db := m.session.DB("local")
	nsCol := db.C("system.namespaces")
	olEntry := &OplogEntry{}

	err := nsCol.Find(bson.M{"name": "local." + oplogCol}).One(&olEntry)
	if err != nil {
		return nil, fmt.Errorf("local.%s, or its options, not found in system.namespaces collection", oplogCol)
	}
	return olEntry, nil
}

func (m *DB) DbRun(dbName string, cmd interface{}, result interface{}) error {
	db := m.session.DB(dbName)
	err := db.Run(cmd, result)
	if err != nil {
		return errors.Wrapf(err, "cannot run cmd on db %s", dbName)
	}
	return nil
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

func (m *DB) RolesCount() (int, error) {
	return m.session.DB("admin").C("system.roles").Count()
}

func (m *DB) ServerStatus() (proto.ServerStatus, error) {
	stat := proto.ServerStatus{}
	err := m.session.DB("admin").Run(bson.D{{"serverStatus", 1}, {"recordStats", 1}}, &stat)
	if err != nil {
		return stat, errors.Wrap(err, "cannot get server status")
	}
	return stat, nil
}

func (m *DB) Session() *mgo.Session {
	return m.session
}

func (m *DB) SessionRun(cmd interface{}, result interface{}) error {
	return m.session.Run(cmd, result)
}

func (m *DB) UsersCount() (int, error) {
	return m.session.DB("admin").C("system.users").Count()
}

//TODO: do we need these functions?

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
