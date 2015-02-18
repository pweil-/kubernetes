/*
Copyright 2015 Google Inc. All rights reserved.

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

package autoscaler

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/rest"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/validation"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
)

// REST adapts an autoscaler registry into apiserver's RESTStorage model.
type REST struct {
	registry Registry
}

// NewREST returns a new REST.
func NewREST(registry Registry) *REST {
	return &REST{
		registry: registry,
	}
}

// NewList returns an empty object that can be used with the List call.
func (rs *REST) NewList() runtime.Object {
	return &api.AutoScalerList{}
}

// List selects resources in the storage which match to the selector.
func (rs *REST) List(ctx api.Context, label, field labels.Selector) (runtime.Object, error) {
	list, err := rs.registry.ListAutoScalers(ctx)

	if err != nil {
		return nil, err
	}

	var filtered []api.AutoScaler

	for _, autoScaler := range list.Items {
		if label.Matches(labels.Set(autoScaler.Labels)) {
			filtered = append(filtered, autoScaler)
		}
	}

	list.Items = filtered
	return list, err
}

// Get finds a resource in the storage by id and returns it.
func (rs *REST) Get(ctx api.Context, id string) (runtime.Object, error) {
	autoScaler, err := rs.registry.GetAutoScaler(ctx, id)
	if err != nil {
		return nil, err
	}
	return autoScaler, err
}

// Delete finds a resource in the storage and deletes it.
func (rs *REST) Delete(ctx api.Context, id string) (runtime.Object, error) {
	//TODO de-register if necessary
	return &api.Status{Status: api.StatusSuccess}, rs.registry.DeleteAutoScaler(ctx, id)
}

// New returns an empty object that can be used with Create after request data has been put into it.
func (rs *REST) New() runtime.Object {
	return &api.AutoScaler{}
}

// Create creates a new version of a resource.
func (rs *REST) Create(ctx api.Context, obj runtime.Object) (runtime.Object, error) {
	autoScaler := obj.(*api.AutoScaler)

	if err := rest.BeforeCreate(rest.AutoScalers, ctx, obj); err != nil {
		return nil, err
	}

	if err := rs.registry.CreateAutoScaler(ctx, autoScaler); err != nil {
		err = rest.CheckGeneratedNameError(rest.AutoScalers, err, autoScaler)
		return nil, err
	}

	return rs.registry.GetAutoScaler(ctx, autoScaler.Name)
}

// Update finds a resource in the storage and updates it.
func (rs *REST) Update(ctx api.Context, obj runtime.Object) (runtime.Object, bool, error) {
	autoScaler := obj.(*api.AutoScaler)
	oldAutoScaler, err := rs.registry.GetAutoScaler(ctx, autoScaler.Name)

	if err != nil {
		return nil, false, err
	}

	// copy over non-user fields
	if errs := validation.ValidateAutoScalerUpdate(oldAutoScaler, autoScaler); len(errs) > 0 {
		return nil, false, errors.NewInvalid("autoScaler", autoScaler.Name, errs)
	}

	err = rs.registry.UpdateAutoScaler(ctx, autoScaler)

	if err != nil {
		return nil, false, err
	}

	out, err := rs.registry.GetAutoScaler(ctx, autoScaler.Name)
	return out, false, err
}

// Watch provides the ability to watch for changes to selected objects
func (rs *REST) Watch(ctx api.Context, label, field labels.Selector, resourceVersion string) (watch.Interface, error) {
	return rs.registry.WatchAutoScalers(ctx, label, field, resourceVersion)
}
