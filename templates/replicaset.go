package templates

const Replicas = `
# Instances ####################################################################################
ID    Host                         Type                                 ReplSet  Engine Status
{{- if .ReplicaMembers -}}
{{- range .ReplicaMembers }} 
{{printf "% 3d" .Id}} {{printf "%-30s" .Name}} {{printf "%-30s" .StateStr}} {{printf "%10s" .Set -}}
{{end}}
{{else}}																		  
                                          No replica sets found
{{end}}

`
