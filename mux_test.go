// This file is part of Gopher Burrow Mux.
//
// Gopher Burrow Mux is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher Burrow Mux is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.

// You should have received a copy of the GNU Lesser General Public License
// along with Gopher Burrow Mux.  If not, see <http://www.gnu.org/licenses/>.
package mux_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/gopherburrow/mux"
)

func emptyHandler(w http.ResponseWriter, r *http.Request) {

}

type testHandler struct {
	response string
}

func newTestHandler(response string) *testHandler {
	return &testHandler{
		response: response,
	}
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, h.response)
}

func TestMux_Handle_success(t *testing.T) {
	m := &mux.Mux{}

	if err := m.Handle(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}", newTestHandler("POST+https://localhost:8080/fixed-path/{variable-path}")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}?query=a", newTestHandler("POST+https://localhost:8080/fixed-path/{variable-path}?query=a")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}?query=c", newTestHandler("POST+https://localhost:8080/fixed-path/{variable-path}?query=c")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}?query=c&query=a", newTestHandler("POST+https://localhost:8080/fixed-path/{variable-path}?query=a&query=c")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}?presence", newTestHandler("POST+https://localhost:8080/fixed-path/{variable-path}?presence")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}/fixed-subpath", newTestHandler("2")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodGet, "https://localhost:8080/fixed-path/{variable-path}/fixed-subpath", newTestHandler("3")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodGet, "https://localhost:8080/fixed-path/{variable-path}", newTestHandler("GET+https://localhost:8080/fixed-path/{variable-path}")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path/{variable-path}/{variable-subpath}", newTestHandler("5")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path/{variable-path}", newTestHandler("6")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodGet, "http://localhost/fixed-path/{variable-path}", newTestHandler("7")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodGet, "https://localhost:8080/a-query-only-path/{variable-path}?query=a", newTestHandler("GET+https://localhost:8080/a-query-only-path/{variable-path}?query=a")); err != nil {
		t.Fatal(err)
	}

	if err := m.Handle(
		http.MethodGet,
		"http://localhost/fixed/{variable-path}",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "http://localhost/fixed/{variable-path}")
		})); err != nil {
		t.Fatal(err)
	}

	if err := m.Handle(http.MethodGet, "http://localhost/root-path", newTestHandler("9")); err != nil {
		t.Fatal(err)
	}

	if err := m.Handle(http.MethodGet, "http://localhost/root-path/{*}", newTestHandler("10")); err != nil {
		t.Fatal(err)
	}

	{
		req := httptest.NewRequest(http.MethodPost, "https://localhost:8080/fixed-path/gopherburrow", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "POST+https://localhost:8080/fixed-path/{variable-path}", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodPost, "https://localhost:8080/fixed-path/gopherburrow?query=b&query=a", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "POST+https://localhost:8080/fixed-path/{variable-path}?query=a", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodPost, "https://localhost:8080/fixed-path/gopherburrow?query&query=a", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "POST+https://localhost:8080/fixed-path/{variable-path}?query=a", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodPost, "https://localhost:8080/fixed-path/gopherburrow?query=b&votz=test&query=c", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "POST+https://localhost:8080/fixed-path/{variable-path}?query=c", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodPost, "https://localhost:8080/fixed-path/gopherburrow?query=a&votz=test&query=c", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "POST+https://localhost:8080/fixed-path/{variable-path}?query=a&query=c", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodPost, "https://localhost:8080/fixed-path/gopherburrow?query=b&query=c&votz=x&query=d", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "POST+https://localhost:8080/fixed-path/{variable-path}?query=c", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodPost, "https://localhost:8080/fixed-path/gopherburrow?presence=yes", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "POST+https://localhost:8080/fixed-path/{variable-path}?presence", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodPost, "https://localhost:8080/fixed-path/gopherburrow?presence", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "POST+https://localhost:8080/fixed-path/{variable-path}?presence", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodPost, "https://localhost:8080/fixed-path/gopherburrow?notpresence&presence", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "POST+https://localhost:8080/fixed-path/{variable-path}?presence", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodPost, "https://localhost:8080/fixed-path/gopherburrow/fixed-subpath", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "2", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "https://localhost:8080/fixed-path/gopherburrow/fixed-subpath", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "3", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "https://localhost:8080/fixed-path/gopherburrow", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "GET+https://localhost:8080/fixed-path/{variable-path}", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/fixed-path/gopher/burrow", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "5", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/fixed-path/gopherburrow", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "6", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "http://localhost/fixed-path/gopherburrow", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "7", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "http://localhost/fixed/gopherburrow", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "http://localhost/fixed/{variable-path}", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "http://localhost/root-path", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "9", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "http://localhost/root-path/gopher", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "10", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "http://localhost/root-path/gopher/burrow", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "10", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "https://localhost:8080/a-query-only-path/gopherburrow?query=a", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "GET+https://localhost:8080/a-query-only-path/{variable-path}?query=a", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "http://localhost/not-found/gopherburrow", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := http.StatusNotFound, rr.Code; want != got {
			t.Fatalf("want=%d, got=%d", want, got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "https://localhost:8080/a-query-only-path/gopherburrow?query=b", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := http.StatusNotFound, rr.Code; want != got {
			t.Fatalf("want=%d, got=%d", want, got)
		}
	}

	m.NotFoundHandler = newTestHandler("Not Found but OK")

	{
		req := httptest.NewRequest(http.MethodGet, "https://localhost:8080/a-query-only-path/gopherburrow?query=b", nil)
		rr := httptest.NewRecorder()
		m.ServeHTTP(rr, req)
		if want, got := "Not Found but OK", rr.Body.String(); want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	if want, got := `GET+http://localhost/fixed/{variable-path}
GET+http://localhost/fixed-path/{variable-path}
GET+http://localhost/root-path
GET+http://localhost/root-path/{*}
GET+http://localhost:8080/fixed-path/{variable-path}
GET+http://localhost:8080/fixed-path/{variable-path}/{variable-subpath}
GET+https://localhost:8080/a-query-only-path/{variable-path}?query=a
GET+https://localhost:8080/fixed-path/{variable-path}
GET+https://localhost:8080/fixed-path/{variable-path}/fixed-subpath
POST+https://localhost:8080/fixed-path/{variable-path}?query=a&query=c
POST+https://localhost:8080/fixed-path/{variable-path}?presence
POST+https://localhost:8080/fixed-path/{variable-path}?query=a
POST+https://localhost:8080/fixed-path/{variable-path}?query=c
POST+https://localhost:8080/fixed-path/{variable-path}
POST+https://localhost:8080/fixed-path/{variable-path}/fixed-subpath
`, m.String(); want != got {
		t.Fatalf("want=%q, got=%q", want, got)
	}
}

func TestMux_Handle_successInRootPathAndRootWildcard(t *testing.T) {
	m := &mux.Mux{}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/{*}", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}
}

func TestMux_Handle_successInPathAndPathWildcard(t *testing.T) {
	m := &mux.Mux{}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/test", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/test/{*}", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}
}

func TestMux_Handle_failHttpMethodMustBeNotEmpty(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle("", "http://localhost:8080/fixed-path/{variable-path}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrMethodMustBeValid {
		t.Fatal("expected: mux.ErrMethodMustBeValid")
	}
}

func TestMux_Handle_failHttpMethodMustBeAllowedValue(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle("FAIL", "http://localhost:8080/fixed-path/{variable-path}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrMethodMustBeValid {
		t.Fatal("expected: mux.ErrMethodMustBeValid")
	}
}

func TestMux_Handle_failUrlPatternMustBeNotEmpty(t *testing.T) {
	m := mux.Mux{}
	err := m.Handle(http.MethodGet, "", http.HandlerFunc(emptyHandler))
	if err != mux.ErrURLPatternMustBeValid {
		t.Fatal("expected: mux.ErrURLPatternMustBeValid")
	}
}
func TestMux_Handle_failUrlPatternMustBeValid(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, ":8080/fixed-path/{variable-path}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrURLPatternMustBeValid {
		t.Fatal("expected: mux.ErrURLPatternMustBeValid")
	}
}

func TestMux_Handle_failUrlPatternMustBeAbsoluteUrl(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "/fixed-path/{variable-path}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrURLPatternMustBeValid {
		t.Fatal("expected: mux.ErrURLPatternMustBeValid")
	}
}

func TestMux_Handle_failInvalidScheme(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "ftp://localhost:21", http.HandlerFunc(emptyHandler))
	if err != mux.ErrURLPatternMustBeValid {
		t.Fatal("expected: mux.ErrURLPatternMustBeValid")
	}
}

func TestMux_Handle_failUrlPatternHostMustNotBeEmpty(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "http:///fixed-path/{variable-path}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrURLPatternMustBeValid {
		t.Fatal("expected: mux.ErrURLPatternMustBeValid")
	}
}

func TestMux_Handle_failUrlPatternHostMustHaveHostName(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "http://:8080/fixed-path/{variable-path}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrURLPatternMustBeValid {
		t.Fatal("expected: mux.ErrURLPatternMustBeValid")
	}
}

func TestMux_Handle_failHandlerMustBeNotNil(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path/{variable-path}", nil)
	if err != mux.ErrHandlerMustBeNotNil {
		t.Fatal("expected: mux.ErrHandlerMustBeNotNil")
	}
}

func TestMux_Handle_failMustNotConflictingWithExistingEntryVariableVsFixed(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path/{variable-path}", http.HandlerFunc(emptyHandler))
	if err != nil {
		t.Fatal(err)
	}
	err = m.Handle(http.MethodGet, "http://localhost:8080/fixed-path/fixed-path2", http.HandlerFunc(emptyHandler))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
}

func TestMux_Handle_failMustNotConflictingWithExistingEntryFixedVsVariableSamePathSizes(t *testing.T) {
	m := &mux.Mux{}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path1/{variable-path}", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path1/{variable-path}/fixed-path2", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}
	err := m.Handle(http.MethodGet, "http://localhost:8080/{variable-path}/{variable-path2}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
	err = m.Handle(http.MethodGet, "http://localhost:8080/fixed-path1/{variable-path}/{variable-path2}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
}

func TestMux_Handle_failMustNotConflictingWithExistingEntryFixedVsVariableDifferentPathSizes(t *testing.T) {
	m := &mux.Mux{}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path1/{variable-path}", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path1/{variable-path}/fixed-path2", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}
	err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path1/{variable-path}/fixed-path2", http.HandlerFunc(emptyHandler))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
	err = m.Handle(http.MethodGet, "http://localhost:8080/{variable-path}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
}

func TestMux_Handle_failMustNotConflictingWithExistingEntryParentVsFixed(t *testing.T) {
	m := &mux.Mux{}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path/{*}", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}
	err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path/gopher/burrow", http.HandlerFunc(emptyHandler))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
}

func TestMux_Handle_failMustNotConflictingWithExistingEntryFixedVsParent(t *testing.T) {
	m := &mux.Mux{}
	if err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path/gopher/burrow", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}
	err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path/{*}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
	err = m.Handle(http.MethodGet, "http://localhost:8080/fixed-path/gopher/{*}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
}

func TestMux_Handle_failMustNotConflictingWithExistingEntryFixedTrailingSlashes(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "http://localhost:8080/fixed-path/", http.HandlerFunc(emptyHandler))
	if err != nil {
		t.Fatal(err)
	}
	err = m.Handle(http.MethodGet, "http://localhost:8080/fixed-path", http.HandlerFunc(emptyHandler))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
}

func TestMux_Handle_failMustNotConflictingWithExistingEntryVariableTrailingSlashes(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "http://localhost:8080/{variable-path}/", http.HandlerFunc(emptyHandler))
	if err != nil {
		t.Fatal(err)
	}
	err = m.Handle(http.MethodGet, "http://localhost:8080/{variable-path}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
}

func TestMux_Handle_failMustNotConflictingWithExistingEntryQueryWithoutValue(t *testing.T) {
	m := &mux.Mux{}

	if err := m.Handle(http.MethodPut, "https://localhost:8080?a", newTestHandler("PUT+https://localhost:8080?a")); err != nil {
		t.Fatal(err)
	}
	if err := m.Handle(http.MethodPut, "https://localhost:8080?b", newTestHandler("PUT+https://localhost:8080?b")); err != nil {
		t.Fatal(err)
	}
	err := m.Handle(http.MethodPut, "https://localhost:8080?a", newTestHandler("PUT+https://localhost:8080?a"))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
}

func TestMux_Handle_failMustNotConflictingWithExistingEntryQueryWithValue(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "https://localhost:8080/fixed-path/{variable-path}?query1=a", http.HandlerFunc(emptyHandler))
	if err != nil {
		t.Fatal(err)
	}
	err = m.Handle(http.MethodGet, "https://localhost:8080/fixed-path/{variable-path}?query1=a", http.HandlerFunc(emptyHandler))
	if err != mux.ErrRouteMustNotConflict {
		t.Fatal("expected: mux.ErrRouteMustNotConflict")
	}
}

func TestMux_Handle_failMustNotHaveEmptyVarName(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "http://localhost:8080/{}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrURLPatternInvalidPathVar {
		t.Fatal("expected: mux.ErrURLPatternInvalidPathVar")
	}
}

func TestMux_Handle_failPathVarMustBeLastParameter(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "http://localhost:8080/{*}/{variable-path}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrURLPatternInvalidPathVar {
		t.Fatal("expected: mux.ErrURLPatternInvalidPathVar")
	}
}

func TestMux_Handle_failMustNotHaveConflitingVars(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "http://localhost:8080/{var}/{var}", http.HandlerFunc(emptyHandler))
	if err != mux.ErrURLPatternInvalidPathVar {
		t.Fatal("expected: mux.ErrURLPatternInvalidPathVar")
	}
}

func TestMux_Handle_failUniquePresenceTest(t *testing.T) {
	m := &mux.Mux{}
	err := m.Handle(http.MethodGet, "http://localhost:8080?presenceTest&presenceTest=present", http.HandlerFunc(emptyHandler))
	if err != mux.ErrURLPatternInvalidQueryRoute {
		t.Fatal("expected: mux.ErrURLPatternInvalidQueryRoute")
	}

	err = m.Handle(http.MethodGet, "http://localhost:8080?presenceTest=present&presenceTest", http.HandlerFunc(emptyHandler))
	if err != mux.ErrURLPatternInvalidQueryRoute {
		t.Fatal("expected: mux.ErrURLPatternInvalidQueryRoute")
	}
}

func TestMux_RemoveHandler_success(t *testing.T) {
	m := &mux.Mux{}

	if err := m.Handle(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}/fixed-path/fixed-path", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}

	if err := m.Handle(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}

	if err := m.Handle(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}?p1&p2=value", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}

	if err := m.Handle(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}/fixed-path", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}

	if err := m.RemoveHandler(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}"); err != nil {
		t.Fatal(err)
	}

	if err := m.RemoveHandler(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}/fixed-path/fixed-path"); err != nil {
		t.Fatal(err)
	}

	if err := m.RemoveHandler(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}?p2=value&p1"); err != nil {
		t.Fatal(err)
	}

	if err := m.RemoveHandler(http.MethodPost, "https://localhost:8080/fixed-path/{variable-path}/fixed-path"); err != nil {
		t.Fatal(err)
	}
}

func TestMux_RemoveHandler_failHttpMethodMustBeNotEmpty(t *testing.T) {
	m := &mux.Mux{}
	err := m.RemoveHandler("", "http://localhost:8080/fixed-path/{variable-path}")
	if err != mux.ErrMethodMustBeValid {
		t.Fatal("expected: mux.ErrMethodMustBeValid")
	}
}

func TestMux_RemoveHandler_failHttpMethodMustBeAllowedValue(t *testing.T) {
	m := &mux.Mux{}
	err := m.RemoveHandler("FAIL", "http://localhost:8080/fixed-path/{variable-path}")
	if err != mux.ErrMethodMustBeValid {
		t.Fatal("expected: mux.ErrMethodMustBeValid")
	}
}

func TestMux_RemoveHandler_failUrlPatternMustBeNotEmpty(t *testing.T) {
	m := &mux.Mux{}
	err := m.RemoveHandler(http.MethodGet, "")
	if err != mux.ErrURLPatternMustBeValid {
		t.Fatal("expected: mux.ErrURLPatternMustBeValid")
	}
}
func TestMux_RemoveHandler_failUrlPatternMustBeValid(t *testing.T) {
	m := &mux.Mux{}
	err := m.RemoveHandler(http.MethodGet, ":8080/fixed-path/{variable-path}")
	if err != mux.ErrURLPatternMustBeValid {
		t.Fatal("expected: mux.ErrURLPatternMustBeValid")
	}
}

func TestMux_RemoveHandler_failUrlPatternMustBeAbsoluteUrl(t *testing.T) {
	m := &mux.Mux{}
	err := m.RemoveHandler(http.MethodGet, "/fixed-path/{variable-path}")
	if err != mux.ErrURLPatternMustBeValid {
		t.Fatal("expected: mux.ErrURLPatternMustBeValid")
	}
}

func TestMux_RemoveHandler_failUrlPatternHostMustNotBeEmpty(t *testing.T) {
	m := &mux.Mux{}
	err := m.RemoveHandler(http.MethodGet, "http:///fixed-path/{variable-path}")
	if err != mux.ErrURLPatternMustBeValid {
		t.Fatal("expected: mux.ErrURLPatternMustBeValid")
	}
}

func TestMux_RemoveHandler_failUrlPatternHostMustHaveHostName(t *testing.T) {
	m := &mux.Mux{}
	err := m.RemoveHandler(http.MethodGet, "http://:8080/fixed-path/{variable-path}")
	if err != mux.ErrURLPatternMustBeValid {
		t.Fatal("expected: mux.ErrURLPatternMustBeValid")
	}
}

func TestMux_RemoveHandler_failMustExists(t *testing.T) {
	m := &mux.Mux{}
	if err := m.Handle(http.MethodGet, "https://localhost:8080/fixed-path", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}

	if err := m.Handle(http.MethodPost, "https://localhost:8080/fixed-path?p2&p3=value", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}

	err := m.RemoveHandler(http.MethodGet, "https://localhost:8080/{fixed-path}")
	if err != mux.ErrRouteMustExist {
		t.Fatal("expected: mux.ErrRouteMustExist")
	}
	err = m.RemoveHandler(http.MethodGet, "http://localhost:8080/fixed-path")
	if err != mux.ErrRouteMustExist {
		t.Fatal("expected: mux.ErrRouteMustExist")
	}
	err = m.RemoveHandler(http.MethodGet, "https://localhost:8080")
	if err != mux.ErrRouteMustExist {
		t.Fatal("expected: mux.ErrRouteMustExist")
	}
	err = m.RemoveHandler(http.MethodPost, "https://localhost:8080/fixed-path")
	if err != mux.ErrRouteMustExist {
		t.Fatal("expected: mux.ErrRouteMustExist")
	}
	err = m.RemoveHandler(http.MethodGet, "https://example.com/fixed-path")
	if err != mux.ErrRouteMustExist {
		t.Fatal("expected: mux.ErrRouteMustExist")
	}
	err = m.RemoveHandler(http.MethodPost, "https://localhost:8080/fixed-path?p3&p2")
	if err != mux.ErrRouteMustExist {
		t.Fatal("expected: mux.ErrRouteMustExist")
	}
	err = m.RemoveHandler(http.MethodPost, "https://localhost:8080/fixed-path?p3&p1")
	if err != mux.ErrRouteMustExist {
		t.Fatal("expected: mux.ErrRouteMustExist")
	}
}

func TestMux_PathVars_success(t *testing.T) {
	m := &mux.Mux{}
	if err := m.Handle(http.MethodGet, "https://localhost:8080/fixed-path/{var1}/{ var2 }", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}

	if err := m.Handle(http.MethodGet, "https://localhost:8080/parent-path/{*}", http.HandlerFunc(emptyHandler)); err != nil {
		t.Fatal(err)
	}

	{
		r := httptest.NewRequest(http.MethodGet, "https://localhost:8080/fixed-path/gopher/burrow", nil)
		if want, got := "gopher", m.PathVars(r)["var1"]; want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}

		if want, got := "burrow", m.PathVars(r)["var2"]; want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		r := httptest.NewRequest(http.MethodGet, "https://localhost:8080/parent-path/gopher/burrow", nil)
		if want, got := "gopher/burrow", m.PathVars(r)["*"]; want != got {
			t.Fatalf("want=%q, got=%q", want, got)
		}
	}

	{
		m2 := &mux.Mux{}
		r := httptest.NewRequest(http.MethodGet, "https://localhost:8080/fixed-path/gopher/burrow", nil)
		_, f := m2.PathVars(r)["var1"]
		if want, got := false, f; want != got {
			t.Fatalf("want=%t, got=%t", want, got)
		}
	}
}

func TestGet_success(t *testing.T) {
	m := &mux.Mux{}
	m.Handle(http.MethodGet, "http://localhost/{*}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := mux.Get(r)
		if err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://localhost/gopher/burrow/", nil)
	rr := httptest.NewRecorder()
	m.ServeHTTP(rr, req)
	if want, got := http.StatusOK, rr.Code; want != got {
		t.Fatalf("want=%d, got=%d", want, got)
	}
}

func TestGet_failMustHaveContext(t *testing.T) {
	m := &mux.Mux{}
	m.Handle(http.MethodGet, "http://localhost/{*}", http.HandlerFunc(emptyHandler))
	r := httptest.NewRequest(http.MethodGet, "http://localhost/gopher/burrow/", nil)
	m, err := mux.Get(r)
	if err != mux.ErrRequestMustHaveContext {
		t.Fatal("expected: mux.ErrRequestMustHaveContext")
	}
	if m != nil {
		t.Fatal("expected: nil")
	}
}

func ExampleMux() {
	m := &mux.Mux{}
	m.Handle(http.MethodGet, "http://localhost/{var}/{*}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mx, err := mux.Get(r)
		if err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello World %q %q\n", mx.PathVars(r)["var"], mx.PathVars(r)["*"])
	}))

	// On real world we would use:
	//  http.ListenAndServe(":8080", m)

	// On tests world we use:
	req := httptest.NewRequest(http.MethodGet, "http://localhost/gopher/burrow/mux", nil)
	rr := httptest.NewRecorder()
	m.ServeHTTP(rr, req)
	fmt.Print(rr.Body.String())

	// Output: Hello World "gopher" "burrow/mux"
}
