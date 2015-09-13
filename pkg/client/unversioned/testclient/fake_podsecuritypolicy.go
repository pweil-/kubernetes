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

package testclient

import (
	"k8s.io/kubernetes/pkg/apis/experimental"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

// FakePodSecurityPolicy implements PodSecurityPolicyInterface. Meant to be
// embedded into a struct to get a default implementation. This makes faking out just
// the method you want to test easier.
type FakePodSecurityPolicy struct {
	Fake      *Fake
	Namespace string
}

func (c *FakePodSecurityPolicy) List(label labels.Selector, field fields.Selector) (*experimental.PodSecurityPolicyList, error) {
	obj, err := c.Fake.Invokes(NewListAction("podsecuritypolicies", c.Namespace, label, field), &experimental.PodSecurityPolicyList{})
	return obj.(*experimental.PodSecurityPolicyList), err
}

func (c *FakePodSecurityPolicy) Get(name string) (*experimental.PodSecurityPolicy, error) {
	obj, err := c.Fake.Invokes(NewGetAction("podsecuritypolicies", c.Namespace, name), &experimental.PodSecurityPolicy{})
	return obj.(*experimental.PodSecurityPolicy), err
}

func (c *FakePodSecurityPolicy) Create(scc *experimental.PodSecurityPolicy) (*experimental.PodSecurityPolicy, error) {
	obj, err := c.Fake.Invokes(NewCreateAction("podsecuritypolicies", c.Namespace, scc), &experimental.PodSecurityPolicy{})
	return obj.(*experimental.PodSecurityPolicy), err
}

func (c *FakePodSecurityPolicy) Update(scc *experimental.PodSecurityPolicy) (*experimental.PodSecurityPolicy, error) {
	obj, err := c.Fake.Invokes(NewUpdateAction("podsecuritypolicies", c.Namespace, scc), &experimental.PodSecurityPolicy{})
	return obj.(*experimental.PodSecurityPolicy), err
}

func (c *FakePodSecurityPolicy) Delete(name string) error {
	_, err := c.Fake.Invokes(NewDeleteAction("podsecuritypolicies", c.Namespace, name), &experimental.PodSecurityPolicy{})
	return err
}

func (c *FakePodSecurityPolicy) Watch(label labels.Selector, field fields.Selector, resourceVersion string) (watch.Interface, error) {
	return c.Fake.InvokesWatch(NewWatchAction("podsecuritypolicies", c.Namespace, label, field, resourceVersion))
}
