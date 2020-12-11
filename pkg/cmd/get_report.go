package cmd

import (
	"fmt"
	"os"

	"github.com/AlaskaAirlines/starboard/pkg/configauditreport"

	"github.com/AlaskaAirlines/starboard/pkg/vulnerabilityreport"

	clientset "github.com/AlaskaAirlines/starboard/pkg/generated/clientset/versioned"
	"github.com/AlaskaAirlines/starboard/pkg/report"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewGetReportCmd(cf *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report (NAME | TYPE/NAME)",
		Short: "Get a full html security report for a specified workload",
		Long: `Generates a report that contains vulnerabilities and config audits found for the specified workload

TYPE is a Kubernetes workload. Shortcuts and API groups will be resolved, e.g. 'po' or 'deployments.apps'.
NAME is the name of a particular Kubernetes workload.			
`,
		Example: fmt.Sprintf(`  # Save report to a file
  %[1]s get report deploy/nginx > report.html`, "starboard"),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := cf.ToRESTConfig()
			if err != nil {
				return err
			}
			starboardClientset, err := clientset.NewForConfig(config)
			if err != nil {
				return err
			}
			ns, _, err := cf.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return err
			}
			mapper, err := cf.ToRESTMapper()
			if err != nil {
				return err
			}
			workload, _, err := WorkloadFromArgs(mapper, ns, args)
			if err != nil {
				return err
			}

			caReader := configauditreport.NewReadWriter(starboardClientset)
			vulnsReader := vulnerabilityreport.NewReadWriter(starboardClientset)

			return report.NewHTMLReporter(caReader, vulnsReader).
				GenerateReport(workload, os.Stdout)
		},
	}

	return cmd
}
