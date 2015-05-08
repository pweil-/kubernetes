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

package scdeny

import (
	"fmt"
	"io"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/admission"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	apierrors "github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/securitycontext/constraints/provider"
	"encoding/json"
)

func init() {
	admission.RegisterPlugin("SecurityContextDeny", func(client client.Interface, config io.Reader) (admission.Interface, error) {
		cfg, err := readConfig(config)
		if err != nil {
			return nil, err
		}
		return NewSecurityContextDeny(client, cfg.namespace, cfg.name)
	})
}

// plugin contains the client used by the SecurityContextDeny admission controller
type plugin struct {
	client      client.Interface
	saNamespace string
	saName      string
}

// NewSecurityContextDeny creates a new instance of the SecurityContextDeny admission controller
func NewSecurityContextDeny(client client.Interface, defaultServiceAccountNamespace, defaultServiceAccountName string) (admission.Interface, error) {
	return &plugin{
		client:      client,
		saNamespace: defaultServiceAccountNamespace,
		saName: defaultServiceAccountName,
	}, nil
}

// Admit will deny any SecurityContext that defines options that were not previously available in the api.Container
// struct (Capabilities and Privileged)
func (p *plugin) Admit(a admission.Attributes) (err error) {
	if a.GetOperation() == "DELETE" {
		return nil
	}
	if a.GetResource() != string(api.ResourcePods) {
		return nil
	}

	pod, ok := a.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	defaultServiceAccount, err := p.client.ServiceAccounts(p.saNamespace).Get(p.saName)
	if err != nil {
		return err
	}
	clusterConstraints := defaultServiceAccount.SecurityContextConstraints
	sccProvider := provider.NewSimpleSecurityProvider(p.client, clusterConstraints)

	// if the pod is making any security context requests then we will validate it against the
	// policy.  Otherwise we'll add a security context to every container
	shouldValidate := false
	for _, c := range pod.Spec.Containers {
		if c.SecurityContext != nil {
			shouldValidate = true
		}
	}

	if shouldValidate {
		errs := sccProvider.ValidateAgainstConstraints(pod)
		if errs != nil {
			return fmt.Errorf("Pod failed to validate against the security context constraints: %#v", errs)
		}
	} else {
		for idx, c := range pod.Spec.Containers {
			pod.Spec.Containers[idx].SecurityContext = sccProvider.CreateContextForContainer(pod, &c)
		}
		errs := sccProvider.ValidateAgainstConstraints(pod)
		if errs != nil {
			return fmt.Errorf("Pod failed to validate against the security context constraints: %#v", errs)
		}
	}
	return nil
}

// TODO this is assuming that it is the only thing configured in the file which is W-R-O-N-G
type config struct {
	namespace, name string
}

func readConfig(r io.Reader) (config, error) {
	decoder := json.NewDecoder(r)
	cfg := config{}
	err := decoder.Decode(&cfg)
	return cfg, err
}
