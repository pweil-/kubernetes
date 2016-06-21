/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package seccomp

import (
	"strings"
	"testing"

	"k8s.io/kubernetes/pkg/api"
)

func TestNewMustRunAs(t *testing.T) {
	tests := map[string]struct {
		allowedProfiles []string
	}{
		"empty":    {allowedProfiles: []string{}},
		"nil":      {allowedProfiles: nil},
		"wildcard": {allowedProfiles: []string{allowAnyProfile}},
		"values":   {allowedProfiles: []string{"foo", "bar", "*"}},
	}

	for k, v := range tests {
		_, err := NewMustRunAs(v.allowedProfiles)

		if err != nil {
			t.Errorf("%s failed with error %v", k, err)
		}
	}
}

func TestGenerate(t *testing.T) {
	foo := "foo"
	tests := map[string]struct {
		allowedProfiles []string
		expectedProfile *string
	}{
		"empty allowed profiles": {
			allowedProfiles: []string{},
			expectedProfile: nil,
		},
		"nil allowed profiles": {
			allowedProfiles: nil,
			expectedProfile: nil,
		},
		"allow wildcard only": {
			allowedProfiles: []string{allowAnyProfile},
			expectedProfile: nil,
		},
		"allow values": {
			allowedProfiles: []string{"foo", "bar"},
			expectedProfile: &foo,
		},
		"allow wildcard and values": {
			allowedProfiles: []string{"*", "foo", "bar"},
			expectedProfile: &foo,
		},
	}

	for k, v := range tests {
		strategy, err := NewMustRunAs(v.allowedProfiles)
		if err != nil {
			t.Errorf("%s failed to create strategy with error %v", k, err)
			continue
		}

		actualProfile, generationError := strategy.Generate(nil, nil)
		if generationError != nil {
			t.Errorf("%s received generation error %v", k, generationError)
			continue
		}

		if v.expectedProfile == nil && actualProfile != nil {
			t.Errorf("%s expected nil but received %s", k, actualProfile)
			continue
		}
		if v.expectedProfile != nil && actualProfile == nil {
			t.Errorf("%s expected %s but received nil", k, v.expectedProfile)
			continue
		}
		if v.expectedProfile != nil && actualProfile != nil && *v.expectedProfile != *actualProfile {
			t.Errorf("%s expected %s but received %s", k, *v.expectedProfile, actualProfile)
		}
	}
}

func TestValidate(t *testing.T) {
	newPod := func(podAnnotation string, containerAnnotation string) *api.Pod {
		pod := &api.Pod{
			ObjectMeta: api.ObjectMeta{
				Annotations: map[string]string{},
			},
		}
		if len(podAnnotation) > 0 {
			pod.ObjectMeta.Annotations[api.SeccompPodAnnotationKey] = podAnnotation
		}
		if len(containerAnnotation) > 0 {
			pod.Spec.Containers = []api.Container{
				api.Container{
					Name: "test",
				},
			}
			pod.ObjectMeta.Annotations[api.SeccompContainerAnnotationKeyPrefix+"test"] = containerAnnotation
		}
		return pod
	}

	tests := map[string]struct {
		allowedProfiles []string
		pod             *api.Pod
		expectedMsg     string
	}{
		"empty allowed profiles, no pod annotation, no container annotation": {
			allowedProfiles: nil,
			pod:             newPod("", ""),
			expectedMsg:     "",
		},
		"empty allowed profiles, pod annotation": {
			allowedProfiles: nil,
			pod:             newPod("foo", ""),
			expectedMsg:     "Unsupported value: \"foo\"",
		},
		"empty allowed profiles, container annotation": {
			allowedProfiles: nil,
			pod:             newPod("", "foo"),
			expectedMsg:     "Unsupported value: \"foo\"",
		},
		"good pod annotation": {
			allowedProfiles: []string{"foo"},
			pod:             newPod("foo", ""),
			expectedMsg:     "",
		},
		// TODO should this be required to be set on the pod too, right now that is how it is working
		// but if there are only container annotations it should pass
		"good container annotation": {
			allowedProfiles: []string{"foo"},
			pod:             newPod("foo", "foo"),
			expectedMsg:     "",
		},
		"wildcard allows pod annotation": {
			allowedProfiles: []string{"*"},
			pod:             newPod("foo", ""),
			expectedMsg:     "",
		},
		"wildcard allows container annotation": {
			allowedProfiles: []string{"*"},
			pod:             newPod("", "foo"),
			expectedMsg:     "",
		},
		"wildcard allows no annotations": {
			allowedProfiles: []string{"*"},
			pod:             newPod("", ""),
			expectedMsg:     "",
		},
	}

	for name, tc := range tests {
		strategy, err := NewMustRunAs(tc.allowedProfiles)
		if err != nil {
			t.Errorf("%s failed to create strategy with error %v", name, err)
			continue
		}

		errs := strategy.Validate(tc.pod, nil)

		//should've passed but didn't
		if len(tc.expectedMsg) == 0 && len(errs) > 0 {
			t.Errorf("%s expected no errors but received %v", name, errs)
		}
		//should've failed but didn't
		if len(tc.expectedMsg) != 0 && len(errs) == 0 {
			t.Errorf("%s expected error %s but received no errors", name, tc.expectedMsg)
		}
		//failed with additional messages
		if len(tc.expectedMsg) != 0 && len(errs) > 1 {
			t.Errorf("%s expected error %s but received multiple errors: %v", name, tc.expectedMsg, errs)
		}
		//check that we got the right message
		if len(tc.expectedMsg) != 0 && len(errs) == 1 {
			if !strings.Contains(errs[0].Error(), tc.expectedMsg) {
				t.Errorf("%s expected error to contain %s but it did not: %v", name, tc.expectedMsg, errs)
			}
		}
	}
}
