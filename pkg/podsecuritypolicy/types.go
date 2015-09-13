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

package podsecuritypolicy

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/fielderrors"
)

// PodSecurityPolicyProvider provides the implementation to generate a new security
// context based on policy or validate an existing security context against policy.
type PodSecurityPolicyProvider interface {
	// Create a SecurityContext based on the given policy.
	CreateSecurityContext(pod *api.Pod, container *api.Container) (*api.SecurityContext, error)
	// Ensure a container's SecurityContext is in compliance with the given policy.
	ValidateSecurityContext(pod *api.Pod, container *api.Container) fielderrors.ValidationErrorList
	// Get the name of the SCC that this provider was initialized with.
	GetSCCName() string
}
