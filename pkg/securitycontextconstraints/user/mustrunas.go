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

import (
	"fmt"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util/fielderrors"
)

// mustRunAs implements the RunAsUserSecurityContextConstraintsStrategy interface
type mustRunAs struct {
	opts   *api.RunAsUserStrategyOptions
	client client.Interface
}

// NewMustRunAs provides a strategy that requires the container to run as a specific UID.
func NewMustRunAs(options *api.RunAsUserStrategyOptions, client client.Interface) (RunAsUserSecurityContextConstraintsStrategy, error) {
	if len(options.AllocatedIDAnnotation) == 0 && options.UID == nil {
		return nil, fmt.Errorf("MustRunAs requires a UID or an annotation")
	}
	return &mustRunAs{
		opts:   options,
		client: client,
	}, nil
}

// Generate creates the uid based on policy rules.  MustRunAs requires that can either
// retrieve a pre-allocated UID from the service account (if specified) or the namespace.  If
// no annotation is specified on the strategy then it will return the configured UID.
func (s *mustRunAs) Generate(pod *api.Pod, container *api.Container) (*int64, error) {
	if len(s.opts.AllocatedIDAnnotation) > 0 {
		return GetAllocatedID(s.client, pod, s.opts.AllocatedIDAnnotation)
	}
	return s.opts.UID, nil
}

// Validate ensures that the specified values fall within the range of the strategy.
func (s *mustRunAs) Validate(pod *api.Pod, container *api.Container) fielderrors.ValidationErrorList {
	allErrs := fielderrors.ValidationErrorList{}

	if container.SecurityContext == nil {
		allErrs = append(allErrs, fmt.Errorf("Unable to validate nil security context for container %s", container.Name))
		return allErrs
	}
	if container.SecurityContext.RunAsUser == nil {
		allErrs = append(allErrs, fmt.Errorf("Unable to validate nil RunAsUser for container %s", container.Name))
		return allErrs
	}

	var uid *int64 = nil

	if len(s.opts.AllocatedIDAnnotation) > 0 {
		u, err := GetAllocatedID(s.client, pod, s.opts.AllocatedIDAnnotation)
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("Annotation was specified but an error was encountered retrieving the UID %v", err))
		}
		uid = u
	} else {
		uid = s.opts.UID
	}

	// couldn't get a UID from the annotation and don't have a UID set on the strategy
	if uid == nil {
		allErrs = append(allErrs, fmt.Errorf("UID for MustRunAs strategy is nil"))
		return allErrs
	}
	if *uid != *container.SecurityContext.RunAsUser {
		allErrs = append(allErrs, fmt.Errorf("UID on container %s does not match required UID.  Found %d, wanted %d", container.Name, *uid, *container.SecurityContext.RunAsUser))
	}

	return allErrs
}
