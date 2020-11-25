package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/kubeutil"
)

var (
	scheme        *runtime.Scheme
	naglfarClient *NaglfarClient
)

type NaglfarClient struct {
	client.Client
	*kubernetes.Clientset
}

func (c *NaglfarClient) GetObject(ctx context.Context, obj runtime.Object) error {
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return err
	}
	err = kubeutil.Retry(3, time.Second, func() error {
		if err := c.Get(ctx, key, obj); err != nil {
			return err
		}
		return nil
	})
	return err
}

func (c *NaglfarClient) GetResource(ctx context.Context, ns string, name string) (*naglfarv1.TestResource, error) {
	resource := naglfarv1.TestResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}
	err := c.GetObject(ctx, &resource)
	return &resource, err
}

func (c *NaglfarClient) GetMachine(ctx context.Context, name string) (*naglfarv1.Machine, error) {
	machine := naglfarv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
		},
	}
	err := c.GetObject(ctx, &machine)
	return &machine, err
}

func NewNaglfarCmd(logger *zap.Logger, streams genericclioptions.IOStreams) *cobra.Command {
	flags := pflag.NewFlagSet("kubectl-naglfar", pflag.ExitOnError)
	pflag.CommandLine = flags

	kubeConfigFlags := genericclioptions.NewConfigFlags(false)
	kubeResourceBuilderFlags := genericclioptions.NewResourceBuilderFlags()

	rootCmd := &cobra.Command{
		Use:   "kubectl-naglfar",
		Short: "It is a kubectl plugin that you can use to manager Naglfar",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	flags.AddFlagSet(rootCmd.PersistentFlags())
	kubeConfigFlags.AddFlags(flags)
	kubeResourceBuilderFlags.WithLabelSelector("")
	kubeResourceBuilderFlags.WithAllNamespaces(false)
	kubeResourceBuilderFlags.AddFlags(flags)

	rootCmd.AddCommand(NewLogsCmd(logger, kubeConfigFlags, kubeResourceBuilderFlags, streams))

	return rootCmd
}

func init() {
	scheme = runtime.NewScheme()
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	utilruntime.Must(kubescheme.AddToScheme(scheme))
	utilruntime.Must(naglfarv1.AddToScheme(scheme))
}
