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
	ControlField      = "Control"
	VersionField      = "Version"
	GlobalField       = "Global"
	TiDBField         = "TiDB"
	TiKVField         = "TiKV"
	PDField           = "PD"
	PumpField         = "Pump"
	DrainerField      = "Drainer"
	MonitorField      = "Monitor"
	GrafanaField      = "Grafana"
	MasterField       = "Master"
	WorkerField       = "Worker"
	AlertManagerField = "AlertManager"
	TiFlashField      = "TiFlash"
	CDCField          = "CDC"
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

	clusterNum := countClusterNum(r)
	if clusterNum > 1 {
		return fmt.Errorf("only one cluster can be created at a time")
	}

	switch {
	case r.Spec.TiDBCluster != nil:
		result := getEmptyRequiredFields(r.Spec.TiDBCluster)
		if len(result) != 0 {
			return fmt.Errorf("you must fill %v", result)
		}
	case r.Spec.FlinkCluster != nil:
	case r.Spec.DMCluster != nil:
	}

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *TestClusterTopology) ValidateUpdate(old runtime.Object) error {
	testclustertopologylog.Info("validate update", "name", r.Name)
	tct := old.(*TestClusterTopology)
	switch {
	case r.Spec.TiDBCluster != nil:
		return r.validateTiDBUpdate(tct)
	case r.Spec.FlinkCluster != nil:
	case r.Spec.DMCluster != nil:
	}

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

func (r *TestClusterTopology) validateTiDBUpdate(tct *TestClusterTopology) error {
	if r.Spec.TiDBCluster == nil || tct.Status.PreTiDBCluster == nil {
		return nil
	}

	if checkUnsupportedComponentsChanged(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) {
		return fmt.Errorf("update unsupport components")
	}

	if !checkOnlyOneUpdation(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) {
		return fmt.Errorf("only one of [upgrade, modify serverConfigs, scale-in/out] can be executed at a time")
	}
	if !checkUpgradePolicy(r.Spec.TiDBCluster) {
		return fmt.Errorf("upgradePolicy must be `force` or empty")
	}

	if checkVersionDownloadURL(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) {
		return fmt.Errorf("don't support update downLoadURL")
	}
	if checkSimultaneousScaleOutAndScaleIn(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) {
		return fmt.Errorf("cluster can't scale-in/out at the same time")
	}

	if checkImmutableFieldChanged(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) {
		return fmt.Errorf("immutable field is changed")
	}

	result := getEmptyRequiredFields(r.Spec.TiDBCluster)
	if len(result) != 0 {
		return fmt.Errorf("you must fill %v", result)
	}

	if IsScaleIn(tct.Status.PreTiDBCluster, r.Spec.TiDBCluster) && (len(r.Status.TiDBClusterInfo.PendingOfflineList) != 0 || len(r.Status.TiDBClusterInfo.OfflineList) != 0) {
		return fmt.Errorf("you must wait scale-in tikvs %v,%v complete the region migration", r.Status.TiDBClusterInfo.PendingOfflineList, r.Status.TiDBClusterInfo.OfflineList)
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
	return !reflect.DeepEqual(pre, cur)
}

func checkScale(pre *TiDBCluster, cur *TiDBCluster) bool {
	return len(pre.TiDB) != len(cur.TiDB) || len(pre.PD) != len(cur.PD) || len(pre.TiKV) != len(cur.TiKV)
}

// checkSimultaneousScaleOutAndScaleIn check if tidb cluster scale-out and scale-out at the same time
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
		for i := 0; i < len(checkComponents); i++ {
			preField := preVal.FieldByName(checkComponents[i])
			curField := curVal.FieldByName(checkComponents[i])
			if !preField.IsValid() || !curField.IsValid() {
				continue
			}
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
	return false
}

// checkImmutableFieldChanged check if immutable fields are changed, like spec.tidbCluster.tidb[i].dataDir
func checkImmutableFieldChanged(pre *TiDBCluster, cur *TiDBCluster) bool {
	checkComponents := []string{TiDBField, PDField, TiKVField}
	preVal := reflect.ValueOf(*pre)
	curVal := reflect.ValueOf(*cur)
	for i := 0; i < len(checkComponents); i++ {
		preField := preVal.FieldByName(checkComponents[i])
		curField := curVal.FieldByName(checkComponents[i])
		if !preField.IsValid() || !curField.IsValid() {
			continue
		}
		for j := 0; j < preField.Len(); j++ {
			for k := 0; k < curField.Len(); k++ {
				if preField.Index(j).FieldByName("Host").String() == curField.Index(k).FieldByName("Host").String() {
					if !reflect.DeepEqual(preField.Index(j).Interface(), curField.Index(k).Interface()) {
						return true
					}
				}
			}

		}
	}
	return false
}

// getEmptyRequiredFields return which required fields are empty
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
		if !components.IsValid() {
			continue
		}
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

// checkUnsupportedComponentsChanged return if unsupported components' fields are changed
func checkUnsupportedComponentsChanged(pre *TiDBCluster, cur *TiDBCluster) bool {
	unsupportedComponents := []string{GlobalField, DrainerField, PumpField, MonitorField, ControlField, GrafanaField}
	preVal := reflect.ValueOf(*pre)
	curVal := reflect.ValueOf(*cur)
	for i := 0; i < len(unsupportedComponents); i++ {
		preField := preVal.FieldByName(unsupportedComponents[i])
		curField := curVal.FieldByName(unsupportedComponents[i])
		if !preField.IsValid() || !curField.IsValid() {
			continue
		}
		if !reflect.DeepEqual(preField.Interface(), curField.Interface()) {
			return true
		}
	}
	return false
}

// checkIn return if str in lists
func checkIn(lists []string, str string) bool {
	for i := 0; i < len(lists); i++ {
		if lists[i] == str {
			return true
		}
	}
	return false
}

func IsScaleIn(pre *TiDBCluster, cur *TiDBCluster) bool {
	return len(pre.TiDB) > len(cur.TiDB) || len(pre.PD) > len(cur.PD) || len(pre.TiKV) > len(cur.TiKV)
}

// countClusterNum count the number of clusters in submitted  yaml
func countClusterNum(tct *TestClusterTopology) int {
	clusterNum := 0
	if tct.Spec.FlinkCluster != nil {
		clusterNum++
	}
	if tct.Spec.TiDBCluster != nil {
		clusterNum++
	}
	if tct.Spec.DMCluster != nil {
		clusterNum++
	}
	return clusterNum
}

func checkUpgrade(pre *TiDBCluster, cur *TiDBCluster) bool {
	return pre.Version.Version != cur.Version.Version
}

func checkVersionDownloadURL(pre *TiDBCluster, cur *TiDBCluster) bool {
	return pre.Version.PDDownloadURL != cur.Version.PDDownloadURL || pre.Version.TiDBDownloadURL != cur.Version.TiDBDownloadURL || pre.Version.TiKVDownloadURL != cur.Version.TiKVDownloadURL
}

func checkOnlyOneUpdation(pre *TiDBCluster, cur *TiDBCluster) bool {
	updatedModules := 0
	if checkScale(pre, cur) {
		updatedModules++
	}
	if checkServerConfigModified(&pre.ServerConfigs, &cur.ServerConfigs) {
		updatedModules++
	}
	if checkUpgrade(pre, cur) {
		updatedModules++
	}
	return updatedModules == 1
}

func checkUpgradePolicy(cur *TiDBCluster) bool {
	return cur.UpgradePolicy == "force" || cur.UpgradePolicy == ""
}
