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
	//templateData.ProcessCount, err = countCurrentOps(conn)

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

	templateData.Security, err = getSecuritySettings(conn)

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
	currentOp, err := conn.GetCurrentOp()
	if err != nil {
		return 0, errors.Wrap(err, "cannot get current op")
	}

	var i int64
	for _, inProg := range currentOp.Inprog {
		if inProg.Query.CurrentOp == 0 {
			i++
		}
	}
	return i, nil
}
