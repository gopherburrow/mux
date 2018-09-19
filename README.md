github.com/riotmergence/mux
===

[![GoDoc](https://godoc.org/github.com/riotemergence/mux?status.svg)](https://godoc.org/github.com/riotemergence/mux)
[![Build Status](https://travis-ci.org/riotemergence/mux.svg?branch=master)](https://travis-ci.org/riotemergence/mux)
[![Sourcegraph](https://sourcegraph.com/github.com/riotemergence/mux/-/badge.svg)](https://sourcegraph.com/github.com/riotemergence/mux?badge)

![Riot Emergence Logo](https://raw.githubusercontent.com/riotemergence/devguidelines/master/riotemergence-256px.png)

`github.com/riotmergence/mux` Implements an URL mutiplexing matcher and dispatcher. It receive a HTTP request, matches it against a pre-configured table and dispatch it to an `http.Handler`. 

It meant to be simple and small. And is made to fit perfectly in riotemergence web project (https://github.com/riotemergence/web) where a HTTP method and an absolute, complete and canonical URL is always requested and is enough to match a request and dispatch it to a Handler. 

For a general use mux, I recommend the amazing Gorilla Webkit Mux (https://github.com/gorilla/mux) that have much more advanced mutiplexing/routing/dispatch capabilities. Specifically, I do not use it in Riot Emergence because it is a bit overkill for my project needs.

Features:
* Complete URL matching using only a HTTP Method (`GET`, `POST`...) and a simple URL pattern;
* Capable of matching based on the protocol scheme (`http`, `https`...), host, port, path, query parameters names and values;
* Capable of extracting path parameters values.

---
Table of Contents
===
- [Install](#install)
- [Usage](#usage)
---
# Install

Within a [Go workspace](https://golang.org/doc/code.html#Workspaces):

```sh
go get -u github.com/riotemergence/mux
```

# Usage

Import:
```go
import (
	"github.com/riotemergence/mux"
)
```
Create a Mux:
```go
m = &mux.Mux{}
```
Register some handlers:
```go
m.Handle(http.MethodGet, "http://localhost:8080/{path}", YourHandler)
//or
m.HandleFunc(http.MethodGet, "http://localhost:8080/{path}", YourHandlerFunc)
```
Start a server passing a Mux as a handler:
```go
log.Fatal(http.ListenAndServe(":8080", m))
```
Extract some path variables:
```go
v := m.PathVars(r)
path := v["path"]
```
# Example

Write and build the folowing lines:

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/riotemergence/mux"
)

var m *mux.Mux

func main() {
	m = &mux.Mux{}
	// Mux routes consist of a HTTP method, an URL and a handler function.
	m.HandleFunc(http.MethodGet, "http://localhost:8080/fixedpath/{dynamicPath}?has-parameter&fixed-parameter=fixed-value", YourHandler)

	// Bind to a port and pass to mux
	log.Fatal(http.ListenAndServe(":8080", m))
}

func YourHandler(w http.ResponseWriter, r *http.Request) {
	v := m.PathVars(r) // ... map[string]string returned
	fmt.Fprint(w, "Riot Emergence Mux Hello World "+v["dynamicPath"])
}
```
And load on the browser:

[http://localhost:8080/fixedpath/Hello%20World?has-parameter&fixed-parameter=fixed-value](http://localhost:8080/fixedpath/Hello%20World?has-parameter&fixed-parameter=fixed-value)

