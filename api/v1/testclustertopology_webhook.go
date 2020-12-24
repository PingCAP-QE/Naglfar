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

package v1

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var testclustertopologylog = logf.Log.WithName("testclustertopology-resource")

func (r *TestClusterTopology) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-naglfar-pingcap-com-v1-testclustertopology,mutating=true,failurePolicy=fail,groups=naglfar.pingcap.com,resources=testclustertopologies,verbs=create;update,versions=v1,name=mtestclustertopology.kb.io

var _ webhook.Defaulter = &TestClusterTopology{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *TestClusterTopology) Default() {
	testclustertopologylog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-naglfar-pingcap-com-v1-testclustertopology,mutating=false,failurePolicy=fail,groups=naglfar.pingcap.com,resources=testclustertopologies,versions=v1,name=vtestclustertopology.kb.io

var _ webhook.Validator = &TestClusterTopology{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *TestClusterTopology) ValidateCreate() error {
	testclustertopologylog.Info("validate create", "name", r.Name)

	result := isFilledAllNeed(r.Spec.TiDBCluster)
	if len(result) != 0 {
		return fmt.Errorf("you must fill %v", result)
	}

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *TestClusterTopology) ValidateUpdate(old runtime.Object) error {
	testclustertopologylog.Info("validate update", "name", r.Name)
	tct := old.(*TestClusterTopology)
	if tct.Status.PreTiDBCluster == nil || r.Spec.TiDBCluster == nil {
		return nil
	}

	if isUnSupport(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) {
		return fmt.Errorf("update unsupport component")
	}

	if isScale(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) && isServerConfigModified(&tct.Status.PreTiDBCluster.ServerConfigs, &r.Spec.TiDBCluster.ServerConfigs) {
		return fmt.Errorf("update and scale-in/out cluster can't use at the same time")
	}

	if isScaleInAndOut(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) {
		return fmt.Errorf("scale-in/out cluster can't use at the same time")
	}

	if isImmutableFieldChanged(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) {
		return fmt.Errorf("immutable field is changed")
	}

	result := isFilledAllNeed(r.Spec.TiDBCluster)
	if len(result) != 0 {
		return fmt.Errorf("you must fill %v", result)
	}

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *TestClusterTopology) ValidateDelete() error {
	testclustertopologylog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func isServerConfigModified(pre *ServerConfigs, cur *ServerConfigs) bool {
	// TODO tidbcluster can't be null,check in webhook
	return !reflect.DeepEqual(pre, cur)
}

func isClusterConfigModified(pre *TiDBCluster, cur *TiDBCluster) bool {
	// TODO tidbcluster can't be null,check in webhook
	return !reflect.DeepEqual(pre, cur)
}

func isScale(pre *TiDBCluster, cur *TiDBCluster) bool {
	// TODO check
	return len(pre.TiDB) != len(cur.TiDB) || len(pre.PD) != len(cur.PD) || len(pre.TiKV) != len(cur.TiKV)
}

func isScaleInAndOut(pre *TiDBCluster, cur *TiDBCluster) bool {
	// TODO check
	scaleIn := len(pre.TiDB) > len(cur.TiDB) || len(pre.PD) > len(cur.PD) || len(pre.TiKV) > len(cur.TiKV)
	scaleOut := len(pre.TiDB) < len(cur.TiDB) || len(pre.PD) < len(cur.PD) || len(pre.TiKV) < len(cur.TiKV)
	if scaleIn && scaleOut {
		return true
	}
	if !scaleIn && !scaleOut {
		// update some host, like tikv(n1,n2,n3)--->tikv(n1,n2,n4)
		for i := 0; i < len(pre.TiDB); i++ {
			var isExist bool
			for j := 0; j < len(cur.TiDB); j++ {
				if pre.TiDB[i].Host == cur.TiDB[j].Host {
					isExist = true
					break
				}
			}
			if !isExist {
				return true
			}
		}
		for i := 0; i < len(pre.PD); i++ {
			var isExist bool
			for j := 0; j < len(cur.PD); j++ {
				if pre.PD[i].Host == cur.PD[j].Host {
					isExist = true
					break
				}
			}
			if !isExist {
				return true
			}
		}
		for i := 0; i < len(pre.TiKV); i++ {
			var isExist bool
			for j := 0; j < len(cur.TiKV); j++ {
				if pre.TiKV[i].Host == cur.TiKV[j].Host {
					isExist = true
					break
				}
			}
			if !isExist {
				return true
			}
		}
	}
	return false
}

func isImmutableFieldChanged(pre *TiDBCluster, cur *TiDBCluster) bool {
	return !aIncludeB(pre, cur) && !aIncludeB(cur, pre)
}
func aIncludeB(pre *TiDBCluster, cur *TiDBCluster) bool {
	for i := 0; i < len(pre.TiDB); i++ {
		var isExist bool
		for j := 0; j < len(cur.TiDB); j++ {
			if reflect.DeepEqual(pre.TiDB[i], cur.TiDB[j]) {
				isExist = true
				break
			}
		}
		if !isExist {
			return false
		}
	}
	for i := 0; i < len(pre.PD); i++ {
		var isExist bool
		for j := 0; j < len(cur.PD); j++ {
			if reflect.DeepEqual(pre.PD[i], cur.PD[j]) {
				isExist = true
				break
			}
		}
		if !isExist {
			return false
		}
	}
	for i := 0; i < len(pre.TiKV); i++ {
		var isExist bool
		for j := 0; j < len(cur.TiKV); j++ {
			if reflect.DeepEqual(pre.TiKV[i], cur.TiKV[j]) {
				isExist = true
				break
			}
		}
		if !isExist {
			return false
		}
	}
	return true
}

func isFilledAllNeed(cur *TiDBCluster) []string {
	var tips []string
	if cur.Global != nil && cur.Global.DeployDir != "" && cur.Global.DataDir != "" {
		return tips
	}

	for i := 0; i < len(cur.TiDB); i++ {
		if cur.TiDB[i].DeployDir == "" && cur.Global == nil {
			tips = append(tips, "tidb/deployDir")
			break
		}
		if cur.TiDB[i].DeployDir == "" && cur.Global.DeployDir == "" {
			tips = append(tips, "tidb/deployDir")
			break
		}
	}
	for i := 0; i < len(cur.PD); i++ {
		if cur.PD[i].DeployDir == "" && cur.Global == nil {
			tips = append(tips, "pd/deployDir")
			break
		}
		if cur.PD[i].DeployDir == "" && cur.Global.DeployDir == "" {
			tips = append(tips, "pd/deployDir")
			break
		}

		if cur.PD[i].DataDir == "" && cur.Global == nil {
			tips = append(tips, "pd/dataDir")
			break
		}
		if cur.PD[i].DataDir == "" && cur.Global.DataDir == "" {
			tips = append(tips, "pd/dataDir")
			break
		}
	}
	for i := 0; i < len(cur.TiKV); i++ {
		if cur.TiKV[i].DeployDir == "" && cur.Global == nil {
			tips = append(tips, "tikv/deployDir")
			break
		}
		if cur.TiKV[i].DeployDir == "" && cur.Global.DeployDir == "" {
			tips = append(tips, "tikv/deployDir")
			break
		}

		if cur.TiKV[i].DataDir == "" && cur.Global == nil {
			tips = append(tips, "tikv/dataDir")
			break
		}
		if cur.TiKV[i].DataDir == "" && cur.Global.DataDir == "" {
			tips = append(tips, "tikv/dataDir")
			break
		}
	}
	return tips
}

func isUnSupport(pre *TiDBCluster, cur *TiDBCluster) bool {
	if !reflect.DeepEqual(pre.Global, cur.Global) {
		return true
	}
	if !reflect.DeepEqual(pre.Drainer, cur.Drainer) {
		return true
	}
	if !reflect.DeepEqual(pre.Pump, cur.Pump) {
		return true
	}
	if !reflect.DeepEqual(pre.Version, cur.Version) {
		return true
	}
	if !reflect.DeepEqual(pre.Monitor, cur.Monitor) {
		return true
	}
	if !reflect.DeepEqual(pre.Control, cur.Control) {
		return true
	}
	if !reflect.DeepEqual(pre.Grafana, cur.Grafana) {
		return true
	}
	return false
}
