package cmd

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naglfarv1 "github.com/PingCAP-QE/Naglfar/api/v1"
	"github.com/PingCAP-QE/Naglfar/pkg/kubeutil"
)

var (
	workloadItemName string
	follow           bool
)

func NewLogsCmd(logger *zap.Logger, configFlag *genericclioptions.ConfigFlags, rbFlags *genericclioptions.ResourceBuilderFlags, streams genericclioptions.IOStreams) *cobra.Command {
	logCmd := &cobra.Command{
		Use:   "logs",
		Short: "tail logs of a test workload",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cmd.SetOut(streams.ErrOut)
			cmd.SetErr(streams.ErrOut)
		},
		Args: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			c.SetOut(streams.ErrOut)
			c.SetErr(streams.ErrOut)

			var err error
			config, err := configFlag.ToRESTConfig()
			if err != nil {
				return err
			}
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return err
			}
			client, err := client.New(config, client.Options{
				Scheme: scheme,
			})
			if err != nil {
				return err
			}
			naglfarClient = &NaglfarClient{
				Client:    client,
				Clientset: clientset,
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if len(args) < 1 {
				return fmt.Errorf("should set the test workload name")
			}
			twName := args[0]

			var tw = naglfarv1.TestWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      twName,
					Namespace: kubeutil.GetNamespaceFromKubernetesFlags(configFlag, rbFlags),
				},
			}
			if err := naglfarClient.GetObject(ctx, &tw); err != nil {
				return err
			}
			return logTestWorkload(ctx, logger, &tw, streams)
		},
	}
	flagsLog := pflag.NewFlagSet("kubectl-naglfar-log", pflag.ExitOnError)
	flagsLog.StringVarP(&workloadItemName, "workload", "w", "", "set the workload name")
	flagsLog.BoolVar(&follow, "follow", false, "follow the log")
	logCmd.Flags().AddFlagSet(flagsLog)
	return logCmd
}

func logTestWorkload(ctx context.Context, logger *zap.Logger, tw *naglfarv1.TestWorkload, streams genericclioptions.IOStreams) error {
	if tw.Status.State == naglfarv1.TestWorkloadStatePending {
		return fmt.Errorf("workload %s/%s is pending, please wait it to run", tw.Namespace, tw.Name)
	}
	if len(tw.Spec.Workloads) == 1 && len(workloadItemName) == 0 {
		workloadItemName = tw.Spec.Workloads[0].Name
	}

	var targetItem *naglfarv1.TestWorkloadItemSpec
	for _, item := range tw.Spec.Workloads {
		if item.Name == workloadItemName {
			targetItem = &item
		}
	}
	if targetItem == nil {
		return fmt.Errorf("there exist no item named %s", workloadItemName)
	}

	node, err := naglfarClient.GetResource(ctx, tw.Namespace, targetItem.DockerContainer.ResourceRequest.Node)
	if err != nil {
		return err
	}
	machine, err := naglfarClient.GetMachineByHostIP(ctx, node.Status.HostIP)
	if err != nil {
		return err
	}
	dockerClient, err := docker.NewClient(machine.DockerURL(), machine.Spec.DockerVersion, nil, nil)
	responseBody, err := dockerClient.ContainerLogs(ctx, fmt.Sprintf("%s.%s", tw.Namespace, node.Name), types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Details:    false,
	})
	if err != nil {
		return err
	}
	_, err = stdcopy.StdCopy(streams.Out, streams.ErrOut, responseBody)
	return err
}
