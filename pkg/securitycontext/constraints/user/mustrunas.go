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
type mustRunAs struct {
	// uid the uid that will always be returned.
	uid *int64
}

// NewMustRunAs creates a new must run as strategy.
func NewMustRunAs(uid *int64) RunAsUserSecurityContextConstraintsStrategy {
	return &mustRunAs{
		uid: uid,
	}
}

// Generate creates the SELinuxOptions based on policy rules.
func (s *mustRunAs) Generate(podSecurityContext *api.SecurityContext) *int64 {
	return s.uid
}
