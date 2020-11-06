// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ref

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Ref struct {
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	UID       types.UID `json:"uid"`
}

func CreateRef(meta *metav1.ObjectMeta) Ref {
	return Ref{
		Namespace: meta.Namespace,
		Name:      meta.Name,
		UID:       meta.UID,
	}
}

func ParseKey(key string) Ref {
	tokens := strings.Split(key, "/")
	if len(tokens) != 3 {
		panic("invalid key: " + key)
	}
	return Ref{
		Namespace: tokens[0],
		Name:      tokens[1],
		UID:       types.UID(tokens[2]),
	}
}

func (r *Ref) Namespaced() types.NamespacedName {
	return types.NamespacedName{
		Namespace: r.Namespace,
		Name:      r.Name,
	}
}

func (r *Ref) Key() string {
	return fmt.Sprintf("%s/%s/%s", r.Namespace, r.Name, r.UID)
}
