gitlab.com/gopherburrow/mux
===

[![GoDoc](https://godoc.org/gitlab.com/gopherburrow/mux?status.svg)](https://godoc.org/gitlab.com/gopherburrow/mux)
[![pipeline status](https://gitlab.com/gopherburrow/mux/badges/master/pipeline.svg)](https://gitlab.com/gopherburrow/mux/commits/master)
[![coverage report](https://gitlab.com/gopherburrow/mux/badges/master/coverage.svg)](https://gitlab.com/gopherburrow/mux/commits/master)

![Gopher Burrow Logo](https://gitlab.com/gopherburrow/art/raw/master/gopherburrow.png)

`gitlab.com/gopherburrow/mux` Implements an URL mutiplexing matcher and dispatcher. It receives an HTTP request then it compares it with a pre-configured table and dispatch it to an `http.Handler`. 

It meant to be simple and small and is made to fit perfectly in a multitenancy web project, where only an HTTP method and an absolute, complete and canonical URL is needed to match a request and dispatch it to a Handler. 

For a general use mux, I recommend the amazing Gorilla Webkit Mux (https://github.com/gorilla/mux) that has much more advanced mutiplexing/routing/dispatch capabilities. Specifically, I do not use it in Riot Emergence because it is a bit overkill for my project needs.

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
go get -u gitlab.com/gopherburrow/mux
```

# Usage

Import:
```go
import (
	"gitlab.com/gopherburrow/mux"
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

	"gitlab.com/gopherburrow/mux"
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

