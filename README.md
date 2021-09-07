# Newsmux

Newsmux is the API server for project Newerfeed. It serves all Newerfeed's
frontend request with RESTful API. This API server is written in Golang.

# Design

The complete design can be seen in the following notion page:
[Notion Link](https://www.notion.so/Backend-296895971b4348aab7e3909063dfc4d2)

# Project layout

```
.
├── README.md
├── Makefile
├── cmd
│   ├── publisher (entry for news feed publisher)
│   │   └── main.go
│   └── server (entry for web server)
│       └── main.go
├── config (config file)
│   └── config.go
├── go.mod
├── go.sum
├── models (data structures and ORM related files)
│   ├── feed.go
│   ├── post.go
│   └── user.go
├── publisher (news feed publisher logic code)
│   └── README.md
├── server (web server logic code)
│   ├── app.go (web server logic code)
│   ├── mutations (graphql mutation schema/resolver code)
│   │   └── README.md
│   ├── queries (graphql query schema/resolver code)
│   │   └── README.md
│   └── schema.graphql (graphql central schema template)
└── utils (shared util functions)
    ├── log.go
    ├── utils.go
    └── utils_test.go

```

# Commands

`make server_dev` to run dev server
