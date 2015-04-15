/*
Copyright 2014 Google Inc. All rights reserved.

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

package securitycontext

import (
	"fmt"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/golang/glog"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
)

type DefaultSecurityContextProvider struct {
	*api.SecurityConstraints
}

// NewDefaultSecurityContextProvider creates a new security context provider with the default constraints
func NewDefaultSecurityContextProvider() SecurityContextProvider {
	return &DefaultSecurityContextProvider{
		SecurityConstraints: &api.SecurityConstraints{
			EnforcementPolicy: api.SecurityConstraintPolicyReject,
			AllowPrivileged:   true,
			AllowCapabilities: true,
			SELinux: &api.SELinuxSecurityConstraints{
				AllowUserLabel:  true,
				AllowRoleLabel:  true,
				AllowTypeLabel:  true,
				AllowLevelLabel: true,
				AllowDisable:    true,
			},
			DefaultSecurityContext: &api.SecurityContext{
				Privileged: false,
			},
		},
	}
}

// ApplySecurityContext ensures that each container in the pod has a security context set and that it
// complies with the policy defined by the SecurityConstraints of the provider
func (p DefaultSecurityContextProvider) ApplySecurityContext(pod *api.Pod) {
	for idx := range pod.Spec.Containers {
		c := &pod.Spec.Containers[idx]
		if c.SecurityContext == nil {
			glog.V(4).Infof("Setting default security context for pod: %s, container: %s", pod.Name, c.Name)
			c.SecurityContext = p.DefaultSecurityContext
			continue
		}
		glog.V(4).Infof("Applying security constraints to %s", c.Name)
		p.applySecurityContextToContainer(c)
	}
}

// applySecurityContextToContainer applies each section of the security context to the container.  As more options
// become available they should be added here with corresponding application methods.
func (p DefaultSecurityContextProvider) applySecurityContextToContainer(c *api.Container) {
	p.applyPrivileged(c)
	p.applyCapRequests(c)
	p.applySELinux(c)
}

// applySELinux will:
// 1.  if there are not selinux options on the security constraints: take no action
// 2.  if there are selinux options on the security constraints AND there are no options defined on the container
//			AND there is a default security context then use all the default settings
// 3.  if there are selinux options on the security constraints AND options on the container then check each
//			setting individually.  If the individual setting is not allowed then remove it or set it to the default if one exists
func (p DefaultSecurityContextProvider) applySELinux(container *api.Container) {
	// no security context settings for SELinux
	if p.SecurityConstraints.SELinux == nil {
		return
	}

	constraints := p.SecurityConstraints
	hasDefault := constraints.DefaultSecurityContext != nil
	hasDefaultSELinux := hasDefault && constraints.DefaultSecurityContext.SELinuxOptions != nil

	// if the container has not defined SELinux options then apply the default if it exists
	if container.SecurityContext.SELinuxOptions == nil {
		if hasDefault {
			glog.V(4).Infof("Setting default SELinux options for container %s", container.Name)
			container.SecurityContext.SELinuxOptions = constraints.DefaultSecurityContext.SELinuxOptions
		}
		return
	}

	// check individual settings of the container's request
	if !constraints.SELinux.AllowDisable && container.SecurityContext.SELinuxOptions.Disabled {
		glog.V(4).Infof("Resetting SELinuxOptions.Disabled for %s", container.Name)
		container.SecurityContext.SELinuxOptions.Disabled = false
	}

	if !constraints.SELinux.AllowLevelLabel {
		glog.V(4).Infof("Resetting SELinuxOptions.Level for %s", container.Name)
		level := ""
		if hasDefault && hasDefaultSELinux {
			level = constraints.DefaultSecurityContext.SELinuxOptions.Level
		}
		container.SecurityContext.SELinuxOptions.Level = level
	}

	if !constraints.SELinux.AllowRoleLabel {
		glog.V(4).Infof("Resetting SELinuxOptions.Role for %s", container.Name)
		role := ""
		if hasDefault && hasDefaultSELinux {
			role = constraints.DefaultSecurityContext.SELinuxOptions.Role
		}
		container.SecurityContext.SELinuxOptions.Role = role
	}

	if !constraints.SELinux.AllowTypeLabel {
		glog.V(4).Infof("Resetting SELinuxOptions.Type for %s", container.Name)
		typeLabel := ""
		if hasDefault && hasDefaultSELinux {
			typeLabel = constraints.DefaultSecurityContext.SELinuxOptions.Type
		}
		container.SecurityContext.SELinuxOptions.Type = typeLabel
	}

	if !constraints.SELinux.AllowUserLabel {
		glog.V(4).Infof("Resetting SELinuxOptions.User for %s", container.Name)
		user := ""
		if hasDefault && hasDefaultSELinux {
			user = constraints.DefaultSecurityContext.SELinuxOptions.User
		}
		container.SecurityContext.SELinuxOptions.User = user
	}
}

// applyCapRequests will take the following steps:
// 1.  if the security context does not allow capability requests and the container has capability requests defined
//			it will set them to the default settings.  If no default settings exist it will remove all requests
// 2.  if the security context allows capability requests it will remove any add/drop requests that are not in
//			the allowed set of add/drops.
//
// NOTE: if cap requests are allowed and requests are defined you will not get the defaults in addition to the
//			requested add/drop list.  This is an override, not a additive operation.
func (p DefaultSecurityContextProvider) applyCapRequests(container *api.Container) {
	context := p.SecurityConstraints
	if !context.AllowCapabilities {
		//if we don't allow cap requests and the container is requesting them then either use the default
		//or remove them completely
		if context.DefaultSecurityContext != nil && context.DefaultSecurityContext.Capabilities != nil {
			glog.V(4).Infof("Resetting cap add/drop for %s to default settings", container.Name)
			container.SecurityContext.Capabilities = context.DefaultSecurityContext.Capabilities
		} else {
			if container.SecurityContext.Capabilities != nil {
				glog.V(4).Infof("Removing cap add/drop for %s", container.Name)
				container.SecurityContext.Capabilities = &api.Capabilities{}
			}
		}

	} else {
		//otherwise check each request to see if it is allowed.  If we haven't defined any cap restrictions
		//then there is nothing to do
		if context.Capabilities != nil && container.SecurityContext.Capabilities != nil {
			container.SecurityContext.Capabilities.Add = p.filterCapabilities(container.SecurityContext.Capabilities.Add, context.Capabilities.Add)
			container.SecurityContext.Capabilities.Drop = p.filterCapabilities(container.SecurityContext.Capabilities.Drop, context.Capabilities.Drop)
		}
	}
}

// applyPrivileged will ensure that if a container is not allowed to make privileged container requests
// the setting will be reset to false
func (p DefaultSecurityContextProvider) applyPrivileged(container *api.Container) {
	if !p.SecurityConstraints.AllowPrivileged && container.SecurityContext.Privileged {
		glog.V(4).Infof("Resetting privileged for %s", container.Name)
		container.SecurityContext.Privileged = false
	}
}

// filterCapabilities filters the capability requests based on what is allowed.
func (p DefaultSecurityContextProvider) filterCapabilities(capRequests, allowedCaps []api.CapabilityType) []api.CapabilityType {
	filteredCaps := make([]api.CapabilityType, 0)

outer:
	for _, cap := range capRequests {
		for _, allowed := range allowedCaps {
			if cap == allowed {
				filteredCaps = append(filteredCaps, cap)
				continue outer
			}
		}
	}
	return filteredCaps
}

func (p DefaultSecurityContextProvider) ValidateSecurityContext(pod *api.Pod) error {
	//TODO
	return nil
}

func (p DefaultSecurityContextProvider) ModifyContainerConfig(pod *api.Pod, container *api.Container, config *docker.Config) error {
	//the only time the SC should be nil by the time we get here is if this is being applied to an infra container
	//or no security contexts are defined
	if container.SecurityContext == nil {
		return nil
	}

	return nil
}

func (p DefaultSecurityContextProvider) ModifyHostConfig(pod *api.Pod, container *api.Container, hostConfig *docker.HostConfig) {
	//the only time the SC should be nil by the time we get here is if this is being applied to an infra container
	//or no security contexts are defined
	if container.SecurityContext == nil {
		return
	}

	hostConfig.Privileged = container.SecurityContext.Privileged

	if container.SecurityContext.Capabilities != nil {
		add, drop := makeCapabilites(container.SecurityContext.Capabilities.Add, container.SecurityContext.Capabilities.Drop)
		hostConfig.CapAdd = add
		hostConfig.CapDrop = drop
	}

	if container.SecurityContext.SELinuxOptions != nil {
		if container.SecurityContext.SELinuxOptions.Disabled {
			hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, dockerLabelDisable)
		} else {
			hostConfig.SecurityOpt = modifySecurityOption(hostConfig.SecurityOpt, dockerLabelUser, container.SecurityContext.SELinuxOptions.User)
			hostConfig.SecurityOpt = modifySecurityOption(hostConfig.SecurityOpt, dockerLabelRole, container.SecurityContext.SELinuxOptions.Role)
			hostConfig.SecurityOpt = modifySecurityOption(hostConfig.SecurityOpt, dockerLabelType, container.SecurityContext.SELinuxOptions.Type)
			hostConfig.SecurityOpt = modifySecurityOption(hostConfig.SecurityOpt, dockerLabelLevel, container.SecurityContext.SELinuxOptions.Level)
		}
	}
}

func modifySecurityOption(config []string, name, value string) []string {
	if len(name) > 0 && len(value) > 0 {
		config = append(config, fmt.Sprintf("%s:%s", name, value))
	}

	return config
}

//TODO copied from manager.go, manager.go can be updated since it will no longer need to apply caps itself
func makeCapabilites(capAdd []api.CapabilityType, capDrop []api.CapabilityType) ([]string, []string) {
	var (
		addCaps  []string
		dropCaps []string
	)
	for _, cap := range capAdd {
		addCaps = append(addCaps, string(cap))
	}
	for _, cap := range capDrop {
		dropCaps = append(dropCaps, string(cap))
	}
	return addCaps, dropCaps
}
