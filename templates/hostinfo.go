package templates

const HostInfo = `# This host
# Mongo Executable #############################################################################
       Path to executable | {{.ProcInfo.Path}}
              Has symbols | No
# Report On {{.ThisHostID}} ########################################
                     User | {{.ProcInfo.UserName}}
                PID Owner | {{.ServerStatus.Process}}
                     Time | {{.ProcInfo.CreateTime}}
                 Hostname | {{.HostInfo.System.Hostname}}
                  Version | {{.ServerStatus.Version}}
                 Built On | {{.HostInfo.Os.Type}} {{.HostInfo.System.CpuArch}}
                  Started | {{.ProcInfo.CreateTime}}
                Databases | {{.HostInfo.DatabasesCount}}
              Collections | {{.HostInfo.CollectionsCount}}
                  Datadir | /data/db
                Processes | {{.ProcessCount}}
                  ReplSet | {{.ServerStatus.Repl.SetName}}
              Repl Status | {{.ReplicaSetStatus.MyState}}
             Process Type | {{.ServerStatus.Process}}
`
