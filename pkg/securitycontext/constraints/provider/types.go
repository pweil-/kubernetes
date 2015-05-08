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

package provider

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util/fielderrors"
)

// SecurityContextConstraintsProvider is responsible for ensuring that every service account has a
// security constraints in place and that a pod's context adheres to the active constraints.
type SecurityContextConstraintsProvider interface {
	// CreateContextForPod creates a security context for the pod based on what was
	// requested and what the policy allows
	CreateContextForContainer(pod *api.Pod, container *api.Container) *api.SecurityContext
	// ValidateAgainstConstraints validates the pod against SecurityContextConstraints
	ValidateAgainstConstraints(pod *api.Pod) fielderrors.ValidationErrorList
}
