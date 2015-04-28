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

package scdeny

import (
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/admission"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
)

func TestAdmission(t *testing.T) {
	handler := NewSecurityContextDeny(nil)

	admitPod := api.Pod{}

	err := handler.Admit(admission.NewAttributesRecord(&admitPod, "Pod", "foo", string(api.ResourcePods), "ignored"))
	if err != nil {
		t.Errorf("Unexpected error returned from admission handler")
	}

	denyPod := api.Pod{
		Spec: api.PodSpec{
			Containers: []api.Container{
				{SecurityContext: &api.SecurityContext{}},
			},
		},
	}
	err = handler.Admit(admission.NewAttributesRecord(&denyPod, "Pod", "foo", string(api.ResourcePods), "ignored"))
	if err == nil {
		t.Errorf("Expected error returned from admission handler")
	}
}
