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

package scadmit

import (
	"io"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/admission"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/plugin/pkg/admission/admit"
)

func init() {
	admission.RegisterPlugin("SecurityContextAdmit", func(client client.Interface, config io.Reader) (admission.Interface, error) {
		return NewAlwaysAdmit(), nil
	})
}

func NewAlwaysAdmit() admission.Interface {
	return admit.NewAlwaysAdmit()
}
