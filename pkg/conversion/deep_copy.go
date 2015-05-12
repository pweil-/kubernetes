/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conversion

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"sync"
)

// pool is a pool of copiers
var pool = sync.Pool{
	New: func() interface{} { return newGobCopier() },
}

// nonGobCopier is used for restricted types
var nonGobCopier = NewConverter()

// DeepCopy makes a deep copy of source or returns an error.
func DeepCopy(source interface{}) (interface{}, error) {
	name := reflect.TypeOf(source).Name()
	return deepCopyInternal(source, isRestrictedType(name))
}

func deepCopyInternal(source interface{}, restrictedType bool) (interface{}, error) {
	if restrictedType {
		src := reflect.ValueOf(source)
		v := reflect.New(src.Type()).Elem()
		s := &scope{
			converter: nonGobCopier,
		}
		if err := nonGobCopier.convert(src, v, s); err != nil {
			return nil, err
		}
		return v.Interface(), nil
	}else{
		v := reflect.New(reflect.TypeOf(source))
		c := pool.Get().(gobCopier)
		defer pool.Put(c)
		if err := c.CopyInto(v.Interface(), source); err != nil {
			return nil, err
		}
		return v.Elem().Interface(), nil
	}
}

// isRestrictedType identifies types that contain pointers to simple types
func isRestrictedType(typeName string) bool {
	switch typeName{
	case "Pod", "RestrictedTestType":
		return true
	}
	return false
}

// gobCopier provides a copy mechanism for objects using Gob.
// This object is not safe for multiple threads because buffer
// is shared.
type gobCopier struct {
	enc *gob.Encoder
	dec *gob.Decoder
}

func newGobCopier() gobCopier {
	buf := &bytes.Buffer{}
	return gobCopier{
		enc: gob.NewEncoder(buf),
		dec: gob.NewDecoder(buf),
	}
}

func (c *gobCopier) CopyInto(dst, src interface{}) error {
	if err := c.enc.Encode(src); err != nil {
		return err
	}
	if err := c.dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

// RestrictedTestType is exposed to allow testing a restricted type
type RestrictedTestType struct {
	I *int
	B *bool
	S *string
}
