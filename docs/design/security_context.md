# Security Contexts
## Abstract
A security context is a set of constraints that are applied to a container in order to achieve the following goals (from [security design](security.md)):

1.  Ensure a clear isolation between container and the underlying host it runs on
2.  Limit the ability of the container to negatively impact the infrastructure or other containers

## Background

The problem of securing containers in Kubernetes has come up [before](https://github.com/GoogleCloudPlatform/kubernetes/issues/398) and the potential problems with container security are [well known](http://opensource.com/business/14/7/docker-security-selinux). Although it is not possible to completely isolate Docker containers from their hosts, new features like [user namespaces](https://github.com/docker/libcontainer/pull/304) make it possible to greatly reduce the attack surface.

## Motivation

### Container isolation

In order to improve container isolation from host and other containers running on the host, containers should only be 
granted the access they need to perform their work. To this end it should be possible to take advantage of Docker 
features such as the ability to [add or remove capabilities](https://docs.docker.com/reference/run/#runtime-privilege-linux-capabilities-and-lxc-configuration) and [assign MCS labels](https://docs.docker.com/reference/run/#security-configuration) 
to the container process.

Support for user namespaces has recently been [merged](https://github.com/docker/libcontainer/pull/304) into Docker's libcontainer project and should soon surface in Docker itself. It will make it possible to assign a range of unprivileged uids and gids from the host to each container, improving the isolation between host and container and between containers.

### External integration with shared storage
In order to support external integration with shared storage, processes running in a Kubernetes cluster 
should be able to be uniquely identified by their Unix UID, such that a chain of  ownership can be established. 
Processes in pods will need to have consistent UID/GID/SELinux category labels in order to access shared disks.

## Constraints and Assumptions
* It is out of the scope of this document to prescribe a specific set 
  of constraints to isolate containers from their host. Different use cases need different
  settings.
* The concept of a security context should not be tied to a particular security mechanism or platform 
  (ie. SELinux, AppArmor)
* Applying a different security context to a scope (namespace or pod) requires a solution such as the one proposed for
  [service accounts](./service_accounts.md).

## Use Cases

In order of increasing complexity, following are example use cases that would 
be addressed with security contexts:

1.  Kubernetes is used to run a single cloud application. In order to protect
    nodes from containers:
    * All containers run as a single non-root user
    * Privileged containers are disabled
    * All containers run with a particular MCS label 
    * Kernel capabilities like CHOWN and MKNOD are removed from containers
    
2.  Just like case #1, except that I have more than one application running on
    the Kubernetes cluster.
    * Each application is run in its own namespace to avoid name collisions
    * For each application a different uid and MCS label is used
    
3.  Kubernetes is used as the base for a PAAS with 
    multiple projects, each project represented by a namespace. 
    * Each namespace is associated with a range of uids/gids on the node that
      are mapped to uids/gids on containers using linux user namespaces. 
    * Certain pods in each namespace have special privileges to perform system
      actions such as talking back to the server for deployment, run docker
      builds, etc.
    * External NFS storage is assigned to each namespace and permissions set
      using the range of uids/gids assigned to that namespace. 

## Proposed Design

### Overview

#### Components
1.  **security context constraints** - defines the policy under which a security context can make
requests.  Has a 1:1 relationship with a service account.  Also exists at a cluster level to provide
a default policy for the entire cluster.  The security context constraints must be extensible
to allow new implementations of context generating strategies to support future use cases like
running with a UID selected from a range or allowing multiple sets of SELinux options.
2.  **security context** - the run time parameters that a container will be configured with before
being created and run.  The security context is attached to the container and is used by the Kubelet
to mutate container API calls (Docker, Rkt, etc) in order to apply the security context.
3.  **security context constraints provider** - provides utility for creating security contexts
based on the security context constraints and for validating that an existing security context
complies with the constraints.
4.  **security context provider** - provides utility to the Kubelet for modifying the
container API calls

### Security Context Constraints

```go
// SecurityContextConstraints governs the ability to make requests that affect the SecurityContext that will
// be applied to a container.
type SecurityContextConstraints struct {
	// AllowPrivilegedContainer determines if a container can request to be run as privileged.
	AllowPrivilegedContainer bool
	// HostNetworkSources is a list of pod sources that are allowed to request to run in the host's
	// network namespace.
	HostNetworkSources []string
	// AllowedCapabilities is a list of capabilities that can be requested to add to the container.
	AllowedCapabilities []CapabilityType
	// SELinuxContext is the strategy that will dictate what labels will be set in the SecurityContext.
	SELinuxContext SELinuxContextStrategy
	// AllowHostDirVolumePlugin determines if the policy allow containers to use the HostDir volume plugin
	AllowHostDirVolumePlugin bool
	// RunAsUser  is the strategy that will dictate what RunAsUser is used in the SecurityContext.
	RunAsUser RunAsUserStrategy
}

// SELinuxContextStrategy provides configuration options for all SELinuxContextStrategy that can be used
// in a SecurityContextConstraints.
type SELinuxContextStrategy struct {
	// StrategyType is the SELinuxContextStrategyType being configured.
	StrategyType SELinuxContextStrategyType

	// SELinuxOptions are the specific SELinux labels to apply.  Required for SELinuxStrategyMustRunAs.
	SELinuxOptions *SELinuxOptions
}

// RunAsUserStrategy provides configuration options for all RunAsUserStrategy that can be used
// in a SecurityContextConstraints.
type RunAsUserStrategy struct {
	// StrategyType is the RunAsUserStrategyType being configured.
	StrategyType RunAsUserStrategyType

	// UID is a specific uid that will run pid 0 in the container.  Required for type RunAsUserStrategyMustRunAs.
	UID *int64
}

// SELinuxContextStrategyType defines different strategies that can be used when determining
// the SELinux labels to be applied to the container.
type SELinuxContextStrategyType string

// RunAsUserStrategyType defines different strategies that can be used when determining
// the uid that pid 1 will run as in the container.
type RunAsUserStrategyType string

const (
	// container must have SELinux labels of X applied.
	SELinuxStrategyMustRunAs SELinuxContextStrategyType = "MustRunAs"
	// container may make requests for any SELinux context labels.
	SELinuxStrategyRunAsAny SELinuxContextStrategyType = "RunAsAny"
	// containers must run with the default settings, their requests are ignored
	SELinuxStrategyRunAsDefault SELinuxContextStrategyType = "RunAsDefault"

	// container must run as a particular uid.
	RunAsUserStrategyMustRunAs RunAsUserStrategyType = "MustRunAs"
	// container must run as a non-root uid
	RunAsUserStrategyMustRunAsNonRoot RunAsUserStrategyType = "MustRunAsNonRoot"
	// container may make requests for any uid.
	RunAsUserStrategyRunAsAny RunAsUserStrategyType = "RunAsAny"
	// containers must run with the default settings, their requests are ignored
	RunAsUserStrategyRunAsDefault RunAsUserStrategyType = "RunAsDefault"
)
```

### Security Context Constraints Provider

```go
// SecurityContextConstraintsProvider is responsible for ensuring that every service account has a
// security constraints in place and that a pod's context adheres to the active constraints.
type SecurityContextConstraintsProvider interface {
	// CreateContextForPod creates a security context for the pod based on what was
	// requested and what the policy allows
	CreateContextForContainer(pod *api.Pod, container *api.Container) *api.SecurityContext
	// ValidateAgainstConstraints validates the pod against SecurityContextConstraints
	ValidateAgainstConstraints(pod *api.Pod) fielderrors.ValidationErrorList
}
```

### Security Context Provider

The Kubelet will have an interface that points to a `SecurityContextProvider`. The `SecurityContextProvider` is invoked before creating and running a given container:

```go
type SecurityContextProvider interface {
	// ModifyContainerConfig is called before the Docker createContainer call.
	// The security context provider can make changes to the Config with which
	// the container is created.
	// An error is returned if it's not possible to secure the container as 
	// requested with a security context. 
	ModifyContainerConfig(pod *api.Pod, container *api.Container, config *docker.Config) error
	
	// ModifyHostConfig is called before the Docker runContainer call.
	// The security context provider can make changes to the HostConfig, affecting
	// security options, whether the container is privileged, volume binds, etc.
	// An error is returned if it's not possible to secure the container as requested 
	// with a security context. 
	ModifyHostConfig(pod *api.Pod, container *api.Container, hostConfig *docker.HostConfig)
}
```

If the value of the SecurityContextProvider field on the Kubelet is nil, the kubelet will create and run the container as it does today.   

### Security Context

```go

// SecurityContext holds security configuration that will be applied to a container.  SecurityContext
// contains duplication of some existing fields from the Container resource.  These duplicate fields
// will be populated based on the Container configuration if they are not set.  Defining them on
// both the Container AND the SecurityContext will result in an error.
type SecurityContext struct {
	// Capabilities are the capabilities to add/drop when running the container
	Capabilities *Capabilities

	// Run the container in privileged mode
	Privileged *bool

	// SELinuxOptions are the labels to be applied to the container
	// and volumes
	SELinuxOptions *SELinuxOptions

	// RunAsUser is the UID to run the entrypoint of the container process.
	RunAsUser *int64
}

// SELinuxOptions are the labels to be applied to the container.
type SELinuxOptions struct {
	// SELinux user label
	User string

	// SELinux role label
	Role string

	// SELinux type label
	Type string

	// SELinux level label.
	Level string
}
```

### Security Context Constraints Lifecycle

A security context constraints configuration exists at both the cluster level and the service account
level.

For service accounts, the lifecycle of the security context constraints follows that of
the service account.  If resources need to be allocated when creating a security
context constraints configuration (for example, assign a range of host uids/gids), a pattern such as [finalizers](https://github.com/GoogleCloudPlatform/kubernetes/issues/3585)
can be used before declaring the constraints / service account / namespace ready for use.

The constraints that live on the cluster level are tied to the lifecycle of the cluster.  It is
expected that not every service account will need to define security context constraints and will
use the default cluster constraints in the absence of an overriding definition.

### Escalating Privileges by an Administrator

It if feasible that an administrator may wish to create a resource in a namespace that runs with
escalated privileges.  This is similar to saying `sudo make me a pod`.  The security context provider
may be made aware of identity and allow a creator with certain roles to pass validation despite
non-conformance to the service account or cluster policy which is expected to be more restrictive. A
good example of this may be a build controller creating privileged containers.  As a system
component the build controller may identify itself with a role that allows it to bypass the more
restrictive set of constraints.

This also allows the system to guard commands being executed in the non-conforming container.  For
instance, an `exec` command can first check the security context of the pod against the service
account or cluster policy.  If it does not validate then it can block users from executing the
command.  Since the validation will be identity aware administrators would still be able to
run the commands that are restricted to normal users.

With this approach there is not a way for users to point to a more privileged service account
and inadvertently get access to create pods with escalated privileges.  However, this makes the
assumption that editing a service account's security context constraints is restricted to
cluster administrators.
