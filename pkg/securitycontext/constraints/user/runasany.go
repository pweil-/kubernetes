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

// runAsAny is the strategy implementation and will always return whatever is already set on the
// pod's context or 0.
type runAsAny struct{}

// NewRunAsAny creates a new runAsAny strategy.
func NewRunAsAny() RunAsUserSecurityContextConstraintsStrategy {
	return &runAsAny{}
}

// Generate creates the uid based on policy rules.
func (s *runAsAny) Generate(podSecurityContext *api.SecurityContext) *int64 {
	if podSecurityContext != nil && podSecurityContext.RunAsUser != nil {
		return podSecurityContext.RunAsUser
	}
	return nil
}
