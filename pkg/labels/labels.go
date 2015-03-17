/*
Copyright 2014 Google Inc. All rights reserved.

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

package labels

import (
	"sort"
	"strings"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels/types"
)

type labelSet struct {
	types.Set
}

func NewLabels(s types.Set) types.Labels {
	return labelSet{s}
}

func NewLabelsFromMap(s map[string]string) types.Labels {
	return labelSet{types.Set(s)}
}

func EmptyLabels() types.Labels {
	return labelSet{}
}

// used by some testing.  If you want a selector and have a raw map use
// NewLabelsFromMap rather than chaining RawLabelSet and NewLabels together
func RawLabelSet(s map[string]string) types.Set {
	return types.Set(s)
}

// String returns all labels listed as a human readable string.
// Conveniently, exactly the format that ParseSelector takes.
func (ls labelSet) String() string {
	selector := make([]string, 0, len(ls.Set))
	for key, value := range ls.Set {
		selector = append(selector, key+"="+value)
	}
	// Sort for determinism.
	sort.StringSlice(selector).Sort()
	return strings.Join(selector, ",")
}

// Has returns whether the provided label exists in the map.
func (ls labelSet) Has(label string) bool {
	_, exists := ls.Set[label]
	return exists
}

// Get returns the value in the map for the provided label.
func (ls labelSet) Get(label string) string {
	return ls.Set[label]
}

// AsSelector converts labels into a selectors.
func (ls labelSet) AsSelector() types.Selector {
	return SelectorFromSet(ls.Set)
}
