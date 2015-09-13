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

package unversioned

import (
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/testapi"
	"k8s.io/kubernetes/pkg/apis/experimental"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"

	"net/url"
)

func TestPodSecurityPolicyCreate(t *testing.T) {
	ns := api.NamespaceNone
	scc := &experimental.PodSecurityPolicy{
		ObjectMeta: api.ObjectMeta{
			Name: "abc",
		},
	}

	c := &testClient{
		Request: testRequest{
			Method: "POST",
			Path:   testapi.Experimental.ResourcePath(getPSPResourcename(), ns, ""),
			Query:  buildQueryValues(nil),
			Body:   scc,
		},
		Response: Response{StatusCode: 200, Body: scc},
	}

	response, err := c.Setup(t).PodSecurityPolicies().Create(scc)
	c.Validate(t, response, err)
}

func TestPodSecurityPolicyGet(t *testing.T) {
	ns := api.NamespaceNone
	scc := &experimental.PodSecurityPolicy{
		ObjectMeta: api.ObjectMeta{
			Name: "abc",
		},
	}
	c := &testClient{
		Request: testRequest{
			Method: "GET",
			Path:   testapi.Experimental.ResourcePath(getPSPResourcename(), ns, "abc"),
			Query:  buildQueryValues(nil),
			Body:   nil,
		},
		Response: Response{StatusCode: 200, Body: scc},
	}

	response, err := c.Setup(t).PodSecurityPolicies().Get("abc")
	c.Validate(t, response, err)
}

func TestPodSecurityPolicyList(t *testing.T) {
	ns := api.NamespaceNone
	sccList := &experimental.PodSecurityPolicyList{
		Items: []experimental.PodSecurityPolicy{
			{
				ObjectMeta: api.ObjectMeta{
					Name: "abc",
				},
			},
		},
	}
	c := &testClient{
		Request: testRequest{
			Method: "GET",
			Path:   testapi.Experimental.ResourcePath(getPSPResourcename(), ns, ""),
			Query:  buildQueryValues(nil),
			Body:   nil,
		},
		Response: Response{StatusCode: 200, Body: sccList},
	}
	response, err := c.Setup(t).PodSecurityPolicies().List(labels.Everything(), fields.Everything())
	c.Validate(t, response, err)
}

func TestPodSecurityPolicyUpdate(t *testing.T) {
	ns := api.NamespaceNone
	scc := &experimental.PodSecurityPolicy{
		ObjectMeta: api.ObjectMeta{
			Name:            "abc",
			ResourceVersion: "1",
		},
	}
	c := &testClient{
		Request:  testRequest{Method: "PUT", Path: testapi.Experimental.ResourcePath(getPSPResourcename(), ns, "abc"), Query: buildQueryValues(nil)},
		Response: Response{StatusCode: 200, Body: scc},
	}
	response, err := c.Setup(t).PodSecurityPolicies().Update(scc)
	c.Validate(t, response, err)
}

func TestPodSecurityPolicyDelete(t *testing.T) {
	ns := api.NamespaceNone
	c := &testClient{
		Request:  testRequest{Method: "DELETE", Path: testapi.Experimental.ResourcePath(getPSPResourcename(), ns, "foo"), Query: buildQueryValues(nil)},
		Response: Response{StatusCode: 200},
	}
	err := c.Setup(t).PodSecurityPolicies().Delete("foo")
	c.Validate(t, nil, err)
}

func TestPodSecurityPolicyWatch(t *testing.T) {
	c := &testClient{
		Request: testRequest{
			Method: "GET",
			Path:   "/experimental/" + testapi.Experimental.Version() + "/watch/" + getPSPResourcename(),
			Query:  url.Values{"resourceVersion": []string{}}},
		Response: Response{StatusCode: 200},
	}
	_, err := c.Setup(t).PodSecurityPolicies().Watch(labels.Everything(), fields.Everything(), "")
	c.Validate(t, nil, err)
}

func getPSPResourcename() string {
	return "podsecuritypolicies"
}
