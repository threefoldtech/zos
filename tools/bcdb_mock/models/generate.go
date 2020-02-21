package models

//go:generate rm -rf generated
//go:generate mkdir -p generated/directory
//go:generate mkdir -p generated/workloads
//go:generate schemac -pkg directory -dir generated/directory -in schema/directory
//go:generate schemac -pkg workloads -dir generated/workloads -in schema/workloads
