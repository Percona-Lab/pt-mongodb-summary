package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"time"

	"labix.org/v2/mgo"

	"github.com/percona/pt-mongodb-summary/db"
	"github.com/percona/pt-mongodb-summary/proto"
	"github.com/percona/pt-mongodb-summary/templates"
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
}

var Debug = false

func main() {
	var opts options
	flag.StringVar(&opts.Host, "hosts", "localhost:17002", "List of host:port to connect to")
	flag.BoolVar(&opts.Debug, "debug", false, "debug mode")
	flag.Parse()

	conn := db.NewMongoConnector(opts.Host)
	err := conn.Connect()
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	templateData := templateData{}
	templateData.BuildInfo, err = conn.BuildInfo()
	if err != nil {
		log.Fatalf("%s", err)
		return
	}
	templateData.ProcessCount, err = countCurrentOps(conn)

	md, err := conn.IsMaster()
	if err != nil {
		log.Print(err)
	}
	templateData.NodeType = GetNodeType(md)

	templateData.ReplicaSetStatus, err = conn.ReplicaSetGetStatus()
	if err != nil {
		log.Print(err)
	}

	templateData.ServerStatus, err = conn.ServerStatus()
	if err != nil {
		log.Print(err)
	}

	templateData.HostInfo, err = conn.HostInfo()
	if err != nil {
		log.Print(err)
	}

	templateData.CommandLineOptions, err = conn.GetCmdLineOpts()
	if err != nil {
		log.Print(err)
	}

	fillMissingInfo(conn, &templateData)

	err = getProcInfo(templateData.ServerStatus.Pid, &templateData.ProcInfo)
	if err != nil {
		log.Printf("cannot get proccess info: %s", err)
	}

	t := template.Must(template.New("replicas").Parse(templates.Replicas))
	t.Execute(os.Stdout, templateData.ReplicaSetStatus)

	t = template.Must(template.New("hosttemplateData").Parse(templates.HostInfo))
	t.Execute(os.Stdout, templateData)

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
	for _, member := range templateData.ReplicaSetStatus.Members {
		if member.Name == templateData.ServerStatus.Repl.Me {
			templateData.ThisHostID = member.Id
		}
	}

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

func countCurrentOps(conn db.MongoConnector) (int64, error) {
	currentOp := proto.CurrentOp{}

	err := conn.Session().DB("admin").C("$cmd.sys.inprog").Find(nil).One(&currentOp)
	if err != nil {
		return 0, err
	}

	var i int64
	for _, inProg := range currentOp.Inprog {
		if inProg.Query.CurrentOp == 0 {
			i++
		}
	}
	return i, nil
}
