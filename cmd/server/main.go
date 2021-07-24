package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Luismorlan/newsmux/server"
	"github.com/Luismorlan/newsmux/server/graphql"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/graph-gophers/graphql-go/relay"
)

func main() {
	schemaString := graphql.GetGQLSchema()
	http.Handle("/graphql", &relay.Handler{
		Schema: utils.ParseGraphQLSchema(schemaString, &server.RootResolver{}),
	})

	fmt.Println("hello world from web backend, serving on 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
