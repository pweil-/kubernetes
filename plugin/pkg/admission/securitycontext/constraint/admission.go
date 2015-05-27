package constraint

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/admission"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/auth/user"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/fields"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/securitycontextconstraints"
)

func init() {
	admission.RegisterPlugin("SecurityContextConstraint", func(client client.Interface, config io.Reader) (admission.Interface, error) {
		return NewConstraint(client), nil
	})
}

type constraint struct {
	*admission.Handler
	client client.Interface
}

var _ admission.Interface = &constaint{}

func NewConstraint(client client.Interface) admission.Interface {
	return &constraint{
		Handler: admission.NewHandler(admission.Create, admission.Update),
		client:  client,
	}
}

func (c *constraint) Admit(a admission.Attributes) error {
	// 1.  find all SCCs the pod has access to:
	//     1.  downconvert to api.Pod
	//     2.  get service account from pod
	//     3.  get service account object
	//     4.  turn SA into user info
	//     5.  look up SCCs for user/group
	// 2.  Fully resolve each SCC
	//     1.  determine uid from the SCC's run as user policy
	//     2.  set the uid on the SCC
	// 3.  Match the pod's SC to an SCC; for each SCC:
	//     1.  generate the SC
	//     2.  validate against the pod's SC

	if a.GetResource() != "pods" {
		return nil
	}

	pod, ok := a.GetObject().(api.Pod)
	if !ok {
		return errors.NewBadRequest("a pod was received, but could not convert the request object.")
	}

	serviceAccountName := pod.Spec.serviceAccount
	if serviceAccountName == "" {
		return errors.NewBadRequest("pod with no service account")
	}

	ns := a.GetNamespace()

	serviceAccount, err := c.client.ServiceAccounts(ns).Get(serviceAccountName)
	if err != nil {
		return err
	}

	constraints, err := c.client.SecurityContextConstraints().List(labels.Everything(), fields.Everything())
	if err != nil {
		return err
	}

	userInfo := UserInfo(ns, serviceAccountName, string(serviceAccount.UID))

	matchedConstraints := make([]api.SecurityContextConstraint)

	for _, constraint := range constraints.Items {
		for _, group := range constraint.Groups {
			if userInfo.Group == group {
				matchedConstraints = append(matchedConstraints, constraint)
				break
			}
		}

		for _, user := range constraint.Users {
			if userInfo.User == user {
				matchedConstraints = append(matchedConstraints, constraint)
				break
			}
		}
	}

	providers := make([]SecurityContextConstraintsAllocator)

	for _, constraint := range matchedConstraints {
		provider, err := securitycontextconstraints.NewSimpleAllocator(constraint, c.client)
		if err != nil {
			return err
		}

		providers = append(providers, provider)
	}

outer:
	for i, container := range pod.Spec.Containers {
		for _, provider := range providers {
			context, err := provider.CreateSecurityContext(pod, container, nil)
			podCopyObj, err := api.Scheme.Copy(pod)
			if err != nil {
				return err
			}
			podCopy, ok := podCopyObj.(api.Pod)
			if !ok {
				return errors.NewBadRequest("pod copy failed")
			}
			podCopy.Spec.Containers[i].SecurityContext = context

			errs := provider.ValidateSecurityContext(podCopy, podCopy.Containers[i], nil)
			if len(errs) != 0 {
				continue
			}

			container.SecurityContext = context
			continue outer
		}
	}

	return nil
}

func UserInfo(namespace, name, uid string) user.Info {
	return &user.DefaultInfo{}
}
