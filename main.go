package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"time"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"github.com/percona/pt-mongodb-summary/db"
	"github.com/percona/pt-mongodb-summary/proto"
	"github.com/percona/pt-mongodb-summary/templates"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/process"
)

type options struct {
	Host     string
	User     string
	Password string
	Debug    bool
}

type procInfo struct {
	CreateTime time.Time
	Path       string
	UserName   string
}

type security struct {
	Users int
	Roles int
	Auth  string
	SSL   string
}

type timedStats struct {
	Min   int64
	Max   int64
	Total int64
	Avg   int64
}

type opCounters struct {
	Insert  timedStats
	Query   timedStats
	Update  timedStats
	Delete  timedStats
	GetMore timedStats
	Command timedStats
}

type databases struct {
	Databases []struct {
		name       string
		sizeOnDisk int64
		empty      bool
		shards     map[string]int64
	}
	TotalSize   int64 `bson:"totalSize"`
	TotalSizeMb int64 `bson:"totalSizeMb"`
	OK          bool  `bson:"ok"`
}

type templateData struct {
	BuildInfo          mgo.BuildInfo
	CommandLineOptions proto.CommandLineOptions
	HostInfo           proto.HostInfo
	ServerStatus       proto.ServerStatus
	ReplicaSetStatus   proto.ReplicaSetStatus
	NodeType           string
	ProcInfo           procInfo
	ThisHostID         int64
	ProcessCount       int64
	Security           *security
	RunningOps         opCounters
	SampleRate         time.Duration
	ReplicaMembers     []proto.Members
}

type DB struct {
	ID          string `bson:"_id"`
	Partitioned bool   `bson:"partitioned"`
	Primary     string `bson:"primary"`
}

type shardStatusCollection struct {
	LastmodEpoch bson.ObjectId `bson:"lastmodEpoch"`
	ID           string        `bson:"_id"`
	LastMod      time.Time     `bson:"lastmod"`
	Dropped      bool          `bson:"dropped"`
	Key          struct {
		ID     string `bson:"_id"`
		Unique bool   `bson:"unique"`
	}
}

type chunk struct {
	LastmodEpoch bson.ObjectId `bson:"lastmodEpoch"`
	NS           string        `bson:"ns"`
	Min          string        `bson:"min._id"`
	Max          string        `bson:"max._id"`
	Shard        string        `bson:"shard"`
	ID           string        `bson:"_id"`
	Lastmod      int64         `bson:"lastmod"`
}

var Debug = false

func main() {
	var opts options
	flag.StringVar(&opts.Host, "hosts", "localhost:27017", "List of host:port to connect to")
	flag.BoolVar(&opts.Debug, "debug", false, "debug mode")
	flag.Parse()

	templateData, err := getTemplateData(opts.Host)
	if err != nil {
		panic(err)
	}

	t := template.Must(template.New("replicas").Parse(templates.Replicas))
	t.Execute(os.Stdout, templateData)

	t = template.Must(template.New("hosttemplateData").Parse(templates.HostInfo))
	t.Execute(os.Stdout, templateData)

	t = template.Must(template.New("runningOps").Parse(templates.RunningOps))
	t.Execute(os.Stdout, templateData)

	t = template.Must(template.New("ssl").Parse(templates.Security))
	t.Execute(os.Stdout, templateData)

	//oplogInfo, err := getOplogInfo(hostnames, db.NewMongoConnector)
	//if oplogInfo != nil && len(oplogInfo) > 0 {
	//	t = template.Must(template.New("oplogInfo").Parse(templates.Oplog))
	//	t.Execute(os.Stdout, oplogInfo[0])
	//}

	//doSomething(conn.Session())
}

func getTemplateData(hostname string) (templateData, error) {
	td := templateData{}
	hostnames, err := getHostnames(hostname)
	if err != nil {
		return templateData{}, err
	}

	session, err := mgo.Dial(hostname)
	if err != nil {
		return templateData{}, err
	}
	defer session.Close()

	//
	td.BuildInfo, err = session.BuildInfo()
	if err != nil {
		return templateData{}, err
	}

	//
	td.NodeType, err = getNodeType(session)
	if err != nil {
		return templateData{}, err
	}

	//
	td.ReplicaMembers, err = getReplicasetMembers(hostnames)
	if err != nil {
		return templateData{}, err
	}

	//
	err = session.DB("admin").Run(bson.D{{"serverStatus", 1}, {"recordStats", 1}}, &td.ServerStatus)
	write("serverstatus", td.ServerStatus)
	if err != nil {
		return templateData{}, err
	}

	// Sample Running Ops
	//var sampleCount int64 = 5
	//var sampleRate time.Duration = 1 // in seconds
	//templateData.SampleRate = time.Duration(sampleCount) * time.Second * sampleRate
	//osChan := getOpCountersStats(conn, sampleCount, sampleRate*time.Second)
	//templateData.RunningOps = <-osChan

	//
	err = session.Run(bson.M{"hostInfo": 1}, &td.HostInfo)
	write("hostinfo", td.HostInfo)
	if err != nil {
		return templateData{}, err
	}

	td.Security, err = getSecuritySettings(session)

	//fillMissingInfo(conn, &templateData)

	err = getProcInfo(int32(td.ServerStatus.Pid), &td.ProcInfo)
	if err != nil {
		return templateData{}, err
	}

	return td, nil
}

func getHostnames(hostname string) ([]string, error) {

	session, err := mgo.Dial(hostname)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	shardsInfo := &proto.ShardsInfo{}
	err = session.Run("listShards", shardsInfo)
	if err != nil {
		return nil, errors.Wrap(err, "cannot list shards")
	}

	hostnames := []string{hostname}
	if shardsInfo != nil {
		for _, shardInfo := range shardsInfo.Shards {
			m := strings.Split(shardInfo.Host, "/")
			h := strings.Split(m[1], ",")
			hostnames = append(hostnames, h[0])
		}
	}
	return hostnames, nil
}

func getClusterWideInfo(newMongoConnector db.ConnectorFactory, mongods []string) {

	shardedCols := make(map[string]bool)
	unshardedCols := make(map[string]bool)

	for _, hostname := range mongods {
		conn := newMongoConnector(hostname)
		err := conn.Connect()
		if err != nil {
			log.Printf("cannot connect to %s: %s", hostname, err)
			continue
		}
		defer conn.Close()
		session := conn.Session()

		r := make(map[string]interface{})
		i := session.DB("config").C("databases").Find(bson.M{"partitioned": true}).Iter()
		for i.Next(r) {
			shardedCols[r["_id"].(string)] = true
		}

		dbs, err := conn.DatabaseNames()
		if err != nil {
			continue
		}
		for _, dbname := range dbs {
			_, ok := shardedCols[dbname]
			if !ok {
				unshardedCols[dbname] = true
			}
		}
	}
	fmt.Printf("Sharded cols: %d\n%+v\n", len(shardedCols), shardedCols)
	fmt.Printf("Unsharded cols: %d\n%+v\n", len(unshardedCols), unshardedCols)

}

func doSomething(session *mgo.Session) {

	var databases databases
	err := session.Run(bson.M{"listDatabases": 1}, &databases)
	if err != nil {
		log.Printf("error en dosomething %s\n", err)
		return
	}
	configDB := session.DB("config")

	var dbs []DB
	_ = configDB.C("databases").Find(nil).All(&dbs)
	var partitionedCount, notPartiionedCount int
	fmt.Printf("%+v\n", dbs)
	for _, db := range dbs {
		if !db.Partitioned {
			notPartiionedCount++
		}
		partitionedCount++
		var collections []shardStatusCollection
		err := configDB.C("collections").Find(bson.M{"_id": bson.RegEx{db.ID, ""}}).All(&collections)
		fmt.Printf("%v %+v\n", err, collections)
		chunksCol := configDB.C("chunks")
		for _, collection := range collections {
			if !collection.Dropped {
				var chunks []chunk
				err := chunksCol.Find(bson.M{"ns": collection.ID}).All(&chunks)
				if err != nil {
					continue
				}
				fmt.Println("----------------------------------------------------------------------------------------------------")
				for i, chunk := range chunks {
					fmt.Printf("%d: %+v\n\n", i, chunk)
				}
			}
		}
	}
	fmt.Printf("Partitioned: %d, not part: %d\n", partitionedCount, notPartiionedCount)
}

func getReplicasetMembers(hostnames []string) ([]proto.Members, error) {
	replicaMembers := []proto.Members{}

	for _, hostname := range hostnames {
		session, err := mgo.Dial(hostname)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get ReplicaSetStatus")
		}
		defer session.Close()

		rss := proto.ReplicaSetStatus{}
		err = session.Run(bson.M{"replSetGetStatus": 1}, &rss)
		if err != nil {
			continue // If a host is a mongos we cannot get info but is not a real error
		}
		for _, m := range rss.Members {
			m.Set = rss.Set
			replicaMembers = append(replicaMembers, m)
		}
	}

	return replicaMembers, nil
}

func getSecuritySettings(session *mgo.Session) (*security, error) {
	s := security{
		Auth: "disabled",
		SSL:  "disabled",
	}

	cmdOpts := proto.CommandLineOptions{}
	err := session.DB("admin").Run(bson.D{{"getCmdLineOpts", 1}, {"recordStats", 1}}, &cmdOpts)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get command line options")
	}

	if cmdOpts.Security.Authorization != "" || cmdOpts.Security.KeyFile != "" {
		s.Auth = "enabled"
	}
	if cmdOpts.Parsed.Net.SSL.Mode != "" && cmdOpts.Parsed.Net.SSL.Mode != "disabled" {
		s.SSL = cmdOpts.Parsed.Net.SSL.Mode
	}

	s.Users, err = session.DB("admin").C("system.users").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get users count")
	}

	s.Roles, err = session.DB("admin").C("system.roles").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get roles count")
	}

	return &s, nil
}

// TODO REMOVE. Used for debug.
func format(title string, templateData interface{}) string {
	txt, _ := json.MarshalIndent(templateData, "", "    ")
	return title + "\n" + string(txt)
}

func write(title string, templateData interface{}) {
	//d := os.Getenv("BASEDIR")
	//if d == "" {
	//	log.Printf("cannot get BASEDIR env var")
	//	return
	//}
	//txt, _ := json.MarshalIndent(templateData, "", "    ")
	//f, _ := os.Create("test/sample/" + title + ".json")
	//f.Write(txt)
	//f.Close()
}

func getNodeType(session *mgo.Session) (string, error) {
	md := proto.MasterDoc{}
	err := session.Run("isMaster", &md)
	if err != nil {
		return "", err
	}

	if md.SetName != nil || md.Hosts != nil {
		return "replset", nil
	} else if md.Msg == "isdbgrid" {
		// isdbgrid is always the msg value when calling isMaster on a mongos
		// see http://docs.mongodb.org/manual/core/sharded-cluster-query-router/
		return "mongos", nil
	}
	return "mongod", nil
}

func getOpCountersStats(conn db.MongoConnector, count int64, sleep time.Duration) chan opCounters {
	ch := make(chan opCounters)
	oc := opCounters{}
	go func() {
		ss, err := conn.ServerStatus()
		if err != nil {

			oc.Insert.Max = ss.Opcounters.Insert
			oc.Insert.Min = ss.Opcounters.Insert
			oc.Insert.Total = ss.Opcounters.Insert

			oc.Command.Max = ss.Opcounters.Command
			oc.Command.Min = ss.Opcounters.Command
			oc.Command.Total = ss.Opcounters.Command

			oc.Query.Max = ss.Opcounters.Query
			oc.Query.Min = ss.Opcounters.Query
			oc.Query.Total = ss.Opcounters.Query

			oc.Update.Max = ss.Opcounters.Update
			oc.Update.Min = ss.Opcounters.Update
			oc.Update.Total = ss.Opcounters.Update

			oc.Delete.Max = ss.Opcounters.Delete
			oc.Delete.Min = ss.Opcounters.Delete
			oc.Delete.Total = ss.Opcounters.Delete

			oc.GetMore.Max = ss.Opcounters.GetMore
			oc.GetMore.Min = ss.Opcounters.GetMore
			oc.GetMore.Total = ss.Opcounters.GetMore

		}

		ticker := time.NewTicker(sleep)
		for i := int64(0); i < count-1; i++ {
			<-ticker.C
			ss, err := conn.ServerStatus()
			if err != nil {
				continue
			}
			// Insert
			if ss.Opcounters.Insert > oc.Insert.Max {
				oc.Insert.Max = ss.Opcounters.Insert
			}
			if ss.Opcounters.Insert < oc.Insert.Min {
				oc.Insert.Min = ss.Opcounters.Insert
			}
			oc.Insert.Total += ss.Opcounters.Insert

			// Query ---------------------------------------
			if ss.Opcounters.Query > oc.Query.Max {
				oc.Query.Max = ss.Opcounters.Query
			}
			if ss.Opcounters.Query < oc.Query.Min {
				oc.Query.Min = ss.Opcounters.Query
			}
			oc.Query.Total += ss.Opcounters.Query

			// Command -------------------------------------
			if ss.Opcounters.Command > oc.Command.Max {
				oc.Command.Max = ss.Opcounters.Command
			}
			if ss.Opcounters.Command < oc.Command.Min {
				oc.Command.Min = ss.Opcounters.Command
			}
			oc.Command.Total += ss.Opcounters.Command

			// Update --------------------------------------
			if ss.Opcounters.Update > oc.Update.Max {
				oc.Update.Max = ss.Opcounters.Update
			}
			if ss.Opcounters.Update < oc.Update.Min {
				oc.Update.Min = ss.Opcounters.Update
			}
			oc.Update.Total += ss.Opcounters.Update

			// Delete --------------------------------------
			if ss.Opcounters.Delete > oc.Delete.Max {
				oc.Delete.Max = ss.Opcounters.Delete
			}
			if ss.Opcounters.Delete < oc.Delete.Min {
				oc.Delete.Min = ss.Opcounters.Delete
			}
			oc.Delete.Total += ss.Opcounters.Delete

			// GetMore -------------------------------------
			if ss.Opcounters.GetMore > oc.GetMore.Max {
				oc.GetMore.Max = ss.Opcounters.GetMore
			}
			if ss.Opcounters.GetMore < oc.GetMore.Min {
				oc.GetMore.Min = ss.Opcounters.GetMore
			}
			oc.GetMore.Total += ss.Opcounters.GetMore
		}
		ticker.Stop()

		oc.Insert.Avg = oc.Insert.Total / count
		oc.Query.Avg = oc.Query.Total / count
		oc.Update.Avg = oc.Update.Total / count
		oc.Delete.Avg = oc.Delete.Total / count
		oc.GetMore.Avg = oc.GetMore.Total / count
		oc.Command.Avg = oc.Command.Total / count
		ch <- oc

	}()
	return ch
}

func getProcInfo(pid int32, templateData *procInfo) error {
	//proc, err := process.NewProcess(templateData.ServerStatus.Pid)
	proc, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("cannot get process %d\n", pid)
	}
	ct, err := proc.CreateTime()
	if err != nil {
		return err
	}

	templateData.CreateTime = time.Unix(ct/1000, 0)
	templateData.Path, err = proc.Exe()
	if err != nil {
		return err
	}

	templateData.UserName, err = proc.Username()
	if err != nil {
		return err
	}
	return nil
}

func getDbsAndCollectionsCount(hostnames []string) (int, int, error) {
	dbnames := make(map[string]bool)
	colnames := make(map[string]bool)

	for _, hostname := range hostnames {
		session, err := mgo.Dial(hostname)
		if err != nil {
			continue
		}
		dbs, err := session.DatabaseNames()
		if err != nil {
			continue
		}

		for _, dbname := range dbs {
			dbnames[dbname] = true
			cols, err := session.DB(dbname).CollectionNames()
			if err != nil {
				continue
			}
			for _, colname := range cols {
				colnames[dbname+"."+colname] = true
			}
		}
	}

	return len(dbnames), len(colnames), nil
}
