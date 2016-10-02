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
	Security           security
	RunningOps         opCounters
	SampleRate         time.Duration
	ReplicaMembers     []proto.Members
}

var Debug = false

func main() {
	var opts options
	flag.StringVar(&opts.Host, "hosts", "localhost", "List of host:port to connect to")
	flag.BoolVar(&opts.Debug, "debug", false, "debug mode")
	flag.Parse()

	conn := db.NewMongoConnector(opts.Host)
	err := conn.Connect()
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	shardsInfo, _ := conn.ListShards() // Error means we are not connected to a mongos

	// Host names for direct connection (not mongos)
	hostnames := []string{opts.Host}
	if shardsInfo != nil {
		hostnames = []string{}
		for _, shardInfo := range shardsInfo.Shards {
			m := strings.Split(shardInfo.Host, "/")
			h := strings.Split(m[1], ",")
			hostnames = append(hostnames, h[0])
		}
	}

	templateData := templateData{}
	templateData.BuildInfo, err = conn.BuildInfo()
	if err != nil {
		log.Fatalf("%s", err)
		return
	}

	md, err := conn.IsMaster()
	if err != nil {
		log.Print(err)
	}
	templateData.NodeType = GetNodeType(md)

	templateData.ReplicaMembers = getReplicasetMembers(hostnames, db.NewMongoConnector)

	templateData.ServerStatus, err = conn.ServerStatus()
	if err != nil {
		log.Print(err)
	}

	var sampleCount int64 = 5
	var sampleRate time.Duration = 1 // in seconds
	templateData.SampleRate = time.Duration(sampleCount) * time.Second * sampleRate
	osChan := getOpCountersStats(conn, sampleCount, sampleRate*time.Second)
	templateData.RunningOps = <-osChan

	templateData.HostInfo, err = conn.HostInfo()
	if err != nil {
		log.Print(err)
	}

	templateData.Security, err = getSecuritySettings(conn)

	fillMissingInfo(conn, &templateData)

	err = getProcInfo(int32(templateData.ServerStatus.Pid), &templateData.ProcInfo)
	if err != nil {
		log.Printf("cannot get proccess info: %s", err)
	}

	t := template.Must(template.New("replicas").Parse(templates.Replicas))
	t.Execute(os.Stdout, templateData)

	t = template.Must(template.New("hosttemplateData").Parse(templates.HostInfo))
	t.Execute(os.Stdout, templateData)

	t = template.Must(template.New("runningOps").Parse(templates.RunningOps))
	t.Execute(os.Stdout, templateData)

	t = template.Must(template.New("ssl").Parse(templates.Security))
	t.Execute(os.Stdout, templateData)

	oplogInfo, err := getOplogInfo(hostnames, db.NewMongoConnector)
	if oplogInfo != nil && len(oplogInfo) > 0 {
		t = template.Must(template.New("oplogInfo").Parse(templates.Oplog))
		t.Execute(os.Stdout, oplogInfo[0])
	}

}

func getReplicasetMembers(hostnames []string, newMongoConnector db.ConnectorFactory) []proto.Members {
	replicaMembers := []proto.Members{}

	for _, hostname := range hostnames {
		conn := newMongoConnector(hostname)
		err := conn.Connect()
		if err != nil {
			log.Printf("cannot connect to %s: %s", hostname, err)
			continue
		}
		defer conn.Close()

		replStatus, err := conn.ReplicaSetGetStatus()
		if err != nil {
			log.Printf("%v at fillReplicasetInfo, getReplicasetStatus", err)
			continue
		}
		for _, m := range replStatus.Members {
			m.Set = replStatus.Set
			replicaMembers = append(replicaMembers, m)
		}
	}

	return replicaMembers
}

func getSecuritySettings(conn db.MongoConnector) (security, error) {
	s := security{
		Auth: "disabled",
		SSL:  "disabled",
	}

	cmdOpts, err := conn.GetCmdLineOpts()
	if err != nil {
		return s, errors.Wrap(err, "cannot get security settings")
	}

	if cmdOpts.Security.Authorization != "" || cmdOpts.Security.KeyFile != "" {
		s.Auth = "enabled"
	}
	if cmdOpts.Parsed.Net.SSL.Mode != "" && cmdOpts.Parsed.Net.SSL.Mode != "disabled" {
		s.SSL = cmdOpts.Parsed.Net.SSL.Mode
	}

	s.Users, err = conn.UsersCount()
	if err != nil {
		return s, errors.Wrap(err, "cannot get users count")
	}

	s.Roles, err = conn.RolesCount()
	if err != nil {
		return s, errors.Wrap(err, "cannot get roles count")
	}

	return s, nil
}

// TODO REMOVE. Used for debug.
func format(title string, templateData interface{}) string {
	txt, _ := json.MarshalIndent(templateData, "", "    ")
	return title + "\n" + string(txt)
}

func GetNodeType(md proto.MasterDoc) string {

	if md.SetName != nil || md.Hosts != nil {
		return "replset"
	} else if md.Msg == "isdbgrid" {
		// isdbgrid is always the msg value when calling isMaster on a mongos
		// see http://docs.mongodb.org/manual/core/sharded-cluster-query-router/
		return "mongos"
	}
	return "mongod"
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

func fillMissingInfo(conn db.MongoConnector, templateData *templateData) error {

	dbNames, err := conn.DatabaseNames()
	if err != nil {
		return err
	}

	templateData.HostInfo.DatabasesCount = len(dbNames)
	for _, dbname := range dbNames {
		collectionNames, err := conn.CollectionNames(dbname)
		if err != nil {
			return err
		}
		templateData.HostInfo.CollectionsCount += len(collectionNames)
	}
	return nil
}
