package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/percona/pt-mongodb-summary/db"
	"github.com/pkg/errors"

	"labix.org/v2/mgo/bson"
)

type OplogInfo struct {
	Hostname      string
	Size          int64
	UsedMB        int64
	TimeDiff      int64
	TimeDiffHours float64
	Running       string // TimeDiffHours in human readable format
	TFirst        time.Time
	TLast         time.Time
	Now           time.Time
	ElectionTime  time.Time
}

type OpLogs []OplogInfo

func (s OpLogs) Len() int {
	return len(s)
}
func (s OpLogs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s OpLogs) Less(i, j int) bool {
	return s[i].TimeDiffHours < s[j].TimeDiffHours
}

type OplogRow struct {
	H  int64  `bson:"h"`
	V  int64  `bson:"v"`
	Op string `bson:"op"`
	O  bson.M `bson:"o"`
	Ts int64  `bson:"ts"`
}

type ColStats struct {
	NumExtents        int
	IndexDetails      bson.M
	Nindexes          int
	TotalIndexSize    int64
	Size              int64
	PaddingFactorNote string
	Capped            bool
	MaxSize           int64
	IndexSizes        bson.M
	GleStats          struct {
		LastOpTime int64
		ElectionId string
	} `bson:"$gleStats"`
	StorageSize    int64
	PaddingFactor  int64
	AvgObjSize     int64
	LastExtentSize int64
	UserFlags      int64
	Max            int64
	Ok             int
	Ns             string
	Count          int64
}

func getOplogInfo(hostnames []string, newMongoConnector db.ConnectorFactory) ([]OplogInfo, error) {

	results := OpLogs{}

	for _, hostname := range hostnames {
		result := OplogInfo{
			Hostname: hostname,
		}
		conn := newMongoConnector(hostname)
		err := conn.Connect()
		if err != nil {
			return nil, errors.Wrapf(err, "cannot connect to %s", hostname)
		}
		defer conn.Close()

		oplogCol, err := conn.GetOplogCollection()
		if err != nil {
			return nil, err
		}

		olEntry, err := conn.GetOplogEntry(oplogCol)
		if err != nil {
			return nil, errors.Wrap(err, "getOplogInfo -> GetOplogEntry")
		}
		result.Size = olEntry.Options.Size / (1024 * 1024)

		var colStats ColStats
		err = conn.DbRun("local", bson.M{"collStats": oplogCol}, &colStats)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get collStats for collection %s", oplogCol)
		}

		result.UsedMB = colStats.Size / (1024 * 1024)

		var firstRow, lastRow OplogRow
		err = conn.FindOne("local", oplogCol, nil, []string{"$natural"}, &firstRow)
		if err != nil {
			return nil, errors.Wrap(err, "cannot read first oplog row")
		}

		err = conn.FindOne("local", oplogCol, nil, []string{"-$natural"}, &lastRow)
		if err != nil {
			return nil, errors.Wrap(err, "cannot read last oplog row")
		}

		// https://docs.mongodb.com/manual/reference/bson-types/#timestamps
		tfirst := firstRow.Ts >> 32
		tlast := lastRow.Ts >> 32
		result.TimeDiff = tlast - tfirst
		result.TimeDiffHours = float64(result.TimeDiff) / 3600

		result.TFirst = time.Unix(tfirst, 0)
		result.TLast = time.Unix(tlast, 0)
		result.Now = time.Now().UTC()
		if result.TimeDiffHours > 24 {
			result.Running = fmt.Sprintf("%d days", result.TimeDiffHours/24)
		} else {
			result.Running = fmt.Sprintf("%0.2f hours", result.TimeDiffHours)
		}

		replSetStatus, err := conn.ReplicaSetGetStatus()
		for _, member := range replSetStatus.Members {
			if member.State == 1 {
				result.ElectionTime = time.Unix(member.ElectionTime>>32, 0)
				break
			}
		}

		results = append(results, result)
	}

	sort.Sort(results)
	return results, nil

}
