<!-- BEGIN MUNGE: UNVERSIONED_WARNING -->

<!-- BEGIN STRIP_FOR_RELEASE -->

<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">

<h2>PLEASE NOTE: This document applies to the HEAD of the source tree</h2>

If you are using a released version of Kubernetes, you should
refer to the docs that go with that version.

<strong>
The latest 1.0.x release of this document can be found
[here](http://releases.k8s.io/release-1.0/docs/proposals/security-context-constraints.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

## Abstract

PodSecurityPolicy allows cluster administrators to control the creation and validation of
 for a pod.

## Motivation

Administration of a multi-tenant cluster requires the ability to provide varying sets of permissions
among the tenants, the infrastructure components, and end users of the system who may themselves be
administrators within their own isolated namespace.

Actors in a cluster may include infrastructure that is managed by administrators, infrastructure
that is exposed to end users (builds, deployments), the isolated end user namespaces in the cluster, and
the individual users inside those namespaces.  Infrastructure components that operate on behalf of a
user (builds, deployments) should be allowed to run at an elevated level of permissions without
granting the user themselves an elevated set of permissions.

## Goals

1.  Associate [service accounts](http://docs.k8s.io/design/service_accounts.md), groups, and users with
a set of constraints that dictate how a  is established for a pod.
1.  Provide the ability for users and infrastructure components to run pods with elevated privileges
on behalf of another user or within a namespace where privileges are more restrictive.
1.  Secure the ability to reference elevated permissions or to change the constraints under which
a user runs.

## Use Cases

Use case 1:
As an administrator, I can create a namespace for a person that can't create privileged containers
AND enforces that the UID of the containers is set to a certain value

Use case 2:
As a cluster operator, an infrastructure component should be able to create a pod with elevated
privileges in a namespace where regular users cannot create pods with these privileges or execute
commands in that pod.

Use case 3:
As a cluster administrator, I can allow a given namespace (or service account) to create privileged
pods or to run root pods

Use case 4:
As a cluster administrator, I can allow a project administrator to control the security contexts of
pods and service accounts within a project


## Requirements

1.  Provide a set of restrictions that controls how a security context is created as a new, cluster-scoped, object
called PodSecurityPolicy.
1.  User information in `user.Info` must be available to admission controllers. (Completed in
https://github.com/GoogleCloudPlatform/kubernetes/pull/8203)
1.  Some authorizers may restrict a user’s ability to reference a service account.  Systems requiring
the ability to secure service accounts on a user level must be able to add a policy that enables
referencing specific service accounts themselves.
1.  Admission control must validate the creation of Pods against the allowed set of constraints.

## Design

### Model

PodSecurityPolicy objects exists in the root scope, outside of a namespace.  The
PodSecurityPolicy will reference users and groups that are allowed
to operate under the constraints.  In order to support this, `ServiceAccounts` must be mapped
to a user name or group list by the authentication/authorization layers.  This allows the security
context to treat users, groups, and service accounts uniformly.

Below is a list of PodSecurityPolicies which will likely serve most use cases:

1.  A default policy object.  This object is permissioned to something covers all actors such
as a `system:authenticated` group and will likely be the most restrictive set of constraints.
1.  A default constraints object for service accounts.  This object can be identified as serving
a group identified by `system:service-accounts` which can be imposed by the service account authenticator / token generator.
1.  Cluster admin constraints identified by `system:cluster-admins` group - a set of constraints with elevated privileges that can be used
by an administrative user or group.
1.  Infrastructure components constraints which can be identified either by a specific service
account or by a group containing all service accounts.

```go
// PodSecurityPolicy governs the ability to make requests that affect the SecurityContext
// that will be applied to a container.
type PodSecurityPolicy struct {
	TypeMeta
	ObjectMeta

	// AllowPrivileged determines if a container can request to be run as privileged.
	AllowPrivileged bool
	// AllowCapabilities is a list of capabilities that can be requested to add to the container.
	AllowCapabilities []Capability
	// AllowHostPath determines if the policy allow containers to use the HostDir volume plugin
	AllowHostPath bool
	// AllowHostNetwork determines if the policy allows the use of HostNetwork in the pod spec.
	AllowHostNetwork bool
	// AllowHostPort determines if the policy allows host ports in the containers.
	AllowHostPort bool
	// SELinuxContext is the strategy that will dictate what labels will be set in the SecurityContext.
	SELinuxContext SELinuxContextStrategyOptions
	// RunAsUser is the strategy that will dictate what RunAsUser is used in the SecurityContext.
	RunAsUser RunAsUserStrategyOptions

	// The users who have permissions to use this pod security policy.
	Users []string
	// The groups that have permission to use this pod security policy.
	Groups []string
}

// SELinuxContextStrategyOptions defines the strategy type and any options used to create the strategy.
type SELinuxContextStrategyOptions struct {
	// Type is the strategy that will dictate what SELinux context is used in the SecurityContext.
	Type SELinuxContextStrategyType
	// seLinuxOptions required to run as; required for MustRunAs
	SELinuxOptions *SELinuxOptions
}


// RunAsUserStrategyOptions defines the strategy type and any options used to create the strategy.
type RunAsUserStrategyOptions struct {
	// Type is the strategy that will dictate what RunAsUser is used in the SecurityContext.
	Type RunAsUserStrategyType
	// UID is the user id that containers must run as.  Required for the MustRunAs strategy if not using
	// namespace/service account allocated uids.
	UID *int64
	// UIDRangeMin defines the min value for a strategy that allocates by range.
	UIDRangeMin *int64
	// UIDRangeMax defines the max value for a strategy that allocates by range.
	UIDRangeMax *int64
}

// SELinuxContextStrategyType denotes strategy types for generating SELinux options for a
// SecurityContext
type SELinuxContextStrategyType string

// RunAsUserStrategyType denotes strategy types for generating RunAsUser values for a
// SecurityContext
type RunAsUserStrategyType string

const (
	// container must have SELinux labels of X applied.
	SELinuxStrategyMustRunAs SELinuxContextStrategyType = "MustRunAs"
	// container may make requests for any SELinux context labels.
	SELinuxStrategyRunAsAny SELinuxContextStrategyType = "RunAsAny"

	// container must run as a particular uid.
	RunAsUserStrategyMustRunAs RunAsUserStrategyType = "MustRunAs"
	// container must run as a particular uid.
	RunAsUserStrategyMustRunAsRange RunAsUserStrategyType = "MustRunAsRange"
	// container must run as a non-root uid
	RunAsUserStrategyMustRunAsNonRoot RunAsUserStrategyType = "MustRunAsNonRoot"
	// container may make requests for any uid.
	RunAsUserStrategyRunAsAny RunAsUserStrategyType = "RunAsAny"
)

// PodSecurityPolicyList is a list of PodSecurityPolicy objects
type PodSecurityPolicyList struct {
	kapi.TypeMeta
	kapi.ListMeta

	Items []PodSecurityPolicy
}
```

### PodSecurityPolicy Lifecycle

As reusable objects in the root scope, PodSecurityPolicy follows the lifecycle of the
cluster itself.  Maintenance of constraints such as adding, assigning, or changing them is the
responsibility of the cluster administrator.

Creating a new user within a namespace should not require the cluster administrator to
define the user's PodSecurityPolicy.  They should receive the default set of policies
that the administrator has defined for the groups they are assigned.


## Default PodSecurityPolicy And Overrides

In order to establish policy for service accounts and users there must be a way
to identify the default set of constraints that is to be used.  This is best accomplished by using
groups.  As mentioned above, groups may be used by the authentication/authorization layer to ensure
that every user maps to at least one group (with a default example of `system:authenticated`) and it
is up to the cluster administrator to ensure that a PodSecurityPolicy object exists that
references the group.

If an administrator would like to provide a user with a changed set of security context permissions
they may do the following:

1.  Create a new PodSecurityPolicy object and add a reference to the user or a group
that the user belongs to.
1.  Add the user (or group) to an existing PodSecurityPolicy object with the proper
elevated privileges.

## Admission

Admission control using an authorizer allows the ability to control the creation of resources
based on capabilities granted to a user.  In terms of the PodSecurityPolicy it means
that an admission controller may inspect the user info made available in the context to retrieve
and appropriate set of policies for validation.

The appropriate set of PodSecurityPolicies is defined as all of the policies
available that have reference to the user or groups that the user belongs to.

Admission will use the PodSecurityPolicy to ensure that any requests for a
specific security context setting are valid and to generate settings using the following approach:

1.  Determine all the available PodSecurityPolicy objects that are allowed to be used
1.  Sort the PodSecurityPolicy objects in a most restrictive to least restrictive order.
1.  For each PodSecurityPolicy, generate a SecurityContext for each container.  The generation phase will not override
and user requested settings in the SecurityContext and will rely on the validation phase to ensure that
the user requests are valid.
1.  Validate the generated SecurityContext to ensure it falls within the boundaries of the PodSecurityPolicy
1.  If all containers validate under a single PodSecurityPolicy then the pod will be admitted
1.  If all containers DO NOT validate under the PodSecurityPolicy then try the next PodSecurityPolicy
1.  If no PodSecurityPolicy validates for the pod then the pod will not be admitted


## Creation of a SecurityContext Based on PodSecurityPolicy

The creation of a SecurityContext based on a PodSecurityPolicy is based upon the configured
settings of the PodSecurityPolicy.

There are three scenarios under which a PodSecurityPolicy field may fall:

1.  Governed by a boolean: fields of this type will be defaulted to the most restrictive value.
For instance, `AllowPrivileged` will always be set to false if unspecified.
1.  Governed by an allowable set: fields of this type will be checked against the set to ensure
their value is allowed.  For example, `AllowCapabilities` will ensure that only capabilities
that are allowed to be requested are considered valid.  `HostNetworkSources` will ensure that
only pods created from source X are allowed to request access to the host network.
1.  Governed by a strategy: Items that have a strategy to generate a value will provide a
mechanism to generate the value as well as a mechanism to ensure that a specified value falls into
the set of allowable values.  See the Types section for the description of the interfaces that
strategies must implement.

Strategies have the ability to become dynamic.  In order to support a dynamic strategy it should be
possible to make a strategy that has the ability to either be pre-populated with dynamic data by
another component (such as an admission controller) or has the ability to retrieve the information
itself based on the data in the pod.  An example of this would be a pre-allocated UID for the namespace.
A dynamic `RunAsUser` strategy could inspect the namespace of the pod in order to find the required pre-allocated
UID and generate or validate requests based on that information.


```go
// SELinuxStrategy defines the interface for all SELinux constraint strategies.
type SELinuxStrategy interface {
	// Generate creates the SELinuxOptions based on constraint rules.
	Generate(pod *api.Pod, container *api.Container) (*api.SELinuxOptions, error)
	// Validate ensures that the specified values fall within the range of the strategy.
	Validate(pod *api.Pod, container *api.Container) fielderrors.ValidationErrorList
}

// RunAsUserStrategy defines the interface for all uid constraint strategies.
type RunAsUserStrategy interface {
	// Generate creates the uid based on policy rules.
	Generate(pod *api.Pod, container *api.Container) (*int64, error)
	// Validate ensures that the specified values fall within the range of the strategy.
	Validate(pod *api.Pod, container *api.Container) fielderrors.ValidationErrorList
}
```

## Escalating Privileges by an Administrator

An administrator may wish to create a resource in a namespace that runs with
escalated privileges.   By allowing security context
constraints to operate on both the requesting user and pod's service account administrators are able to
create pods in namespaces with elevated privileges based on the administrator's security context
constraints.

This also allows the system to guard commands being executed in the non-conforming container.  For
instance, an `exec` command can first check the security context of the pod against the security
context constraints of the user or the user's ability to reference a service account.
If it does not validate then it can block users from executing the command.  Since the validation
will be user aware administrators would still be able to run the commands that are restricted to normal users.

## Interaction with the Kubelet

In certain cases, the Kubelet may need provide information about
the image in order to validate the security context.  An example of this is a cluster
that is configured to run with a UID strategy of `MustRunAsNonRoot`.

In this case the admission controller can set the existing `MustRunAsNonRoot` flag on the SecurityContext
based on the UID strategy of the SecurityPolicy.  It should still validate any requests on the pod
for a specific UID and fail early if possible.  However, if the `RunAsUser` is not set on the pod
it should still admit the pod and allow the Kubelet to ensure that the image does not run as
root with the existing non-root checks.




<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/proposals/security-context-constraints.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
