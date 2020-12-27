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
	"strconv"
	"strings"

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


const (
	ControlField = "control"
	VersionField = "Version"
	GlobalField = "Global"
	TiDBField = "TiDB"
	TiKVField = "TiKV"
	PDField   = "PD"
	PumpField = "Pump"
	DrainerField = "Drainer"
	MonitorField = "Monitor"
	GrafanaField = "Grafana"
)

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

	result := getEmptyRequiredFields(r.Spec.TiDBCluster)
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

	if checkUnSupportComponentsChanged(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) {
		return fmt.Errorf("update unsupport component")
	}

	if checkScale(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) && checkServerConfigModified(&tct.Status.PreTiDBCluster.ServerConfigs, &r.Spec.TiDBCluster.ServerConfigs) {
		return fmt.Errorf("update and scale-in/out cluster can't use at the same time")
	}

	if checkSimultaneousScaleOutAndScaleIn(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) {
		return fmt.Errorf("scale-in/out cluster can't use at the same time")
	}

	if checkImmutableFieldChanged(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) {
		return fmt.Errorf("immutable field is changed")
	}

	result := getEmptyRequiredFields(r.Spec.TiDBCluster)
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

func checkServerConfigModified(pre *ServerConfigs, cur *ServerConfigs) bool {
	// TODO tidbcluster can't be null,check in webhook
	return !reflect.DeepEqual(pre, cur)
}

func checkScale(pre *TiDBCluster, cur *TiDBCluster) bool {
	// TODO check
	return len(pre.TiDB) != len(cur.TiDB) || len(pre.PD) != len(cur.PD) || len(pre.TiKV) != len(cur.TiKV)
}

func checkSimultaneousScaleOutAndScaleIn(pre *TiDBCluster, cur *TiDBCluster) bool {
	// TODO check
	scaleIn := len(pre.TiDB) > len(cur.TiDB) || len(pre.PD) > len(cur.PD) || len(pre.TiKV) > len(cur.TiKV)
	scaleOut := len(pre.TiDB) < len(cur.TiDB) || len(pre.PD) < len(cur.PD) || len(pre.TiKV) < len(cur.TiKV)
	if scaleIn && scaleOut {
		return true
	}
	if !scaleIn && !scaleOut {
		// update some host, like tikv(n1,n2,n3)--->tikv(n1,n2,n4)
		checkComponents := []string{TiDBField, PDField, TiKVField}
		preVal := reflect.ValueOf(*pre)
		curVal := reflect.ValueOf(*cur)
		for i := 0; i < curVal.Type().NumField(); i++ {
			if checkIn(checkComponents, curVal.Type().Field(i).Name) {
				preField := preVal.Field(i)
				curField := curVal.Field(i)
				var isExist bool
				for j := 0; j < preField.Len(); j++ {
					for k := 0; k < curField.Len(); k++ {
						if preField.Index(j).FieldByName("Host").String() == curField.Index(k).FieldByName("Host").String() {
							isExist = true
							break
						}
					}
				}
				if !isExist {
					return true
				}
			}
		}

	}
	return false
}

func checkImmutableFieldChanged(pre *TiDBCluster, cur *TiDBCluster) bool {
	checkComponents := []string{TiDBField, PDField, TiKVField}
	preVal := reflect.ValueOf(*pre)
	curVal := reflect.ValueOf(*cur)
	for i := 0; i < curVal.Type().NumField(); i++ {
		if checkIn(checkComponents, curVal.Type().Field(i).Name) {
			preField := preVal.Field(i)
			curField := curVal.Field(i)
			for j := 0; j < preField.Len(); j++ {
				for k := 0; k < curField.Len(); k++ {
					if preField.Index(j).FieldByName("host") == curField.Index(k).FieldByName("host") {
						if !reflect.DeepEqual(preField.Index(j).Interface(),curField.Index(k).Interface()){
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func checkInclusion(pre *TiDBCluster, cur *TiDBCluster) bool {
	checkComponents := []string{TiDBField, PDField, TiKVField}
	preVal := reflect.ValueOf(*pre)
	curVal := reflect.ValueOf(*cur)
	for i := 0; i < curVal.Type().NumField(); i++ {
		if checkIn(checkComponents, curVal.Type().Field(i).Name) {
			preField := preVal.Field(i)
			curField := curVal.Field(i)
			var isExist bool
			for j := 0; j < preField.Len(); j++ {
				for k := 0; k < curField.Len(); k++ {
					if reflect.DeepEqual(preField.Index(j).Interface(), curField.Index(k).Interface()) {
						isExist = true
						break
					}
				}
			}
			if !isExist {
				return false
			}
		}
	}
	return true
}

func getEmptyRequiredFields(cur *TiDBCluster) []string {
	var tips []string
	if cur.Global != nil && cur.Global.DeployDir != "" && cur.Global.DataDir != "" {
		return tips
	}
	curVal := reflect.ValueOf(*cur)
	checkMaps := map[string][]string{
		TiDBField: {"DeployDir"},
		PDField:   {"DeployDir", "DataDir"},
		TiKVField: {"DeployDir", "DataDir"},
	}

	prefix := "spec.tidbCluster"
	for key, val := range checkMaps {
		components := curVal.FieldByName(key)
		for i := 0; i < components.Len(); i++ {
			for j := 0; j < len(val); j++ {
				if components.Index(i).FieldByName(val[j]).String() == "" && curVal.FieldByName("Global").IsNil() {
					tmp := strings.ToLower(key) + "[" + strconv.Itoa(i) + "]"
					tips = append(tips, strings.Join([]string{prefix, tmp, val[j]}, "."))
					continue
				}
				if components.Index(i).FieldByName(val[j]).String() == "" && curVal.FieldByName("Global").FieldByName(val[j]).String() == "" {
					tmp := strings.ToLower(key) + "[" + strconv.Itoa(i) + "]"
					tips = append(tips, strings.Join([]string{prefix, tmp, val[j]}, "."))
				}
			}
		}
	}
	return tips
}

func checkUnSupportComponentsChanged(pre *TiDBCluster, cur *TiDBCluster) bool {
	unSupportComponents := []string{GlobalField, DrainerField , PumpField, VersionField , MonitorField , ControlField, GrafanaField}
	preVal := reflect.ValueOf(*pre)
	curVal := reflect.ValueOf(*cur)
	for i := 0; i < curVal.Type().NumField(); i++ {
		if checkIn(unSupportComponents, curVal.Type().Field(i).Name) {
			preField := preVal.Field(i)
			curField := curVal.Field(i)
			if !reflect.DeepEqual(preField.Interface(), curField.Interface()) {
				return true
			}
		}
	}
	return false
}

func checkIn(lists []string, str string) bool {
	for i := 0; i < len(lists); i++ {
		if lists[i] == str {
			return true
		}
	}
	return false
}
