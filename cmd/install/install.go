/*
Copyright © 2024 Brad Soper BRADLEY.SOPER@CNVRG.IO
*/
package install

import (
	"context"
	"fmt"
	"log"
	"os"

	root "github.com/dilerous/cnvrgctl/cmd"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install cnvrg and supporting applications.",
	Long: `Install ArgoCD to manage deployments of additional helm 
charts.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("install cmd called")
	},
}

func init() {
	root.RootCmd.AddCommand(installCmd)
}

func loadChart(f root.Flags) (*chart.Chart, error) {
	log.Println("calling the loadChart function")

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

// Deploy the helm chart defined with the values built
func deployHelmChart(ns string, c *chart.Chart, f root.Flags, vals map[string]interface{}) error {

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
	api, err := root.ConnectToK8s()
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
	fmt.Printf("Installing %s please wait...\n", releaseName)
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
