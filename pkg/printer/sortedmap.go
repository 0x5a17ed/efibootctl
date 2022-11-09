/*
The MIT License (MIT)

Copyright (c) 2015 Takashi Kokubun
Copyright (c) 2022 Arthur Skowronek

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package printer

import (
	"reflect"
	"sort"
)

type sortedMap struct{ keys, values []reflect.Value }

func (s *sortedMap) Len() int { return len(s.keys) }

func (s *sortedMap) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	s.values[i], s.values[j] = s.values[j], s.values[i]
}

func (s *sortedMap) Less(i, j int) bool {
	a, b := s.keys[i], s.keys[j]
	if a.Type() != b.Type() {
		return false // give up
	}

	// Return true if b is bigger
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() < b.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return a.Uint() < b.Uint()
	case reflect.String:
		return a.String() < b.String()
	case reflect.Float32, reflect.Float64:
		if a.Float() != a.Float() || b.Float() != b.Float() {
			return false // NaN
		}
		return a.Float() < b.Float()
	case reflect.Bool:
		return !a.Bool() && b.Bool()
	case reflect.Ptr:
		return a.Pointer() < b.Pointer()
	case reflect.Struct:
		return a.NumField() < b.NumField()
	case reflect.Array:
		return a.Len() < b.Len()
	default:
		return false // not supported yet
	}
}

func sortMap(value reflect.Value) *sortedMap {
	if value.Type().Kind() != reflect.Map {
		panic("sortMap is used for a non-Map value")
	}

	keys := make([]reflect.Value, 0, value.Len())
	values := make([]reflect.Value, 0, value.Len())
	mapKeys := value.MapKeys()
	for i := 0; i < len(mapKeys); i++ {
		keys = append(keys, mapKeys[i])
		values = append(values, value.MapIndex(mapKeys[i]))
	}

	sorted := &sortedMap{
		keys:   keys,
		values: values,
	}
	sort.Stable(sorted)
	return sorted
}
