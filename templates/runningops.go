package templates

const RunningOps = `
# Running Ops ##################################################################################

Type         Min        Max        Avg
Insert    {{printf "% 8d" .RunningOps.Insert.Min}}   {{printf "% 8d" .RunningOps.Insert.Max}}   {{printf "% 8d" .RunningOps.Insert.Avg}}/{{.SampleRate}}
Query     {{printf "% 8d" .RunningOps.Query.Min}}   {{printf "% 8d" .RunningOps.Query.Max}}   {{printf "% 8d" .RunningOps.Query.Avg}}/{{.SampleRate}}
Update    {{printf "% 8d" .RunningOps.Update.Min}}   {{printf "% 8d" .RunningOps.Update.Max}}   {{printf "% 8d" .RunningOps.Update.Avg}}/{{.SampleRate}}
Delete    {{printf "% 8d" .RunningOps.Delete.Min}}   {{printf "% 8d" .RunningOps.Delete.Max}}   {{printf "% 8d" .RunningOps.Delete.Avg}}/{{.SampleRate}}
GetMore   {{printf "% 8d" .RunningOps.GetMore.Min}}   {{printf "% 8d" .RunningOps.GetMore.Max}}   {{printf "% 8d" .RunningOps.GetMore.Avg}}/{{.SampleRate}}
Command   {{printf "% 8d" .RunningOps.Command.Min}}   {{printf "% 8d" .RunningOps.Command.Max}}   {{printf "% 8d" .RunningOps.Command.Avg}}/{{.SampleRate}}
`
