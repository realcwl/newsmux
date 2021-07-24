package utils

import (
	graphql "github.com/graph-gophers/graphql-go"
)

// Reads and parses the schema
// Associates root resolver
func ParseGraphQLSchema(schemaString string, resolver interface{}) *graphql.Schema {
	var opts = []graphql.SchemaOpt{graphql.UseFieldResolvers()}

	parsedSchema, err := graphql.ParseSchema(
		schemaString,
		resolver,
		opts...,
	)
	if err != nil {
		panic(err)
	}

	return parsedSchema
}
