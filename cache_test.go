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

 package ristretto

 import (
	"bytes"
	"testing"
)


func TestGet(t *testing.T){	
	k := []byte("1")
	v := []byte("one")
	cache := New(0)
	cache.Set(k,v)
	val, err := cache.Get(k)
	if err != nil {
		t.Fatalf(err.Error())
	} else if !bytes.Equal(val, v) {
		t.Fatalf("%s expected %v but got %v", k, v, val)
	}
}