package proto

type Query struct {
	CurrentOp float64 `bson:"currentOp"`
}

type Progress struct {
	Done  float64 `bson:"done"`
	Total float64 `bson:"total"`
}

type AcquireCount struct {
	Rr float64 `bson:"r"`
	Ww float64 `bson:"w"`
	R  float64 `bson:"R"`
	W  float64 `bson:"W"`
}

type Global struct {
	DeadlockCount       AcquireCount `bson:"deadlockCount"`
	AcquireCount        AcquireCount `bson:"acquireCount"`
	AcquireWaitCount    AcquireCount `bson:"acquireWaitCount"`
	TimeAcquiringMicros AcquireCount `bson:"timeAcquiringMicros"`
}

type CurrentOpLockStats struct {
	Global        Global      `bson:"Global"`
	MMAPV1Journal interface{} `bson:"MMAPV1Journal"`
	Database      interface{} `bson:"Database"`
}

type Locks struct {
	Global        string `bson:"Global"`
	MMAPV1Journal string `bson:"MMAPV1Journal"`
	Database      string `bson:"Database"`
	Collection    string `bson:"Collection"`
	Metadata      string `bson:"Metadata"`
	Oplog         string `bson:"oplog"`
}

type Inprog struct {
	Desc             string             `bson:"desc"`
	ConnectionId     float64            `bson:"connectionId"`
	Opid             float64            `bson:"opid"`
	Msg              string             `bson:"msg"`
	NumYields        float64            `bson:"numYields"`
	Locks            Locks              `bson:"locks"`
	WaitingForLock   float64            `bson:"waitingForLock"`
	ThreadId         string             `bson:"threadId"`
	Active           float64            `bson:"active"`
	MicrosecsRunning float64            `bson:"microsecs_running"`
	SecsRunning      float64            `bson:"secs_running"`
	Op               string             `bson:"op"`
	Ns               string             `bson:"ns"`
	Insert           interface{}        `bson:"insert"`
	PlanSummary      string             `bson:"planSummary"`
	Client           string             `bson:"client"`
	Query            Query              `bson:"query"`
	Progress         Progress           `bson:"progress"`
	KillPending      float64            `bson:"killPending"`
	LockStats        CurrentOpLockStats `bson:"lockStats"`
}

type CurrentOp struct {
	Info      string   `bson:"info"`
	Inprog    []Inprog `bson:"inprog"`
	FsyncLock float64  `bson:"fsyncLock"`
}
