package main

import (
	"time"

	"github.com/percona/pt-mongodb-summary/db"
	"github.com/pkg/errors"

	"labix.org/v2/mgo/bson"
)

type OplogInfo struct {
	Size          int64
	UsedMB        int64
	TimeDiff      int64
	TimeDiffHours int64
	TFirst        time.Time
	TLast         time.Time
	Now           time.Time
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

func getOplogInfo(conn db.MongoConnector) (*OplogInfo, error) {

	result := &OplogInfo{}

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

	/*
		from MongoDB's DB.tsToSeconds:
			function (x){
			    if ( x.t && x.i )
			        return x.t;
			    return x / 4294967296; // low 32 bits are ordinal #s within a second
			}
	*/

	tfirst := firstRow.Ts / 4294967296
	tlast := lastRow.Ts / 4294967296
	result.TimeDiff = tlast - tfirst
	result.TimeDiffHours = result.TimeDiff / 3600

	result.TFirst = time.Unix(tfirst, 0)
	result.TLast = time.Unix(tlast, 0)
	result.Now = time.Now().UTC()

	return result, nil

}
