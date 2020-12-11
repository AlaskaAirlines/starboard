package cmd

import (
	"context"

	"github.com/AlaskaAirlines/starboard/pkg/starboard"

	starboardapi "github.com/AlaskaAirlines/s/starboard/pkg/generated/clientset/versioned"
	"github.com/AlaskaAirlines/s/starboard/pkg/kubehunter"
	"github.com/AlaskaAirlines/s/starboard/pkg/kubehunter/crd"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

const (
	kubeHunterCmdShort = "Hunt for security weaknesses in your Kubernetes cluster"
)

func NewScanKubeHunterReportsCmd(cf *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubehunterreports",
		Short: kubeHunterCmdShort,
		RunE:  ScanKubeHunterReports(cf),
	}

	registerScannerOpts(cmd)

	return cmd
}

const (
	kubeHunterReportName = "cluster"
)

func ScanKubeHunterReports(cf *genericclioptions.ConfigFlags) func(cmd *cobra.Command, args []string) (err error) {
	return func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		config, err := cf.ToRESTConfig()
		if err != nil {
			return err
		}
		kubernetesClientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}
		opts, err := getScannerOpts(cmd)
		if err != nil {
			return err
		}
		report, err := kubehunter.NewScanner(starboard.NewScheme(), kubernetesClientset, opts).Scan(ctx)
		if err != nil {
			return err
		}
		starboardClientset, err := starboardapi.NewForConfig(config)
		if err != nil {
			return err
		}
		return crd.NewWriter(starboardClientset).Write(ctx, report, kubeHunterReportName)
	}
}
