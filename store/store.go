/*
 * Copyright 2019 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package store

// Map is the interface fulfilled by all hash map implementations in the store
// package. Some hash map implementations are better suited for certain data
// distributions than others, so this allows us to abstract that out for use
// in Ristretto.
//
// Every Map is safe for concurrent usage.
type Map interface {
	// Get returns the value associated with the key parameter.
	Get(string) interface{}
	// Set adds the key-value pair to the Map or updates the value if it's
	// already present.
	Set(string, interface{})
	// Del deletes the key-value pair from the Map.
	Del(string)
	// Run applies the function parameter to random key-value pairs. No key
	// will be visited more than once. If the function returns false, the
	// iteration stops. If the function returns true, the iteration will
	// continue until every key has been visited once.
	Run(func(interface{}, interface{}) bool)
}

// NewMap returns the Default Map implementation.
func NewMap() Map {
	return NewDefault()
}
