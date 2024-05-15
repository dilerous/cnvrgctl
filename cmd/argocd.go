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

var Scheme = runtime.NewScheme()
var settings = cli.New()

// argocdCmd represents the argocd command
var argocdCmd = &cobra.Command{
	Use:   "argocd",
	Short: "Installs the gitops tool, argocd",
	Long: `Install the gitops tool argocd as a helm release.  

Usage:
  cnvrgctl install argocd [flags]

Examples:
# Install argocd into the argocd namespace with the default values.
  cnvrgctl -n argocd install argocd
  
# Perform a dry run install of argocd.
  cnvrgctl -n argocd install argocd --dry-run
  
# Install argocd nd specify a custom chart URL .
  cnvrgctl -n argocd install argocd --repo  https://github.com/argo-helm`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("called the install argocd command function")

		// grab the namespace from the -n flag if not specified default is used
		ns, _ := cmd.Flags().GetString("namespace")

		// Flag to set the chart repo url
		chartURLFlag, _ := cmd.Flags().GetString("repo")

		// Flag to set the helm release name for the install
		chartNameFlag, _ := cmd.Flags().GetString("chart-name")

		// Flag to set the helm release name for the install
		releaseNameFlag, _ := cmd.Flags().GetString("release-name")

		//TODO: add ability to define values file
		//ADDED under string heading of values

		// Flag to set if dry-run should be ran for the install
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Load the helm chart from the chartURL
		chart, err := loadChart(chartURLFlag, chartNameFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading the chart, check the url and path. %v", err)
			log.Fatalf("error loading the chart, check the url and path. %v", err)
		}

		// Install the helm chart, specifiy namespace and the release name for the install
		err = deployHelmChart(ns, chart, releaseNameFlag, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error installing the helm release, check the logs. %v", err)
			log.Fatalf("error installing the helm release, check the logs. %v", err)
		}
	},
}

func init() {
	installCmd.AddCommand(argocdCmd)

	// flag to define the path to the chart repo
	argocdCmd.PersistentFlags().StringP("repo", "", "https://argoproj.github.io/argo-helm", "define the chart repository url")

	// flag to define the repository chart name
	argocdCmd.PersistentFlags().StringP("chart-name", "", "argo-cd", "specify the chart name in the repository defined.")

	// flag to define the release name
	argocdCmd.PersistentFlags().StringP("release-name", "", "argocd", "define the argocd helm release name.")

	// flag to define the values file for the install
	argocdCmd.PersistentFlags().StringP("values", "f", "", "specify values in a YAML file or a URL.")

	// Adds the flag -t --tar to the logs command this is local
	argocdCmd.Flags().BoolP("dry-run", "", false, "Perform a dry run of the install of argocd.")
}

func deployHelmChart(ns string, c *chart.Chart, release string, dryRun bool) error {

	// Define namespace, release name and chart to deploy the helm release
	var (
		namespace   = ns
		releaseName = release
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
	if dryRun {
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
	upgradeClient := action.NewUpgrade(actionConfig)
	upgradeClient.Install = true
	upgradeClient.Namespace = namespace

	// get the values for the helm install
	vals, err := createValues()
	if err != nil {
		log.Printf("error getting the values. %v", err)
		return fmt.Errorf("error getting the values. %w", err)
	}

	// install the chart here
	rel, err := client.Run(chart, vals)
	if err != nil {
		log.Printf("error installing the chart. %v", err)
		return fmt.Errorf("error installing the chart. %v", err)
	}

	//log and print install was successful
	fmt.Printf("Installed Chart from path: %s in namespace: %s\n", rel.Name, rel.Namespace)
	log.Printf("Installed Chart from path: %s in namespace: %s\n", rel.Name, rel.Namespace)

	// this will confirm the values set during installation
	fmt.Println(rel.Info)
	log.Println(rel.Info)
	return nil
}

func loadChart(url string, chartName string) (*chart.Chart, error) {

	// Set path to helm repository and chart name

	var chartPath = url

	// Define the chart options, set the repo url
	var chartPathOptions action.ChartPathOptions = action.ChartPathOptions{
		RepoURL: chartPath,
	}
	// locate the chart from the chartPathOptions
	locateChartPath, err := chartPathOptions.LocateChart(chartName, settings)
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

func createValues() (map[string]interface{}, error) {

	// define values
	vals := map[string]interface{}{
		"redis": map[string]interface{}{
			"sentinel": map[string]interface{}{
				"masterName": "BigMaster",
				"pass":       "random",
				"addr":       "localhost",
				"port":       "26379",
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
		log.Fatalf("error getting namespace. %v", err)
		return false, fmt.Errorf("Namespace %s does not exist: %w\n", namespace, err)
	} else {
		// If no error is returned, the namespace exists
		fmt.Printf("Namespace %s exists\n", namespace)
		log.Printf("Namespace %s exists\n", namespace)
		return true, nil

	}
}
