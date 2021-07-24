module github.com/Luismorlan/newsmux

go 1.16

// This comment specify how do we do code gen when run go generate
// DO NOT REMOVE
//go:generate ./graphql_schema_gen.sh

require (
	github.com/graph-gophers/graphql-go v1.1.0
	github.com/stretchr/testify v1.7.0 // indirect
)
