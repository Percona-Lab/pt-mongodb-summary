package templates

const Replicas = `
# Instances ####################################################################################
ID    Host                         Type                                 ReplSet  Engine Status
{{$set:= .Set -}}
{{if .Members}}
{{range .Members }} 
{{printf "% 3d" .Id}} {{printf "%-30s" .Name}} {{printf "%-30s" .StateStr}} {{printf "%10s" $set -}}
{{end}}
{{else}}																		  
                                          No replica sets found
{{end}}

`
