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

package sim

import (
	"bytes"
	"testing"
)

func TestZipfian(t *testing.T) {
	s := NewZipfian(1.25, 2, 100)
	for i := 0; i < 100; i++ {
		if _, err := s(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUniform(t *testing.T) {
	s := NewUniform(100)
	for i := 0; i < 100; i++ {
		if _, err := s(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestParseLirs(t *testing.T) {
	s := NewReader(ParseLirs, bytes.NewReader([]byte{
		'0', '\r', '\n',
		'1', '\r', '\n',
		'2', '\r', '\n',
	}))
	for i := uint64(0); i < 3; i++ {
		v, err := s()
		if err != nil {
			t.Fatal(err)
		}
		if v != i {
			t.Fatal("value mismatch")
		}
	}
}

func TestCollect(t *testing.T) {
	s := NewUniform(100)
	c := Collection(s, 100)
	if len(c) != 100 {
		t.Fatal("collection not full")
	}
}
