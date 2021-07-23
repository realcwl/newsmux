package main

import (
	"fmt"
	"log"
	"net/http"

	api "github.com/Luismorlan/newsmux/server"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/graph-gophers/graphql-go/relay"
)

func main() {
	schemaString := `
		############################################
		# Data Types
		############################################
		type User {
		
		}
		
		type Post {
		
		}
		
		type Feed {
		
		}
		
		############################################
		# Queries
		# key should match resolver function name
		############################################
		type Query {
		# users: [User!]!
		# posts: [Post!]!
		# feeds: [Feed!]!
		}
		
		############################################
		# Mutations
		# key should match resolver function name
		############################################
		type Mutation {
		# addPost(title: String!): Post!
		# addFeed(title: String!): Feed!
		}
		
		############################################
		# Subscriptions
		# key should match resolver function name
		############################################
		type Subscription {
		}
		
		############################################
		# Top level schema, do not change
		############################################
		schema {
		query: Query
		mutation: Mutation
		subscription: Subscription
		}
	`

	http.Handle("/graphql", &relay.Handler{
		Schema: utils.ParseGraphQLSchema(schemaString, &api.RootResolver{}),
	})

	fmt.Println("hello world from web backend, serving on 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
