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
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/validation/field"
)

const (
	allowAnyProfile = "*"
)

type mustRunAs struct {
	allowedProfiles []string
}

var _ SeccompStrategy = &mustRunAs{}

func NewMustRunAs(allowedProfiles []string) (SeccompStrategy, error) {
	return &mustRunAs{allowedProfiles}, nil
}

// Generate creates the profile based on policy rules.
func (s *mustRunAs) Generate(pod *api.Pod, container *api.Container) (*string, error) {
	if len(s.allowedProfiles) > 0 {
		// return the first non-wildcard profile
		for _, p := range s.allowedProfiles {
			if p != allowAnyProfile {
				return &p, nil
			}
		}
	}
	// if we reached this point then either there are no allowed profiles (empty slice)
	// or the only thing in the slice is the wildcard.  In either case just return nil.
	return nil, nil
}

// Validate ensures that the specified values fall within the range of the strategy.
func (s *mustRunAs) Validate(pod *api.Pod, container *api.Container) field.ErrorList {
	allErrs := field.ErrorList{}

	// TODO
	annotationsPath := field.NewPath("annotations")

	podProfile, hasPodProfile := pod.Annotations[api.SeccompPodAnnotationKey]
	// validate the pod level annotation if it is set or if it is unset and we
	// have a list of allowed profiles.  If it is not set and we are not validating
	// against specific names then allow it to retain the unset value.
	if hasPodProfile || (!hasPodProfile && len(s.allowedProfiles) > 0) {
		if !isProfileAllowed(podProfile, s.allowedProfiles) {
			allErrs = append(allErrs, field.NotSupported(annotationsPath.Child(api.SeccompPodAnnotationKey), podProfile, s.allowedProfiles))
		}
	}

	// see if there are any container level annotations and validate them as well
	for _, c := range pod.Spec.Containers {
		containerProfileKey := api.SeccompContainerAnnotationKeyPrefix + c.Name
		containerProfile, hasContainerProfile := pod.Annotations[containerProfileKey]

		// same logic applies to containers as pods, see above.
		if hasContainerProfile || (!hasContainerProfile && len(s.allowedProfiles) > 0) {
			if !isProfileAllowed(containerProfile, s.allowedProfiles) {
				allErrs = append(allErrs, field.NotSupported(annotationsPath.Child(containerProfileKey), containerProfile, s.allowedProfiles))
			}
		}
	}
	return allErrs
}

func isProfileAllowed(profile string, allowedProfiles []string) bool {
	for _, p := range allowedProfiles {
		if profile == p || p == allowAnyProfile{
			return true
		}
	}
	return false
}
