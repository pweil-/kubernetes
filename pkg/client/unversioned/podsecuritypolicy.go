/*
Copyright 2014 The Kubernetes Authors All rights reserved.

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

package unversioned

import (
	"k8s.io/kubernetes/pkg/apis/experimental"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

type PodSecurityPoliciesInterface interface {
	PodSecurityPolicies() PodSecurityPolicyInterface
}

type PodSecurityPolicyInterface interface {
	Get(name string) (result *experimental.PodSecurityPolicy, err error)
	Create(scc *experimental.PodSecurityPolicy) (*experimental.PodSecurityPolicy, error)
	List(label labels.Selector, field fields.Selector) (*experimental.PodSecurityPolicyList, error)
	Delete(name string) error
	Update(*experimental.PodSecurityPolicy) (*experimental.PodSecurityPolicy, error)
	Watch(label labels.Selector, field fields.Selector, resourceVersion string) (watch.Interface, error)
}

// podSecurityPolicy implements PodSecurityPolicyInterface
type podSecurityPolicy struct {
	client *ExperimentalClient
}

// newPodSecurityPolicy returns a podSecurityPolicy object.
func newPodSecurityPolicy(c *ExperimentalClient) *podSecurityPolicy {
	return &podSecurityPolicy{c}
}

func (s *podSecurityPolicy) Create(scc *experimental.PodSecurityPolicy) (*experimental.PodSecurityPolicy, error) {
	result := &experimental.PodSecurityPolicy{}
	err := s.client.Post().
		Resource("podsecuritypolicies").
		Body(scc).
		Do().
		Into(result)

	return result, err
}

// List returns a list of PodSecurityPolicies matching the selectors.
func (s *podSecurityPolicy) List(label labels.Selector, field fields.Selector) (*experimental.PodSecurityPolicyList, error) {
	result := &experimental.PodSecurityPolicyList{}

	err := s.client.Get().
		Resource("podsecuritypolicies").
		LabelsSelectorParam(label).
		FieldsSelectorParam(field).
		Do().
		Into(result)

	return result, err
}

// Get returns the given PodSecurityPolicy, or an error.
func (s *podSecurityPolicy) Get(name string) (*experimental.PodSecurityPolicy, error) {
	result := &experimental.PodSecurityPolicy{}
	err := s.client.Get().
		Resource("podsecuritypolicies").
		Name(name).
		Do().
		Into(result)

	return result, err
}

// Watch starts watching for PodSecurityPolicies matching the given selectors.
func (s *podSecurityPolicy) Watch(label labels.Selector, field fields.Selector, resourceVersion string) (watch.Interface, error) {
	return s.client.Get().
		Prefix("watch").
		Resource("podsecuritypolicies").
		Param("resourceVersion", resourceVersion).
		LabelsSelectorParam(label).
		FieldsSelectorParam(field).
		Watch()
}

func (s *podSecurityPolicy) Delete(name string) error {
	return s.client.Delete().
		Resource("podsecuritypolicies").
		Name(name).
		Do().
		Error()
}

func (s *podSecurityPolicy) Update(psp *experimental.PodSecurityPolicy) (result *experimental.PodSecurityPolicy, err error) {
	result = &experimental.PodSecurityPolicy{}
	err = s.client.Put().
		Resource("podsecuritypolicies").
		Name(psp.Name).
		Body(psp).
		Do().
		Into(result)

	return
}
