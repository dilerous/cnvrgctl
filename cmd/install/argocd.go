/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package install

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	root "github.com/dilerous/cnvrgctl/cmd"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	Scheme   = runtime.NewScheme()
	settings = cli.New()
)

// argocdCmd represents the argocd command
var argocdCmd = &cobra.Command{
	Use:   "argocd",
	Short: "Installs the gitops tool, argocd",
	Long: `Install the gitops tool argocd as a helm release. The deployment
will use the default ingress controller for external access.

Usage:
  cnvrgctl install argocd [flags]

Examples:
# Install argocd into the argocd namespace with the default values.
  cnvrgctl -n argocd install argocd -n argocd

# Perform a dry run install of argocd.
  cnvrgctl -n argocd install argocd --dry-run

# Install argocd and specify a custom chart URL.
  cnvrgctl -n argocd install argocd --repo  https://github.com/argo-helm

# Install with a user specific domain.
  cnvrgctl -n argocd install argocd --domain argocd.dilerous.cloud
  
# Install with a tls enabled, the secret that will be used is argo-ui-tls.
  cnvrgctl -n argocd install argocd -d argocd.dilerous.cloud --enable-tls
  
# Install with a tls enabled, and a cert-manager cluserIssuer defined.
  cnvrgctl -n argocd install argocd -d argocd.dilerous.cloud --enable-tls --cluster-issuer letsencrypt-prod`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("called the install argocd command function")

		// define the Flags struct
		flags := root.Flags{}

		// grab the namespace from the -n flag if not specified default is used
		ns, _ := cmd.Flags().GetString("namespace")

		// Flag to set the chart repo url
		flags.Repo, _ = cmd.Flags().GetString("repo")

		// Flag to set the chart name for the install
		flags.ChartName, _ = cmd.Flags().GetString("chart-name")

		// Flag to set the helm release name for the install
		flags.ReleaseName, _ = cmd.Flags().GetString("release-name")

		// Flag to set the domain of the argocd deployment
		flags.Domain, _ = cmd.Flags().GetString("domain")

		// Flag to enable tls support for the ingress object
		flags.Tls, _ = cmd.Flags().GetBool("enable-tls")

		// Flag to define the cert-manager cluster-issuer
		flags.ClusterIssuer, _ = cmd.Flags().GetString("cluster-issuer")

		// Flag to set if dry-run should be ran for the install
		flags.DryRun, _ = cmd.Flags().GetBool("dry-run")

		// Load the helm chart from the chartURL
		chart, err := loadChart(flags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading the chart, check the url and path. %v", err)
			log.Fatalf("error loading the chart, check the url and path. %v", err)
		}

		// create the default values file
		vals, _ := createValues(&flags)

		// Install the helm chart, specifiy namespace and the release name for the install
		err = deployHelmChart(ns, chart, flags, vals)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error installing the helm release, check the logs. %v", err)
			log.Fatalf("error installing the helm release, check the logs. %v", err)
		}
	},
}

func init() {
	installCmd.AddCommand(argocdCmd)

	// flag to define the path to the chart repo
	argocdCmd.Flags().StringP("repo", "", "https://argoproj.github.io/argo-helm", "define the chart repository url")

	// flag to define the repository chart name
	argocdCmd.Flags().StringP("chart-name", "", "argo-cd", "specify the chart name in the repository defined.")

	// flag to define the release name
	argocdCmd.Flags().StringP("release-name", "", "argocd", "define the argocd helm release name.")

	// flag to define the values file for the install
	argocdCmd.Flags().StringP("values", "f", "", "specify values in a YAML file or a URL.")

	// flag to define the argocd domain for install
	argocdCmd.Flags().StringP("domain", "d", "argocd.example.com", "define the domain for the argocd deployment.")

	// Flag to enable tls support for the ingress object
	argocdCmd.Flags().Bool("enable-tls", false, "enable tls for the ingress object.")

	// Flag to define the cert-manager cluster issuer
	argocdCmd.Flags().StringP("cluster-issuer", "i", "", "define a cert-manager clusterIssuer.")

	// Adds the flag -t --tar to the logs command this is local
	argocdCmd.Flags().BoolP("dry-run", "", false, "Perform a dry run of the install of argocd.")
}

func createValues(f *root.Flags) (map[string]interface{}, error) {
	log.Println("called the createValues function")

	// define the default values
	vals := map[string]interface{}{
		"global": map[string]interface{}{
			"domain": f.Domain,
		},

		"server": map[string]interface{}{
			"ingress": map[string]interface{}{
				"enabled": true,
				"annotations": map[string]interface{}{
					"nginx.ingress.kubernetes.io/backend-protocol":   "HTTPS",
					"nginx.ingress.kubernetes.io/force-ssl-redirect": "true",
					"nginx.ingress.kubernetes.io/ssl-passthrough":    "true",
				},
			},
		},
		"configs": map[string]interface{}{
			"params": map[string]interface{}{
				"server": map[string]interface{}{
					"insecure": true,
				},
			},
		},
	}

	// check if the enable-tls flag has been set to true
	if f.Tls {
		// Update insecure to false
		vals["configs"].(map[string]interface{})["params"].(map[string]interface{})["server"].(map[string]interface{})["insecure"] = false

		// Create the TLS configuration
		tls := []map[string]interface{}{
			{
				"secretName": "argo-ui-tls",
				"hosts": []string{
					f.Domain,
				},
			},
		}
		// Add TLS to vals
		vals["server"].(map[string]interface{})["ingress"].(map[string]interface{})["tls"] = tls
	}

	if f.ClusterIssuer != "" {
		//add in the annotation for clusterIssuer/issuer
		newAnnotationKey := "cert-manager.io/cluster-issuer"
		newAnnotationValue := f.ClusterIssuer

		//Add the annotation for cert-manager
		vals["server"].(map[string]interface{})["ingress"].(map[string]interface{})["annotations"].(map[string]interface{})[newAnnotationKey] = newAnnotationValue
	}
	return vals, nil
}
