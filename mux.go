// This file is part of Riot Emergence Mux.
//
// Riot Emergence Mux is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Riot Emergence Mux is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Riot Emergence Mux.  If not, see <http://www.gnu.org/licenses/>.

//Package mux contains an URL mutiplexing matcher and dispatcher.
//
//It stores a routing table comprised of URL Patterns on one side and a set of `http.Handler` on the other.
//
//And when it receive a HTTP request, matches it against the pre-configured routing table
//and dispatch it to the apropriate `http.Handler`.
package mux

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
)

const (
	//Used in request contexts.
	ctxGetValue = "github.com/riotemergence/mux Get"
)

//Allowed values for Schemes and HTTP Methods used in validations.
var (
	defaultAllowedSchemes     = []string{"http", "https"}
	defaultAllowedHTTPMethods = []string{http.MethodPut, http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodTrace}
)

//Errors returned by methods.
var (
	//ErrHandlerMustBeNotNil is returned by Handle method when the handler parameter is nil.
	ErrHandlerMustBeNotNil = errors.New("mux: Handler must be not nil")
	//ErrMethodMustBeValid is returned by Handle and RemoveHandler methods when the httpMethod parameter is invalid.
	//Valid values are: http.MethodPut, http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodConnect and http.MethodTrace.
	ErrMethodMustBeValid = errors.New("mux: Invalid HTTP method")
	//ErrRequestMustHaveContext is returned by Get method when an context is not found.
	ErrRequestMustHaveContext = errors.New("mux: context not found (request must came from a mux Handler)")
	//ErrRouteMustExist is returned by RemoveHandler method when the route is not found.
	ErrRouteMustExist = errors.New("mux: route not found")
	//ErrRouteMustNotConflict is returned by Handle method when a conflicting route is found.
	ErrRouteMustNotConflict = errors.New("mux: route conflicting with a pre existing route")
	//ErrURLPatternInvalidQueryRoute is returned by Handle and RemoveHandler methods when an invalid query routing is found in urlPattern parameter.
	ErrURLPatternInvalidQueryRoute = errors.New("mux: invalid URL pattern query routing (query parameter presence tests or value tests are mutually exclusive)")
	//ErrURLPatternInvalidPathVar is returned by Handle and RemoveHandler methods when an invalid path variable is found in urlPattern parameter.
	ErrURLPatternInvalidPathVar = errors.New("mux: invalid URL pattern path variables")
	//ErrURLPatternMustBeValid is returned by Handle and RemoveHandler methods when the urlPattern parameter is invalid.
	//Valid schemes are https and http.
	ErrURLPatternMustBeValid = errors.New("mux: invalid URL pattern")
)

//Used in request contexts. Go suggests using a specific type different from string for context keys.
type ctxType string

//The key used to store the mux used in route dispatching. So it is possible to retrieve it inside a `http.Handler` to extract path vars for example.
var ctxGet = ctxType(ctxGetValue)

//queryEntry represents a single query parameter with or without value. Eg: name=value or name-without-value .
type queryEntry struct {
	Name  string
	Value string
}

//queryRoute represents a structured (simply sorted) set of query entries able to be used in request routing.
//
//There are two types oy query parameters routing:
//Using presence tests: The value of parameter is not used;
//Using value tests: Using both name and value to trigger a routing.
//They are exclusive. Only one type of test can be used per parameter name.
//Using value tests can use the same parameter name and values many times over.
type queryRoute []queryEntry

//newQueryRoute creates a valid `queryEntries`.
func newQueryRoute(urlQueryParamsAndValues url.Values) (queryRoute, error) {
	//Iterate over each query parameter...
	entries := make(queryRoute, 0)
	for paramName, paramValues := range urlQueryParamsAndValues {

		//...to validate if only presence tests or value tests are made exclusively on each parameter name.
		alreadyHavePresenceTest := false
		alreadyHaveValueTest := false
		for _, paramValue := range paramValues {
			if alreadyHavePresenceTest {
				return nil, ErrURLPatternInvalidQueryRoute
			}
			if paramValue == "" {
				if alreadyHaveValueTest {
					return nil, ErrURLPatternInvalidQueryRoute
				}
				alreadyHavePresenceTest = true
			} else {
				alreadyHaveValueTest = true
			}

			//If everything is OK, use each query parameter test...
			entries = append(entries, queryEntry{
				Name:  paramName,
				Value: paramValue,
			})
		}
	}
	//...And sort the entire set for sake of performance on Acceptable tests.
	sort.Sort(entries)
	return entries, nil
}

//Acceptable test if a URL query string is eligible to be routed.
func (route queryRoute) Acceptable(requestQueryValues url.Values) bool {
	//Create a sorted set of query entries from request, so it can be compared vs queryRoute with some performance.
	reqSortedQueryEntries := make(queryRoute, 0)
	for paramName, paramValues := range requestQueryValues {
		for _, paramValue := range paramValues {
			reqSortedQueryEntries = append(reqSortedQueryEntries, queryEntry{
				Name:  paramName,
				Value: paramValue,
			})
		}
	}
	sort.Sort(reqSortedQueryEntries)

	//Iterate synchronously over both sorted collections (route and query) searching for matches in each query parameter name.
	routeParamIndex := 0
	queryParamIndex := 0
	routeParamAlreadyUsed := false
	for routeParamIndex < len(route) && queryParamIndex < len(reqSortedQueryEntries) {
		routeParam := route[routeParamIndex]
		queryParam := reqSortedQueryEntries[queryParamIndex]

		//Sometimes the query  parameter name is not used by route...
		if routeParam.Name > queryParam.Name || (routeParam.Name == queryParam.Name && routeParam.Value != "" && routeParam.Value > queryParam.Value) {
			//...So lets skip it.
			queryParamIndex++
			continue
		}

		//And sometimes we find a match like now...
		if routeParam.Name == queryParam.Name && (routeParam.Value == "" || routeParam.Value == queryParam.Value) {
			//...So the query parameter will be skiiped and ...
			queryParamIndex++
			//...the matched could be skipped route as well (routeParamIndex++), but because sometimes a route can match more than one query parameter value....
			//...it will be flagged as used, and it can be discarded if in the next iteration we dont find a compatible parameter.
			routeParamAlreadyUsed = true
			continue
		}

		//Now a match not happened. we could return false now...

		//...But if the route parameter was flagged 'used' then everything is fine...
		if routeParamAlreadyUsed {
			//..The already used route parameter will be skipped and the mark reseted.
			routeParamAlreadyUsed = false
			routeParamIndex++
			continue
		}

		//Yeah the unused route parameter is not mached. Nothing left to do.
		return false
	}

	//Adjust the route parameter counter in the case that the last route parameter was used and not skipped.
	if routeParamAlreadyUsed {
		routeParamIndex++
	}

	//If ALL the route parameters were used then the route matches.
	return routeParamIndex == len(route)
}

//queryRouting Sort Interface. Sorted by parameter name and then parameter value.
func (route queryRoute) Len() int {
	return len(route)
}
func (route queryRoute) Less(i, j int) bool {
	if route[i].Name < route[j].Name {
		return true
	}
	if route[i].Name > route[j].Name {
		return false
	}
	return route[i].Value < route[j].Value
}
func (route queryRoute) Swap(i, j int) {
	route[i], route[j] = route[j], route[i]
}

//queryRouting Stringer Interface. Shows a query string in the format ?query1=value1&query2=value2&...
func (route queryRoute) String() string {
	if route.Len() == 0 {
		return ""
	}
	b := bytes.Buffer{}
	for i, queryEntry := range route {
		if i == 0 {
			b.WriteString("?")
		} else {
			b.WriteString("&")
		}
		b.WriteString(queryEntry.Name)
		if queryEntry.Value == "" {
			continue
		}
		b.WriteString("=")
		b.WriteString(queryEntry.Value)
	}
	return b.String()
}

//muxRoute represents a route in a mux entry.
type muxRoute struct {
	method string
	scheme string
	host   string
	path   []string
	vars   map[string]int
	query  queryRoute
}

//newMuxRoute ia a constructor for muxRoute.
func newMuxRoute(httpMethod string, urlPattern string) (*muxRoute, error) {
	//Validates all the aspects from inputs. Probably needs more validations.
	if !containsString(defaultAllowedHTTPMethods, httpMethod) {
		return nil, ErrMethodMustBeValid
	}
	if urlPattern == "" {
		return nil, ErrURLPatternMustBeValid
	}
	url, err := url.Parse(urlPattern)
	if err != nil {
		return nil, ErrURLPatternMustBeValid
	}
	if !url.IsAbs() {
		return nil, ErrURLPatternMustBeValid
	}
	if !containsString(defaultAllowedSchemes, url.Scheme) {
		return nil, ErrURLPatternMustBeValid
	}
	if url.Host == "" {
		return nil, ErrURLPatternMustBeValid
	}
	if strings.HasPrefix(url.Host, ":") {
		return nil, ErrURLPatternMustBeValid
	}

	//Extract the path in segments.
	pathSegments := splitPathSegs(url.Path)

	//And then extract dynamic vars from path segments, creating a map from names to path segment indexes.
	lastSeg := len(pathSegments) - 1
	vars := map[string]int{}
	for i, v := range pathSegments {
		if !strings.HasPrefix(v, "{") || !strings.HasSuffix(v, "}") {
			continue
		}
		k := strings.TrimSpace(strings.Trim(v, "{}"))
		if k == "" {
			return nil, ErrURLPatternInvalidPathVar
		}
		if k == "*" && i != lastSeg {
			return nil, ErrURLPatternInvalidPathVar
		}
		if _, r := vars[k]; r {
			return nil, ErrURLPatternInvalidPathVar
		}
		vars[k] = i
	}

	//Create a structured query routing.
	queryRoute, err := newQueryRoute(url.Query())
	if err != nil {
		return nil, err
	}

	//And finally created.
	return &muxRoute{
		method: httpMethod,
		scheme: url.Scheme,
		host:   url.Host,
		path:   pathSegments,
		vars:   vars,
		query:  queryRoute,
	}, nil
}

//String is Stringer Interface for muxRoute.
//Format: method+scheme://host:port/path/...?query1=value&... Eg: GET+http://localhost:8080/examplepath/examplesubpath?exampleparam1=value1&exampleparam2=value2
func (r *muxRoute) String() string {
	b := bytes.Buffer{}
	b.WriteString(r.method)
	b.WriteString("+")
	b.WriteString(r.scheme)
	b.WriteString("://")
	b.WriteString(r.host)
	for _, p := range r.path {
		b.WriteString("/")
		b.WriteString(p)
	}
	b.WriteString(r.query.String())
	return b.String()
}

//muxEntry Binds together a route and a Handler.
type muxEntry struct {
	route   *muxRoute
	handler http.Handler
}

//muxEntries Collection
type muxEntries []muxEntry

//Mux implements an URL mutiplexing matcher and dispatcher.
type Mux struct {
	//NotFoundHandler specifies an optional `http.Handler` when a route match from request is not found.
	//If nil, the Mux will use the default http.NotFound handler.
	NotFoundHandler http.Handler
	entriesLock     sync.RWMutex
	entries         muxEntries
}

//Get retrieves the mux used in dispatch, So it can be used to extract path variables throught PathVars method.
//
//Possible error returns:
//
//• mux.ErrRequestMustHaveContext
func Get(r *http.Request) (*Mux, error) {
	m, ok := r.Context().Value(ctxGet).(*Mux)
	if !ok {
		return nil, ErrRequestMustHaveContext
	}
	return m, nil
}

//Handle creates a routing entry in routing table and assigns a `http.Handler` to be dispatched when ServeHTTP receives a request that matches the route.
//
//This method does not allow routes patterns (method+url) conflicts and will return an error.
//
//Parameters
//
//• httpMethod: a string containing a HTTP method (Eg: GET)
//
//• urlPattern: an URL containing necessarily a scheme and host, and optionally a port, a path segments (static or dynamic) and query strings.
//
//Check the `mux.Mux.ServeHTTP` method to check how this urlPattern is compared with the actual `http.Request` being served.
//
//• handler: a `http.Handler` that will be called when the request matches the route.
//
//Variable Paths
//
//Dynamic paths and path variables can be defined using a name inside a pair of open and closed braces on a path segment.
//Eg: The GET http://localhost/{path-var} route can be matched on a request GET http://localhost/hello-world and the `path-var` variable can be extracted as the value "hello-world" using PathVars method.
//
//Path variables can be also used to match a complete sub path using the keyword {*}. Eg: The GET http://localhost/some-path/{*} will match a request like GET http://localhost/some-path/sub1/sub2 . The PathVar "*" will be valued "sub1/sub2".
//
//Query Strings Routing
//
//Query routing rules uses two types of testing: Presence test (Eg: http://localhost/path?param) and Value test (Eg: http://localhost/path?param=value). Only one type of testing per parameter name is allowed. The tests follow an alfabetic order,
//
//Errors
//
//• mux.ErrHandlerMustBeNotNil
//
//• mux.ErrMethodMustBeValid
//
//• mux.ErrRouteMustNotConflict
//
//• mux.ErrURLPatternInvalidQueryRoute
//
//• mux.ErrURLPatternInvalidPathVar
//
//• mux.ErrURLPatternMustBeValid
func (m *Mux) Handle(httpMethod string, urlPattern string, handler http.Handler) error {
	//Validate method inputs and convert to usable route.
	route, err := newMuxRoute(httpMethod, urlPattern)
	if err != nil {
		return err
	}
	if handler == nil {
		return ErrHandlerMustBeNotNil
	}

	//Validate route conflicts and find a place to put the new route entry.
	m.entriesLock.RLock()
	eLen := len(m.entries)
	i, _, found := searchRange(
		eLen, func(i int) int {
			return compareDynamicRoutes(route, m.entries[i].route)
		})

	//If a conflict is found return an error.
	if found {
		m.entriesLock.RUnlock()
		return ErrRouteMustNotConflict
	}
	m.entriesLock.RUnlock()

	//Put the new entry in place and return successfully.
	m.entriesLock.Lock()
	m.entries = append(m.entries, muxEntry{})
	copy(m.entries[i+1:], m.entries[i:])
	m.entries[i] = muxEntry{
		route: route, handler: handler,
	}
	m.entriesLock.Unlock()
	return nil
}

//RemoveHandler removes a handler from an existing route.
//
//Errors
//
//• mux.ErrMethodMustBeValid
//
//• mux.ErrRouteMustExist
//
//• mux.ErrURLPatternInvalidQueryRoute
//
//• mux.ErrURLPatternInvalidPathVar
//
//• mux.ErrURLPatternMustBeValid
func (m *Mux) RemoveHandler(httpMethod, urlPattern string) error {
	//Validate method inputs and convert to usable route.
	route, err := newMuxRoute(httpMethod, urlPattern)
	if err != nil {
		return err
	}

	//Find a route match and its index on entries.
	m.entriesLock.RLock()
	i, _, found := searchRange(
		len(m.entries),
		func(i int) int {
			return compareStaticRoutes(route, m.entries[i].route)
		})
	m.entriesLock.RUnlock()

	//But if it not exists return an error.
	if !found {
		return ErrRouteMustExist
	}

	//Remove the route entry and return successfully.
	m.entriesLock.Lock()
	m.entries = m.entries[:i+copy(m.entries[i:], m.entries[i+1:])]
	m.entriesLock.Unlock()
	return nil
}

//ServeHTTP dispatches requests according to routing rules to pre configured `http.Handler` added by `Handle` or `HandleFunc` methods.
//
//This methods implies that the requests are being served directly, not behind a reverse proxy. The values are extracted from *http.Request in the following way:
//
//• The scheme value (http or https) is based in the `*http.Request.TLS` field. If it is not `nil` the "https" value will be used, otherwise "http" will be used.
//
//• The host is extracted from `*http.Request.Host`.
//
//If the requests are being served behind a reverse proxy, adjust the values before handler is called. This is achieved normally by creating a intermediate delegating http.Handler that translate the requests.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//Try to find the route match using, method, scheme, host, port and path. Query strings will be tested ahead.
	m.entriesLock.RLock()
	lo, hi, found := searchRange(
		len(m.entries), func(i int) int {
			return compareRequestRoute(r, m.entries[i].route)
		})

	//If a match is not found, call NotFoundHandler.
	if !found {
		m.entriesLock.RUnlock()
		m.notFound(w, r)
		return
	}

	//Test query strings for a match.
	i := lo
	for ; i < hi && !m.entries[i].route.query.Acceptable(r.URL.Query()); i++ {
	}

	//And, again, If a match is not found, call NotFoundHandler.
	if i == hi {
		m.entriesLock.RUnlock()
		m.notFound(w, r)
		return
	}
	m.entriesLock.RUnlock()

	//But if it is found, call the assigned Handler passing the mux in Context.
	m.entries[i].handler.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxGet, m)))
}

//notFound calls a handler when a route match is not found in ServeHTTP method. And if it is not set call the default http.NotFound handler.
func (m *Mux) notFound(w http.ResponseWriter, r *http.Request) {
	if m.NotFoundHandler == nil {
		http.NotFound(w, r)
		return
	}
	m.NotFoundHandler.ServeHTTP(w, r)
	return
}

//PathVars extract all the variable path segments values as a map from a request that was handled by a Mux.
//
//It returns a map with all variables found using the name during the Handle(...) call.
//
//Only path segments can be extracted using PathVars. There is no scheme, host, port or query values extraction mechanisms in Mux, they can be extracted throught the usual methods in the http.Request parameter.
func (m *Mux) PathVars(r *http.Request) map[string]string {
	//Find the used route.
	vars := map[string]string{}
	m.entriesLock.RLock()
	eLen := len(m.entries)
	i, _, found := searchRange(
		eLen, func(i int) int {
			return compareRequestRoute(r, m.entries[i].route)
		})

	//If not found the route match. Return the empty map.
	if !found {
		m.entriesLock.RUnlock()
		return vars
	}

	//When the route is found return  each path segment value based on the previously processed and stored index...
	entry := m.entries[i]
	m.entriesLock.RUnlock()
	pathSegs := splitPathSegs(r.URL.Path)
	for k, v := range entry.route.vars {
		//...for sub paths join all sub segments values.
		if k == "*" {
			vars[k] = strings.Join(pathSegs[v:], "/")
			continue
		}
		vars[k] = pathSegs[v]
	}
	return vars
}

//String shows a sorted list of registered routes.
func (m *Mux) String() string {
	b := bytes.Buffer{}
	for _, e := range m.entries {
		b.WriteString(e.route.String())
		b.WriteString("\n")
	}
	return b.String()
}

//compareDynamicRoutes compares two routes at insertion on routing table. It is used to guarantee that entries do not conflict with each other.
//It differs from a simple static comparation because it verifies some dynamic path segments and query parameters vs static ones.
func compareDynamicRoutes(r1, r2 *muxRoute) int {
	//Compare the common static part.
	if r := compareMethodSchemeHost(r1.method, r2.method, r1.scheme, r2.scheme, r1.host, r2.host); r != 0 {
		return r
	}

	//Compare the url path...
	rp1Len, rp2Len := len(r1.path), len(r2.path)
	for i := 0; i < rp1Len && i < rp2Len; i++ {
		//...checking if a sub-path is used, so any comparation at this path segment matches...
		if (i == (rp1Len-1) && r1.path[i] == "{*}") || (i == (rp2Len-1) && r2.path[i] == "{*}") {
			return 0
		}
		//...a variable segment tested against a static segment matches too...
		seg1, seg2 := r1.path[i], r2.path[i]
		varSeg1, varSeg2 := strings.HasPrefix(seg1, "{") && strings.HasSuffix(seg1, "}"), strings.HasPrefix(seg2, "{") && strings.HasSuffix(seg2, "}")
		if varSeg1 != varSeg2 {
			return 0
		}
		//...but two variable segments must test subsequent segments...
		if varSeg1 {
			continue
		}
		//...and two static segments are compared using their values...
		if r := strings.Compare(seg1, seg2); r != 0 {
			return r
		}
	}
	//...if everything matches until now, compare path sizes.
	if r := rp1Len - rp2Len; r != 0 {
		return r
	}

	//Compare query strings...
	//...Sorting larger quantities to smaller...
	if r := len(r2.query) - len(r1.query); r != 0 {
		return r
	}
	for i := 0; i < len(r1.query); i++ {
		//... And each query parameter name alphabetically...
		if r := strings.Compare(r1.query[i].Name, r2.query[i].Name); r != 0 {
			return r
		}
		//...If the names are equal, a test of presence against values always match...
		if r1.query[i].Value == "" || r2.query[i].Value == "" {
			continue
		}
		//...If Both are value tests, so test each value againt other alphabetically.
		if r := strings.Compare(r1.query[i].Value, r2.query[i].Value); r != 0 {
			return r
		}
	}

	//Nothing more to test. Routes matches.
	return 0
}

//compareStaticRoutes compares two routes at removal on routing table. It is used to guarantee that entries do not conflict with each other.
//It is a simple static comparation. Variable path segments and query parameter and values are not taken into account.
func compareStaticRoutes(r1, r2 *muxRoute) int {
	//Compare the common static part.
	if r := compareMethodSchemeHost(r1.method, r2.method, r1.scheme, r2.scheme, r1.host, r2.host); r != 0 {
		return r
	}

	//Compare path segments
	rp1Len, rp2Len := len(r1.path), len(r2.path)
	for i := 0; i < rp1Len && i < rp2Len; i++ {
		r := strings.Compare(r1.path[i], r2.path[i])
		if r != 0 {
			return r
		}
	}
	if r := rp1Len - rp2Len; r != 0 {
		return r
	}

	//Compare query strings...
	//...Sorting larger quantities to smaller...
	q1Len, q2Len := len(r1.query), len(r2.query)
	if r := q2Len - q1Len; r != 0 {
		return r
	}
	for i := 0; i < q1Len; i++ {
		//... And each query parameter name alphabetically...
		if r := strings.Compare(r1.query[i].Name, r2.query[i].Name); r != 0 {
			return r
		}
		//... And each query parameter value alphabetically...
		if r := strings.Compare(r1.query[i].Value, r2.query[i].Value); r != 0 {
			return r
		}
	}

	//Nothing more to test. Routes matches.
	return 0
}

//compareRequestRoute compares two routes at lookup on routing table. It is used to find a entries when serving requests.
//It is similar to dynamic comparation but it assumes that only the routing side could have dynamic parts, while the request side only have static parts.
func compareRequestRoute(req *http.Request, route *muxRoute) int {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	//Compare the common static part.
	if r := compareMethodSchemeHost(
		req.Method, route.method,
		scheme, route.scheme,
		req.Host, route.host,
	); r != 0 {
		return r
	}

	//Extract path segments from request
	reqSegs := splitPathSegs(req.URL.RequestURI())

	//Compare the url path...
	reqLen, routeLen := len(reqSegs), len(route.path)
	for i := 0; i < reqLen && i < routeLen; i++ {
		//...checking if a sub-path matching is used, so any comparation at this path segment matches...
		if (i == routeLen-1) && route.path[i] == "{*}" {
			return 0
		}

		//...a variable segment tested against a request segment matches, so test the subsequent segments...
		reqSeg, routeSeg := reqSegs[i], route.path[i]
		if dynRouteSeg := strings.HasPrefix(routeSeg, "{") && strings.HasSuffix(routeSeg, "}"); dynRouteSeg {
			continue
		}

		//...and two static segments are compared using their values...
		if r := strings.Compare(reqSeg, routeSeg); r != 0 {
			return r
		}
	}

	//...if everything matches until now, compare path sizes.
	return reqLen - routeLen
}

//compareMethodSchemeHost Compares the common static url parts.
func compareMethodSchemeHost(httpMethod1, httpMethod2, scheme1, scheme2, host1, host2 string) int {
	if r := strings.Compare(httpMethod1, httpMethod2); r != 0 {
		return r
	}
	if r := strings.Compare(scheme1, scheme2); r != 0 {
		return r
	}
	if r := strings.Compare(host1, host2); r != 0 {
		return r
	}
	return 0
}
