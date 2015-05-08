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

package user

import "github.com/GoogleCloudPlatform/kubernetes/pkg/api"

// mustRunAs is the strategy implementation and will always return the configured uid.
type mustRunAsNonRoot struct{}

// NewMustRunAsNonRoot creates a new must run as non-root strategy.
func NewMustRunAsNonRoot() RunAsUserSecurityContextConstraintsStrategy {
	return &mustRunAsNonRoot{}
}

// Generate creates the SELinuxOptions based on policy rules.
func (s *mustRunAsNonRoot) Generate(podSecurityContext *api.SecurityContext) *int64 {
	if podSecurityContext != nil && podSecurityContext.RunAsUser != nil && *podSecurityContext.RunAsUser > 0 {
		return podSecurityContext.RunAsUser
	}
	// TODO - this should be able to pick a valid non-root user
	var uid int64 = 1
	return &uid
}
