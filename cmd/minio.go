/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// minioCmd represents the minio command
var minioCmd = &cobra.Command{
	Use:   "minio",
	Short: "deploys the minio operator and 1 tenant",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("minio called")

		// grab the namespace from the -n flag if not specified default is used
		ns, _ := cmd.Flags().GetString("namespace")

		// Flag to set the chart repo url
		chartURLFlag, _ := cmd.Flags().GetString("repo")

		// Flag to set the helm release name for the install
		releaseNameFlag, _ := cmd.Flags().GetString("release-name")

		// Flag to set the domain of the argocd deployment
		domainFlag, _ := cmd.Flags().GetString("domain")

		api, err := connectToK8s()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error connecting to kubernetes, check your connectivity. %v", err)
			log.Fatalf("error connecting to kubernetes, check your connectivity. %v", err)
		}

		// create the Minio Application file for argocd
		err = createMinioOperatorApp(api, ns, chartURLFlag, releaseNameFlag, domainFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error applying the minio operator application custom resource. %v", err)
			log.Fatalf("error applying the minio operator application custom resource. %v", err)
		}

	},
}

// initializes the flags and the install command
func init() {
	installCmd.AddCommand(minioCmd)

	// flag to define the path to the chart repo
	minioCmd.Flags().StringP("repo", "", "https://operator.min.io", "define the chart repository url")

	// flag to define the repository chart name
	minioCmd.Flags().StringP("chart-name", "", "operator", "specify the chart name in the repository defined.")

	// flag to define the release name
	minioCmd.Flags().StringP("release-name", "", "minio-operator", "define the argocd helm release name.")

	// flag to define the argocd domain for install
	minioCmd.Flags().StringP("domain", "d", "minio-operator.example.com", "define the domain for the argocd deployment.")

}

// TODO create a struct for all of the flags being passed
// apply the minio operator application yaml
func createMinioOperatorApp(api *KubernetesAPI, ns string, repo string, release string, domain string) error {
	log.Println("createMinioValues called")

	// define the application yaml
	app := &unstructured.Unstructured{}
	app.Object = map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]interface{}{
			"name":      "minio-operator",
			"namespace": "argocd",
		},

		"spec": map[string]interface{}{
			"project": "default",
			"source": map[string]interface{}{
				"chart":          "operator",
				"repoURL":        repo,
				"targetRevision": "v5.0.15",
				"helm": map[string]interface{}{
					"releaseName": release,
					"valuesObject": map[string]interface{}{
						"console": map[string]interface{}{
							"ingress": map[string]interface{}{
								"enabled": true,
								"host":    domain,
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
				"automated": map[string]interface{}{
					"syncOptions": map[string]interface{}{
						"CreateNamespace": "true",
					},
				},
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
		fmt.Println("error create the minio-operator application", "ERR", err)
	} else {
		fmt.Println("created the minio-operator application.", api.Rest)
	}

	return nil
}

// Here you will define your flags and configuration settings.

// Cobra supports Persistent Flags which will work for this command
// and all subcommands, e.g.:
// minioCmd.PersistentFlags().String("foo", "", "A help for foo")

// Cobra supports local flags which will only run when this command
// is called directly, e.g.:
// minioCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
