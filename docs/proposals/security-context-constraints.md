## Abstract

Security context constraints relate [security contexts](./security_contexts.md) and
[service accounts](http://docs.k8s.io/design/service_accounts.md).  They are configured for a
cluster, namespaces, and users within the cluster to govern how a [security context](./security_contexts.md) is established.

## Motivation

Administration of a multi-tenant cluster requires the ability to provide varying sets of permissions
among the tenants, the infrastructure components, and end users of the system who may themselves be
administrators within their own isolated namespace.
Actors in a cluster may include infrastructure that is managed by administrators, infrastructure
that is exposed to end users (builds, deployments), the isolated end user namespaces in the cluster, and
the individual users inside those namespaces.  Infrastructure components that operate on behalf of a
user (builds, deployments) should be allowed to run at an elevated level of permissions without
granting the user themselves an elevated set of permissions.

### Goals

1.  Associate [service accounts](http://docs.k8s.io/design/service_accounts.md) and identities with
a set of constraints that dictate how a [security context](./security_contexts.md) is established for a pod.
1.  Provide the ability for users and infrastructure components to run pods with elevated privileges
on behalf of another user or within a namespace where privileges are more restrictive.
1.  Secure the ability to reference elevated permissions or to change the constraints under which
a user runs.

### Use Cases

Use cases 1:
As an administrator, I can create a namespace for a person that can't create privileged containers
AND enforces that the UID of the containers is set to a certain value

Use cases 2:
A build controller should be able to create a privileged pod in a namespace, but regular users
can't create privileged pods AND regular users can't exec into that pod.

Use cases 3:
As a cluster administrator, I can allow a given namespace (or service account) to create privileged
pods or to run root pods

Use case 4:
As a cluster administrator, I can allow a project administrator to control the security contexts of
pods and service accounts within a project


### Requirements

1.  Provide a set of restrictions that a security context operates under a new object will be
introduced called SecurityContextConstraints.
1.  SecurityContextConstraints may also exist outside of a namespace.
1.  A cluster default service account will be created for the default namespace and be available to
admission controllers via configuration.
1.  User information must be available to admission controllers.
1.  Some authorizers may restrict a user’s ability to reference a service account.  Systems requiring
the ability to secure service accounts on a user level will be able to add a policy that enables
referencing specific service accounts themselves.
1.  Admission control must validate the creation of Pods, PodTemplates, and Replication controllers
against the allowed set of constraints.

### Design

#### Model

The model for a security context constraint is a combination of the security context constraint
itself and cluster wide strategies enforced by an allocator.  The separation of concerns falls into
two distinct categories: permissions that the pod can request and runtime parameters the cluster
admin controls.

An example of a permission that can be requested is to run a pod as privileged.  This is set on a
pod by pod basis.  A cluster wide setting may be the allocation of user ids that pods must run under.
The cluster administrator may decide that no pod can run as the root user in their cluster and
provide a block of UIDs that the allocator can use to distribute amongst namespaces.

A security context constraint object exists in the root scope, outside of a namespace.  The
security context constraints can reference users, groups, or a service account that is allowed
to operate under the constraints.

A one to one relationship will be maintained for a security context constraint and a service account.
Users may be allowed to have many security context constraints and admission will create a unioned
set of their highest allowed permissions when creating a pod (discussed in admission, below).

To minimize the amount of objects that must be managed by a cluster administrator the following
security context constraints objects can be provided as a default set:

1.  User/Group Constraints - this is the default security context constraints for the cluster and is usually the
most restrictive set of constraints.
1.  Service Account Constraints - security context constraints for service accounts will be created that allows
pods to run with different permissions than the user.  The user should be required to have
permissions to reference the service account in order to run a pod with the service account's
security context constraints.
1.  Administrator Constraints - a set of constraints with elevated privileges that can be used
by an administrative user or group.
1.  Infrastructure Component Constraints - constraints that may have elevated privileges within the
cluster but possibly not as elevated as administrator constraints.

```go
// SecurityContextConstraints governs the ability to make requests that affect the SecurityContext that will
// be applied to a container.
type SecurityContextConstraints struct {
  // Type distinguishes the type of SecurityContextConstraints object.  Acceptable values are
  // INFRA, USER, DEFAULT_USER, SERVICE_ACCOUNT, DEFAULT_SERVICE_ACCOUNT
  Type string
  // AllowPrivilegedContainer determines if a container can request to be run as privileged.
  AllowPrivilegedContainer bool
  // AllowedCapabilities is a list of capabilities that can be requested to add to the container.
  AllowedCapabilities []CapabilityType
  // AllowHostDirVolumePlugin determines if the policy allow containers to use the HostDir volume plugin
  AllowHostDirVolumePlugin bool
  // The users who have permissions to use this security context constraints
  AllowedUsers []string
  // The service accounts that have permission to use this security context constraints
  AllowedServiceAccounts []NamespacedName
}

type SecurityContextConstraintsAllocator interface {
  // Create a SecurityContext based on the given constraints
  CreateSecurityConstraints(constraints *api.SecurityContextConstraints) api.SecurityContext
  // Ensure a container's SecurityContext is in compliance with the given constraints
  ValidateSecurityConstraints(container *api.Container, constraints *api.SecurityContextConstraints) fielderrors.ValidationErrorList
}

// implements the SecurityContextConstraintsAllocator interface
type SimpleSecurityContextConstraintsAllocator struct {
    // SELinuxContext is the strategy that will dictate what labels will be set in the SecurityContext.
    SELinuxContext SELinuxContextStrategy
    // RunAsUser is the strategy that will dictate what RunAsUser is used in the SecurityContext.
    RunAsUser RunAsUserStrategy
}

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

#### Security Context Constraints Lifecycle

As reusable objects in the root scope, security context constraints follow the lifecycle of the
cluster itself.  Maintenance of constraints such as adding, assigning, or changing them is the
responsibility of the cluster administrator but is assisted by automation within the cluster.  For
instance, creating a new user within a namespace should not require the cluster administrator to
define the user's security context constraints.  They should receive the default set of constraints
that the administrator has defined for users.

When a service account or user is removed finalizers should ensure that the references from the security context
restraints are also removed in order to avoid the risk of having the same user or service account
receive permissions inadvertently.  This can also be avoided by using a unique identifier like
the object UID in the security context constraints list of allowed users but clean up should
still occur to avoid unnecessary object growth.

#### Default Security Context Constraints And Overrides

In order to establish security context constraints for service accounts and users there must be a way
to identify the default set of constraints that is to be used.  In order to identify the constraints
a type field or default flag may be used on the security context constraints object.

To avoid unnecessarily large objects with references to many users in the system the constraints
identified as the default for the object type can be considered referenceable by all objects
of that type.  For example, rather than listing every user in the cluster in the default user
security context constraint object the mechanism that retrieves the available security context
constraints for the user may fetch all constraints with specific references to the user and
the default constraints.

If an administrator would like to provide a user with a changed set of security context permissions
they may do the following:

1.  Create a new security context constraints object and add a reference to the user.
1.  Create a new security context constraints object and add a reference to a serivce account
that the user has access to reference.

#### Admission

Admission control using an authorizer allows the ability to control the creation of resources
based on capabilities granted to a user.  In terms of the security context constraints it means
that an admission controller may inspect the user info made available in the context to retrieve
and appropriate set of security context constraints for validation.

The appropriate set of security context constraints is defined as the set of security context
constraints that the user has references to the user, the default set of constraints for the
user type, and any security context constraints that they are allowed to reference via a service
account.

Admission will use the SecurityContextConstraintsAllocator to ensure that any requests for a
specific security context constraint setting or to generate settings using the following approach:

**Pod with referenced service account**

1.  Ensure that the user has permissions to reference the service account.
2.  Retrieve the service account's security context constraints.
3.  Ask the SecurityContextConstraintsAllocator to create the security context.  The allocator
will not overwrite fields already set in the container.
4.  Validate that the generated SecurityContext falls within the boundaries of the security context
constraints and accept or reject the pod.

**Pods with no service account**

1.  Create a union of the user level permissions in the constraints.  This is the highest level
under which they can create a pod if they are not referencing a service account.
2.  Ask the SecurityContextConstraintsAllocator to create the security context
3.  Ask the SecurityContextConstraintsAllocator to create the security context.  The allocator
will not overwrite fields already set in the container.
4.  Validate that the generated SecurityContext falls within the boundaries of the security context
constraints and accept or reject the pod.

#### Creation of a Security Context Based on Security Context Constraints

The creation of a security context based on security context constraints is based upon the configured
settings of the security context constraints and the cluster wide strategies enforce by the
allocator (discussed below).

There are three scenarios under which a security context constraint field may fall:

1.  Governed by a boolean: fields of this type will be defaulted to the most restrictive value.
For instance, `AllowPrivilegedContainer` will always be set to false if unspecified.
1.  Goverened by an allowable set: fields of this type will be checked against the set to ensure
their value is allowed.  For example, `AllowedCapabilities` will ensure that only capabilities
that are allowed to be requested are considered valid.  `HostNetworkSources` will ensure that
only pods created from source X are allowed to request access to the host network.
1.  Governed by a strategy: Items that have a strategy to generate a value will provide a
mechanism to generate the value as well as a mechanism to ensure that a specified value falls into
the set of allowable values.  See the Types section for the description of the interfaces that
strategies must implement.

Some items of the security context only need to be allocated a single time.  For instance if the
cluster defines that each namespace should have a block of UIDs used for allocation to service
accounts then the allocator may allocate UID 1000 to service account X for the lifetime of the
service account.  This allocation can be stored as an annotation on the namespace level so
that it is uneditable.  Allocation of a UID to a user may be stored on the user object.  When
a security context is generated on behalf of security context constraints it should first check
the annotations for pre-allocated values and use them if appropriate.

Some strategies such as a "allow pods to run as any user" will not need to store annotations.

The allocator will use the configured strategies with interfaces shown below to either validate
the settings fall within the allowed values or create a new value.

```go
// SELinuxSecurityContextConstraintsStrategy defines the interface for all SELinux constraint strategies.
type SELinuxSecurityContextConstraintsStrategy interface {
  // Generate creates the SELinuxOptions based on constraint rules.
  Generate(podSecurityContext *api.SecurityContext) *api.SELinuxOptions
  // Validate ensures that the specified values fall within the range of the strategy.
  Validate(*api.SELinuxOptions) fielderrors.ValidationErrorList
}

// RunAsUserSecurityContextConstraintsStrategy defines the interface for all uid constraint strategies.
type RunAsUserSecurityContextConstraintsStrategy interface {
  // Generate creates the uid based on policy rules.
  Generate(podSecurityContext *api.SecurityContext) *int64
  // Validate ensures that the specified values fall within the range of the strategy.
   Validate(runAsUser *int64) fielderrors.ValidationErrorList
}
```

#### Escalating Privileges by an Administrator

An administrator may wish to create a resource in a namespace that runs with
escalated privileges.  This is similar to saying `sudo make me a pod`.  By allowing security context
constraints to operate on both an user and service account level, administrators are able to
create pods in namespaces with elevated privileges based on the administrator's security context
constraints.

This also allows the system to guard commands being executed in the non-conforming container.  For
instance, an `exec` command can first check the security context of the pod against the security
context constraints of the user or the user's ability to reference a service account.
If it does not validate then it can block users from executing the command.  Since the validation
will be user aware administrators would still be able to run the commands that are restricted to normal users.

#### Interaction with the Kubelet

In some cases interaction with the kubelet is necessary.  An example of this is a cluster
that is configured to run with a UID strategy of `RunAsUserStrategyMustRunAsNonRoot` but without
a UID allocator.

In this case the validation can either require that the pod be submitted with a non-root `RunAsUser`
set in the security context or assume that images run in the cluster will be providing the user.  In
this case, if a pod reaches the kubelet and does not have a user set by the image then the kubelet
should not run the pod.

