/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
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
  cnvrgctl -n argocd install argocd
  
# Perform a dry run install of argocd.
  cnvrgctl -n argocd install argocd --dry-run
  
# Install argocd and specify a custom chart URL.
  cnvrgctl -n argocd install argocd --repo  https://github.com/argo-helm

# Install with a user specific domain.
  cnvrgctl -n argocd install argocd -d argocd.dilerous.cloud`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("called the install argocd command function")

		flags := Flags{}

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

		// Flag to set if dry-run should be ran for the install
		flags.DryRun, _ = cmd.Flags().GetBool("dry-run")

		// Load the helm chart from the chartURL
		chart, err := loadChart(flags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading the chart, check the url and path. %v", err)
			log.Fatalf("error loading the chart, check the url and path. %v", err)
		}

		vals, _ := createValues(flags.Domain)

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

	// Adds the flag -t --tar to the logs command this is local
	argocdCmd.Flags().BoolP("dry-run", "", false, "Perform a dry run of the install of argocd.")
}

func deployHelmChart(ns string, c *chart.Chart, f Flags, vals map[string]interface{}) error {

	// Define namespace, release name and chart to deploy the helm release
	var (
		namespace   = ns
		releaseName = f.ReleaseName
		chart       = c
	)

	actionConfig := new(action.Configuration)
	// You can pass an empty string instead of settings.Namespace() to list
	// all namespaces
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace,
		os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		log.Printf("%+v", err)
	}

	// create a new actiom for the helm install
	client := action.NewInstall(actionConfig)
	client.Namespace = namespace
	client.ReleaseName = releaseName

	// if dry run is enabled from cli set client to run --dry-run on install
	if f.DryRun {
		client.DryRun = true
	}

	// check if the namespace exists
	// connect to k8s to query if namespace exists
	api, err := connectToK8s()
	if err != nil {
		log.Fatalf("error connecting to the cluster, check your connectivity. %v", err)
		return fmt.Errorf("error connecting to the cluster, check your connectivity. %w", err)
	}

	// check if the namespace exists
	exists, err := checkNamespaceExists(client.Namespace, api.Client)
	if err != nil {
		log.Printf("error checking if the namespace exists. %v", err)
		return fmt.Errorf("error checking if the namespace exists. %w", err)
	}

	// create the namespace if the namespace doesn't exist
	if !exists {
		client.CreateNamespace = true
	}

	//TODO: add ability to upgrade
	//TODO: add flag to specify upgrade
	upgradeClient := action.NewUpgrade(actionConfig)
	upgradeClient.Install = true
	upgradeClient.Namespace = namespace

	// install the chart here
	fmt.Println("Installing ArgoCD please wait ...")
	rel, err := client.Run(chart, vals)
	if err != nil {
		log.Printf("error installing the chart. %v", err)
		return fmt.Errorf("error installing the chart. %v", err)
	}

	//log and print install was successful
	fmt.Printf("installed Chart from path: %s in namespace: %s\n", rel.Name, rel.Namespace)
	log.Printf("installed Chart from path: %s in namespace: %s\n", rel.Name, rel.Namespace)

	// this will print the release info after the install
	fmt.Println(rel.Info)
	log.Println(rel.Info)
	return nil
}

func loadChart(f Flags) (*chart.Chart, error) {

	// Set path to helm repository and chart name

	var chartPath = f.Repo

	// Define the chart options, set the repo url
	var chartPathOptions action.ChartPathOptions = action.ChartPathOptions{
		RepoURL: chartPath,
	}
	// locate the chart from the chartPathOptions
	locateChartPath, err := chartPathOptions.LocateChart(f.ChartName, settings)
	if err != nil {
		log.Printf("error locating the chart. %v", err)
		return nil, fmt.Errorf("error locating the chart. %w", err)
	}

	// load chart from the chart path
	chart, err := loader.Load(locateChartPath)
	if err != nil {
		log.Printf("error loading the chart. %v", err)
		return nil, fmt.Errorf("error loading the chart. %w", err)
	}
	return chart, nil
}

func createValues(domain string) (map[string]interface{}, error) {

	// define values

	vals := map[string]interface{}{
		"global": map[string]interface{}{
			"domain": domain,
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
	return vals, nil
}

// Check if the namespace exists
func checkNamespaceExists(ns string, clientset kubernetes.Interface) (bool, error) {
	// Specify the namespace you want to check
	var namespace = ns

	//TODO: play with adding cts to the KubernetesAPI struct
	// Attempt to get the namespace object
	_, err := clientset.CoreV1().Namespaces().Get(context.Background(), namespace, v1.GetOptions{})
	if err != nil {
		// If the namespace does not exist, an error will be returned
		fmt.Printf("the namespace %v, doesn't exist.\n", namespace)
		log.Printf("the namespace %v, doesn't exist.\n", namespace)
		return false, nil
	} else {
		// If no error is returned, the namespace exists
		fmt.Printf("namespace %s exists\n", namespace)
		log.Printf("namespace %s exists\n", namespace)
		return true, nil
	}
}
