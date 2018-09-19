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

package mux

import (
	"sort"
	"strings"
)

//ContainsString TODO
func containsString(stringSlice []string, value string) bool {
	for _, stringSliceValue := range stringSlice {
		if stringSliceValue == value {
			return true
		}
	}
	return false
}

func searchRange(n int, fn func(i int) int) (lo int, hi int, found bool) {
	lo = sort.Search(n, func(i int) bool {
		return fn(i) <= 0
	})

	hi = sort.Search(n, func(i int) bool {
		return fn(i) < 0
	})
	found = (hi - lo) > 0
	return
}

func splitPathSegs(path string) []string {
	pathSegs := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathSegs) == 1 && pathSegs[0] == "" {
		pathSegs = []string{}
	}
	return pathSegs
}
