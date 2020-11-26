package kubeutil

import (
	"fmt"
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	AllNamespace = ""
)

func GetNamespaceFromKubernetesFlags(
	configFlag *genericclioptions.ConfigFlags,
	rbFlags *genericclioptions.ResourceBuilderFlags) string {

	namespace := "default"
	if configFlag.Namespace != nil && *configFlag.Namespace != "" {
		namespace = *configFlag.Namespace
	}
	if rbFlags.AllNamespaces != nil && *rbFlags.AllNamespaces {
		namespace = AllNamespace
	}
	return namespace
}

func Retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; ; i++ {
		err = f()
		if err == nil {
			return
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
