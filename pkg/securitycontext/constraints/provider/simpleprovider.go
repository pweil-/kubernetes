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
	"fmt"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	apierrors "github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/securitycontext/constraints/selinux"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/securitycontext/constraints/user"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util/fielderrors"
)

type simpleProvider struct {
	client          client.Interface
	constraints     *api.SecurityContextConstraints
	userStrategy    user.RunAsUserSecurityContextConstraintsStrategy
	seLinuxStrategy selinux.SELinuxSecurityContextConstraintsStrategy
}

// NewSimpleSecurityProvider provides a new security context provider.  It should be initialized
// with the default security context constraints which will be used for new constraint requests.
// In functions that deal with specific pods the ServiceAccount that the pod references will
// be queried for an overriding set of constraints, if it doesnt exist then the default will
// be used
func NewSimpleSecurityProvider(client client.Interface, constraints *api.SecurityContextConstraints) SecurityContextConstraintsProvider {
	p := &simpleProvider{
		constraints: constraints,
	}
	switch constraints.RunAsUser.StrategyType {
	case api.RunAsUserStrategyMustRunAs:
		p.userStrategy = user.NewMustRunAs(constraints.RunAsUser.UID)
	case api.RunAsUserStrategyRunAsAny:
		p.userStrategy = user.NewRunAsAny()
	case api.RunAsUserStrategyMustRunAsNonRoot:
		p.userStrategy = user.NewMustRunAsNonRoot()
	case api.RunAsUserStrategyRunAsDefault:
		p.userStrategy = user.NewRunAsDefault()
	default:
		p.userStrategy = user.NewRunAsAny()
	}

	switch constraints.SELinuxContext.StrategyType {
	case api.SELinuxStrategyMustRunAs:
		p.seLinuxStrategy = selinux.NewMustRunAs(constraints.SELinuxContext.SELinuxOptions)
	case api.SELinuxStrategyRunAsAny:
		p.seLinuxStrategy = selinux.NewRunAsAny()
	case api.SELinuxStrategyRunAsDefault:
		p.seLinuxStrategy = selinux.NewRunAsDefault()
	default:
		p.seLinuxStrategy = selinux.NewRunAsAny()
	}

	return p
}

// CreateContextForPod creates or modifies the existing pod SecurityContext with
// policy values.
func (p *simpleProvider) CreateContextForContainer(pod *api.Pod, container *api.Container) *api.SecurityContext {
	constraints := p.getConstraintsForPod(pod)
	ctx := &api.SecurityContext{}

	if constraints.AllowPrivilegedContainer {
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil {
			ctx.Privileged = container.SecurityContext.Privileged
		} else {
			priv := false
			ctx.Privileged = &priv
		}
	} else {
		priv := false
		ctx.Privileged = &priv
	}

	uid := p.userStrategy.Generate(container.SecurityContext)
	ctx.RunAsUser = uid

	seLinuxOptions := p.seLinuxStrategy.Generate(container.SecurityContext)
	ctx.SELinuxOptions = seLinuxOptions

	if container.SecurityContext != nil && container.SecurityContext.Capabilities != nil {
		caps := []api.CapabilityType{}

		for _, c := range container.SecurityContext.Capabilities.Add {
			for _, allowed := range constraints.AllowedCapabilities {
				if c == allowed {
					caps = append(caps, c)
					break
				}
			}
		}

		ctx.Capabilities = &api.Capabilities{
			Add:  caps,
			Drop: container.SecurityContext.Capabilities.Drop,
		}
	}

	//TODO host network sources
	//TODO AllowHostDirVolumePlugin
	return ctx
}

// ValidateAgainstConstraints validates the pod against SecurityContextConstraints
//
// TODO: make this identity aware.  In order to allow cluster admins to create pods with elevated
// privileges but not allow normal users we can check the identity of the creator.  If they
// have a role that allows elevated privileges then we can force the SecurityContext on the
// pod to be considered valid.  This requires them to specifically request the privs in a
// pod.container.SecurityContext and would be the equivalent of sudo make me a pod and checking
// the sudoers file.
//
// To be effective we would need to control who could change the SecurityConstraints on given
// ServiceAccount.  This would allow the pod to point to a ServiceAccount in the namespace but
// still run with elevated privileges.  If a user with a non-admin role tries the run a command
// in the pod, the pod SC can be validated against the service account's SCC which will not match
// and the command can be denied.
//
// This could cause some confusion, though, as to how the pod became running with elevated privileges.
// Perhaps an annotation can be added.
func (p *simpleProvider) ValidateAgainstConstraints(pod *api.Pod) fielderrors.ValidationErrorList {
	errs := fielderrors.ValidationErrorList{}
	for _, v := range pod.Spec.Containers {
		if v.SecurityContext != nil {
			if v.SecurityContext.SELinuxOptions != nil {
				errs = append(errs, apierrors.NewForbidden("Pod", pod.Name, fmt.Errorf("SecurityContext.SELinuxOptions is forbidden")))
			}
			if v.SecurityContext.RunAsUser != nil {
				errs = append(errs, apierrors.NewForbidden("Pod", pod.Name, fmt.Errorf("SecurityContext.RunAsUser is forbidden")))
			}
		}
	}
	return errs
}

// getConstraintsForPod gives either the constraints for the service account that the pod
// references or if it is nil then the cluster defaults
func (p *simpleProvider) getConstraintsForPod(pod *api.Pod) *api.SecurityContextConstraints {
	serviceAccount, err := p.client.ServiceAccounts(pod.Namespace).Get(pod.Name)
	if err != nil {
		return p.constraints
	}
	if serviceAccount.SecurityContextConstraints != nil {
		return serviceAccount.SecurityContextConstraints
	}
	return p.constraints
}
