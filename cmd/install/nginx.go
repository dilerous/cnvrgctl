/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package install

import (
	"context"
	"fmt"
	"log"
	"os"

	root "github.com/dilerous/cnvrgctl/cmd"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// nginxCmd represents the nginx command
var nginxCmd = &cobra.Command{
	Use:   "nginx",
	Short: "Installs the ingress controller, ingress-nginx",
	Long: `Install the ingress controller nginx as an application in ArgoCD. The deployment
will use the ingress-nginx helm chart.

Usage:
  cnvrgctl install nginx [flags]

Examples:
# Install nginx into the nginx namespace with the default values.
  cnvrgctl -n nginx install nginx
  
# Install argocd and specify a custom chart URL.
  cnvrgctl -n nginx install nginx --repo  https://github.com/nginx-helm
  
 # Install argocd and specify a helm release name.
  cnvrgctl -n nginx install nginx --release my-nginx`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("nginx called")

		// call the Flags structs
		flags := root.Flags{}

		// grab the namespace from the -n flag if not specified default is used
		ns, _ := cmd.Flags().GetString("namespace")

		// Flag to set the chart repo url
		flags.Repo, _ = cmd.Flags().GetString("repo")

		// Flag to set the chart name
		flags.ChartName, _ = cmd.Flags().GetString("chart")

		// Flag to set the helm release name for the install
		flags.ReleaseName, _ = cmd.Flags().GetString("release")

		// Flag to set the target version of the minio deployment
		flags.TargetRevision, _ = cmd.Flags().GetString("target-version")

		// Flag to set an external IP
		flags.ExternalIps, _ = cmd.Flags().GetString("external-ips")

		// Flag to set an external IP
		flags.App, _ = cmd.Flags().GetBool("app")

		// create the default values file
		vals, _ := createNginxValues(flags)

		// connect to kubernetes and get the client and rest api
		api, err := root.ConnectToK8s()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error connecting to kubernetes, check your connectivity. %v", err)
			log.Fatalf("error connecting to kubernetes, check your connectivity. %v", err)
		}
		// create the Minio Application file for argocd if flag set to true
		if flags.App {
			err = createNginxApp(api, ns, flags)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error applying the minio operator application custom resource. %v", err)
				log.Fatalf("error applying the minio operator application custom resource. %v", err)
			}
		}

		// Load the helm chart from the chartURL
		chart, err := loadChart(flags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading the chart, check the url and path. %v", err)
			log.Fatalf("error loading the chart, check the url and path. %v", err)
		}

		// Install the helm chart, specifiy namespace and the release name for the install
		err = deployHelmChart(ns, chart, flags, vals)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error installing the helm release, check the logs. %v", err)
			log.Fatalf("error installing the helm release, check the logs. %v", err)
		}

	},
}

func init() {
	installCmd.AddCommand(nginxCmd)

	// flag to define the path to the chart repo
	nginxCmd.Flags().StringP("repo", "", "https://kubernetes.github.io/ingress-nginx", "define the chart repository url")

	// flag to define the repository chart name
	nginxCmd.Flags().StringP("chart", "", "ingress-nginx", "specify the chart name in the repository defined.")

	// flag to define the release name
	nginxCmd.Flags().StringP("release", "", "nginx", "define the nginx helm release name.")

	// flag to define the helm chart version
	nginxCmd.Flags().StringP("target-version", "t", "4.10.1", "define the helm chart version.")

	// flag to define the externalIps
	nginxCmd.Flags().StringP("external-ips", "e", "", "define an external IP.")

	// flag to define the externalIps
	nginxCmd.Flags().BoolP("app", "", false, "install nginx as an application in ArgoCD.")

	// flag to define the namespace argocd is deployed
	nginxCmd.Flags().StringP("argocd-namespace", "a", "argocd", "define the namespace for the argocd deployment.")
}

// apply the ingress nginx application yaml
func createNginxApp(api *root.KubernetesAPI, ns string, f root.Flags) error {
	log.Println("createMinioValues function called")

	// define the application yaml
	app := &unstructured.Unstructured{}
	app.Object = map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]interface{}{
			"name":      f.ReleaseName,
			"namespace": "argocd",
		},

		"spec": map[string]interface{}{
			"project": "default",
			"source": map[string]interface{}{
				"chart":          f.ChartName,
				"repoURL":        f.Repo,
				"targetRevision": f.TargetRevision,
				"helm": map[string]interface{}{
					"releaseName": f.ReleaseName,
					"valuesObject": map[string]interface{}{
						"controller": map[string]interface{}{
							"service": map[string]interface{}{
								"externalIPs": []string{f.ExternalIps},
							},
							"config": map[string]interface{}{
								"proxy-body-size": "128m",
							},
							"ingressClassResource": map[string]interface{}{
								"controller.ingressClassResource.default": true,
							},
						},
					},
				},
			},
			"destination": map[string]interface{}{
				"server":    "https://kubernetes.default.svc",
				"namespace": ns,
			},
			"syncPolicy": map[string]interface{}{
				"automated":   map[string]interface{}{},
				"syncOptions": []interface{}{"CreateNamespace=true"},
			},
		},
	}

	// define the custom resource schema
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	// apply the minio-operator yaml file app
	err := api.Rest.Create(context.Background(), app)
	if err != nil {
		fmt.Println("error creating the nginx application", err)
		log.Fatalf("error creating the nginx application. %v", err)
		errorResponse(err)
	} else {
		fmt.Println("created the nginx application.")
		log.Println("created the nginx application")
	}

	return nil
}

// create the values for the helm install
func createNginxValues(f root.Flags) (map[string]interface{}, error) {

	// define default values
	vals := map[string]interface{}{

		"controller": map[string]interface{}{
			"service": map[string]interface{}{
				"externalIPs": []string{f.ExternalIps},
			},
			"config": map[string]interface{}{
				"proxy-body-size": "128m",
			},
			"ingressClassResource": map[string]interface{}{
				"default": true,
			},
		},
	}

	//return the values file and no error
	return vals, nil
}
