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

package conversion_test

import (
	"io/ioutil"
	"math/rand"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/conversion"

	"github.com/google/gofuzz"
)

func TestDeepCopy(t *testing.T) {
	semantic := conversion.EqualitiesOrDie()
	f := fuzz.New().NilChance(.5).NumElements(0, 100)
	table := []interface{}{
		map[string]string{},
		int(5),
		"hello world",
		struct {
			A, B, C struct {
				D map[string]int
			}
			X []int
			Y []byte
		}{},
	}
	for _, obj := range table {
		obj2, err := conversion.DeepCopy(obj)
		if err != nil {
			t.Errorf("Error: couldn't copy %#v", obj)
			continue
		}
		if e, a := obj, obj2; !semantic.DeepEqual(e, a) {
			t.Errorf("expected %#v\ngot %#v", e, a)
		}

		obj3 := reflect.New(reflect.TypeOf(obj)).Interface()
		f.Fuzz(obj3)
		obj4, err := conversion.DeepCopy(obj3)
		if err != nil {
			t.Errorf("Error: couldn't copy %#v", obj)
			continue
		}
		if e, a := obj3, obj4; !semantic.DeepEqual(e, a) {
			t.Errorf("expected %#v\ngot %#v", e, a)
		}
		f.Fuzz(obj3)
	}
}

func copyOrDie(t *testing.T, in interface{}) interface{} {
	out, err := conversion.DeepCopy(in)
	if err != nil {
		t.Fatalf("DeepCopy failed: %#q: %v", in, err)
	}
	return out
}

func TestDeepCopySliceSeparate(t *testing.T) {
	x := []int{5}
	y := copyOrDie(t, x).([]int)
	x[0] = 3
	if y[0] == 3 {
		t.Errorf("deep copy wasn't deep: %#q %#q", x, y)
	}
}

func TestDeepCopyArraySeparate(t *testing.T) {
	x := [1]int{5}
	y := copyOrDie(t, x).([1]int)
	x[0] = 3
	if y[0] == 3 {
		t.Errorf("deep copy wasn't deep: %#q %#q", x, y)
	}
}

func TestDeepCopyMapSeparate(t *testing.T) {
	x := map[string]int{"foo": 5}
	y := copyOrDie(t, x).(map[string]int)
	x["foo"] = 3
	if y["foo"] == 3 {
		t.Errorf("deep copy wasn't deep: %#q %#q", x, y)
	}
}

func TestDeepCopyPointerSeparate(t *testing.T) {
	z := 5
	x := &z
	y := copyOrDie(t, x).(*int)
	*x = 3
	if *y == 3 {
		t.Errorf("deep copy wasn't deep: %#q %#q", x, y)
	}
}

var result interface{}

func BenchmarkDeepCopy(b *testing.B) {
	table := []interface{}{
		map[string]string{},
		int(5),
		"hello world",
		struct {
			A, B, C struct {
				D map[string]int
			}
			X []int
			Y []byte
		}{},
	}

	f := fuzz.New().RandSource(rand.NewSource(1)).NilChance(.5).NumElements(0, 100)
	for i := range table {
		out := table[i]
		obj := reflect.New(reflect.TypeOf(out)).Interface()
		f.Fuzz(obj)
		table[i] = obj
	}

	b.ResetTimer()
	var r interface{}
	for i := 0; i < b.N; i++ {
		for j := range table {
			r, _ = conversion.DeepCopy(table[j])
		}
	}
	result = r
}

func BenchmarkPodCopy(b *testing.B) {
	data, err := ioutil.ReadFile("pod_example.json")
	if err != nil {
		b.Fatalf("unexpected error while reading file: %v", err)
	}
	var pod api.Pod
	if err := api.Scheme.DecodeInto(data, &pod); err != nil {
		b.Fatalf("unexpected error decoding pod: %v", err)
	}

	var result *api.Pod
	for i := 0; i < b.N; i++ {
		obj, err := conversion.DeepCopy(&pod)
		if err != nil {
			b.Fatalf("unexpected error copying pod: %v", err)
		}
		result = obj.(*api.Pod)
	}
	if !api.Semantic.DeepEqual(pod, *result) {
		b.Fatalf("incorrect copy: expected %v, got %v", pod, *result)
	}
}

func BenchmarkNodeCopy(b *testing.B) {
	data, err := ioutil.ReadFile("node_example.json")
	if err != nil {
		b.Fatalf("unexpected error while reading file: %v", err)
	}
	var node api.Node
	if err := api.Scheme.DecodeInto(data, &node); err != nil {
		b.Fatalf("unexpected error decoding node: %v", err)
	}

	var result *api.Node
	for i := 0; i < b.N; i++ {
		obj, err := conversion.DeepCopy(&node)
		if err != nil {
			b.Fatalf("unexpected error copying node: %v", err)
		}
		result = obj.(*api.Node)
	}
	if !api.Semantic.DeepEqual(node, *result) {
		b.Fatalf("incorrect copy: expected %v, got %v", node, *result)
	}
}

func BenchmarkReplicationControllerCopy(b *testing.B) {
	data, err := ioutil.ReadFile("replication_controller_example.json")
	if err != nil {
		b.Fatalf("unexpected error while reading file: %v", err)
	}
	var replicationController api.ReplicationController
	if err := api.Scheme.DecodeInto(data, &replicationController); err != nil {
		b.Fatalf("unexpected error decoding node: %v", err)
	}

	var result *api.ReplicationController
	for i := 0; i < b.N; i++ {
		obj, err := conversion.DeepCopy(&replicationController)
		if err != nil {
			b.Fatalf("unexpected error copying replication controller: %v", err)
		}
		result = obj.(*api.ReplicationController)
	}
	if !api.Semantic.DeepEqual(replicationController, *result) {
		b.Fatalf("incorrect copy: expected %v, got %v", replicationController, *result)
	}
}
