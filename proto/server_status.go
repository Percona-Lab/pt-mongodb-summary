package proto

type ServerStatus struct {
	UptimeEstimate     float64            `bson:"uptimeEstimate"`
	BackgroundFlushing BackgroundFlushing `bson:"backgroundFlushing"`
	Connections        Connections        `bson:"connections"`
	GlobalLock         GlobalLock         `bson:"globalLock"`
	Host               string             `bson:"host"`
	Mem                Mem                `bson:"mem"`
	OpcountersRepl     Opcounters         `bson:"opcountersRepl"`
	Uptime             float64            `bson:"uptime"`
	UptimeMillis       float64            `bson:"uptimeMillis"`
	Cursors            Cursors            `bson:"cursors"`
	Dur                Dur                `bson:"dur"`
	LocalTime          string             `bson:"localTime"`
	// Metrics            *Metrics            `bson:"metrics"`
	Network           Network    `bson:"network"`
	Opcounters        Opcounters `bson:"opcounters"`
	Pid               int32      `bson:"pid"`
	Asserts           Asserts    `bson:"asserts"`
	Ok                float64    `bson:"ok"`
	Process           string     `bson:"process"`
	Repl              Repl       `bson:"repl"`
	StorageEngineName string     `bson:"storageEngine.name"`
	ExtraInfo         ExtraInfo  `bson:"extra_info"`
	Locks             Locks      `bson:"locks"`
	Version           string     `bson:"version"`
	WriteBacksQueued  bool       `bson:"writeBacksQueued"`
}
