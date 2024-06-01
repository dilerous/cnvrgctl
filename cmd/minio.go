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
	Long: `Deploy the minio operator and a minio tenant can be used to store backups
during the migration process. The installation by default will deploy an ingress resource,
to access the minio operator ui, the minio tenant api, and tenant ui. Please see examples
below to define the domain and other configurations.

Usage:
  cnvrgctl install minio [flags]

Examples:
# Install minio tenant and operator into the minio-operator namespace with a custom domain defined.
  cnvrgctl install minio -n minio-operator -d minio-console.nginx.dilerous.cloud

# Intall minio with a different helm release name and repo defined.
cnvrgctl install minio -n minio-operator --repo http://example.minio.com --op-release operator --tenant-release minio
`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("minio command called")

		// call the Flags structs
		opFlags := Flags{}
		tenantFlags := Flags{}

		// grab the namespace from the -n flag if not specified default is used
		ns, _ := cmd.Flags().GetString("namespace")

		// Flag to set the chart repo url
		opFlags.Repo, _ = cmd.Flags().GetString("repo")
		tenantFlags.Repo, _ = cmd.Flags().GetString("repo")

		// Flag to set the chart repo url
		opFlags.ChartName, _ = cmd.Flags().GetString("op-chart")

		// Flag to set the helm release name for the install
		opFlags.ReleaseName, _ = cmd.Flags().GetString("op-release")

		// Flag to set the domain of the argocd deployment
		opFlags.Domain, _ = cmd.Flags().GetString("op-url")

		// Flag to set the domain of the argocd deployment
		opFlags.TargetRevision, _ = cmd.Flags().GetString("op-target-version")

		// Flag to set the chart repo url
		tenantFlags.ChartName, _ = cmd.Flags().GetString("tenant-chart")

		// Flag to set the helm release name for the install
		tenantFlags.ReleaseName, _ = cmd.Flags().GetString("tenant-release")

		// Flag to set the domain of the argocd deployment
		tenantFlags.Domain, _ = cmd.Flags().GetString("tenant-url")

		// Flag to set the domain of the argocd deployment
		tenantFlags.TargetRevision, _ = cmd.Flags().GetString("tenant-target-version")

		api, err := connectToK8s()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error connecting to kubernetes, check your connectivity. %v", err)
			log.Fatalf("error connecting to kubernetes, check your connectivity. %v", err)
		}

		// create the Minio Application file for argocd
		err = createMinioOperatorApp(api, ns, opFlags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error applying the minio operator application custom resource. %v", err)
			log.Fatalf("error applying the minio operator application custom resource. %v", err)
		}

		// create the Minio Application file for argocd
		err = createMinioTenantApp(api, ns, tenantFlags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error applying the minio tenant application custom resource. %v", err)
			log.Fatalf("error applying the minio tenant application custom resource. %v", err)
		}

	},
}

// initializes the flags and the install command
func init() {
	installCmd.AddCommand(minioCmd)

	// flag to define the path to the chart repo
	minioCmd.Flags().StringP("repo", "", "https://operator.min.io", "define the chart repository url")

	// flag to define the repository chart name
	minioCmd.Flags().StringP("op-chart", "", "operator", "specify the chart name in the repository defined.")

	// flag to define the release name
	minioCmd.Flags().StringP("op-release", "", "minio-operator", "define the argocd helm release name.")

	// flag to define the operator url for ingress
	minioCmd.Flags().StringP("op-url", "", "minio-operator.example.com", "define the url for the operator deployment.")

	// flag to define the helm chart version
	minioCmd.Flags().StringP("op-target-version", "", "v5.0.15", "define the helm chart version.")

	// flag to define the tenant repository chart name
	minioCmd.Flags().StringP("tenant-chart", "", "tenant", "specify the chart name in the repository defined.")

	// flag to define the tenant release name
	minioCmd.Flags().StringP("tenant-release", "", "minio-tenant", "define the argocd helm release name.")

	// flag to define the tenant url for ingress
	minioCmd.Flags().StringP("tenant-url", "", "minio.example.com", "define the url for the tenant deployment.")

	// flag to define the tenentant helm chart version
	minioCmd.Flags().StringP("tenant-target-version", "", "v5.0.15", "define the helm chart version for the tenant.")

}

// apply the minio operator application yaml
func createMinioOperatorApp(api *KubernetesAPI, ns string, f Flags) error {
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
						"console": map[string]interface{}{
							"ingress": map[string]interface{}{
								"enabled": true,
								"host":    f.Domain,
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
		fmt.Println("error creating the minio-operator application", err)
		log.Fatalf("error creating the minio-operator application. %v", err)
		errorResponse(err)
	} else {
		fmt.Println("created the minio-operator application.")
		log.Println("created the minio-operator application")
	}

	return nil
}

// TODO create a struct for all of the flags being passed
// apply the minio operator application yaml
func createMinioTenantApp(api *KubernetesAPI, ns string, f Flags) error {
	log.Println("createMinioTenantApp called")

	//TODO: add ability to dynamically grab the argocd namespace
	argoNamespace := "argocd"

	// define the application yaml
	app := &unstructured.Unstructured{}
	app.Object = map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]interface{}{
			"name":      f.ReleaseName,
			"namespace": argoNamespace,
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
						"tenant": map[string]interface{}{
							"name": "minio",
							"pools": []interface{}{
								map[string]interface{}{
									"servers": 1,
									"name":    "pool-0",
									"size":    "10Gi",
								},
							},
							"buckets": []interface{}{
								map[string]interface{}{
									"name": "cnvrg-backups",
								},
							},
						},
						"ingress": map[string]interface{}{
							"api": map[string]interface{}{
								"enabled": true,
								"host":    f.Domain,
								"annotations": map[string]interface{}{
									"nginx.ingress.kubernetes.io/force-ssl-redirect": "false",
									"nginx.ingress.kubernetes.io/ssl-passthrough":    "true",
									"nginx.ingress.kubernetes.io/backend-protocol":   "HTTPS",
								},
							},
							"console": map[string]interface{}{
								"enabled": true,
								"host":    "console-" + f.Domain,
								"annotations": map[string]interface{}{
									"nginx.ingress.kubernetes.io/force-ssl-redirect": "true",
									"nginx.ingress.kubernetes.io/ssl-passthrough":    "true",
									"nginx.ingress.kubernetes.io/backend-protocol":   "HTTPS",
								},
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
	err := api.Rest.Create(context.TODO(), app)

	if err != nil {
		fmt.Println("error creating the minio-tenant application.", err)
		//updateMinioOperator(api, ns)
		errorResponse(err)
	} else {
		fmt.Println("created the minio-tenant application.")
		log.Println("created the minio-tenance application")
	}

	return nil
}

func errorResponse(err error) {
	// Check if the erros match, if the do call the function to display the warning
	tenantError := "applications.argoproj.io " + `"` + "minio-tenant" + `"` + " already exists"
	if err.Error() == tenantError {
		fmt.Printf("updates to the minio-tenant deployment should be done using argocd.\n")
		log.Fatalf("updates to the minio-tenant deployment should be done using argocd.\n%v", err)
	}

	operatorError := "applications.argoproj.io " + `"` + "minio-operator" + `"` + " already exists"
	if err.Error() == operatorError {
		fmt.Printf("updates to the minio-operator deployment should be done using argocd.\n")
		log.Fatalf("updates to the minio-operator deployment should be done using argocd.\n%v", err)
	}
}

/*
// TODO: could never get update to work - needs to be fixed
func updateMinioOperator(api *KubernetesAPI, ns string) {

	app := &unstructured.Unstructured{}
	// define the custom resource schema
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	err := api.Rest.Get(context.TODO(), client.ObjectKey{
		Namespace: "argocd",
		Name:      "minio-tenant"}, app)

	if err != nil {
		fmt.Println("error getting the custom resource. %w", err)
		log.Fatalf("error getting the custom resource. %v", err)
	}

	// Update the custom resource
	app.SetUnstructuredContent(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				"updateby/cnvrgctl": "true",
			},
		},
	})

	fmt.Println(ns)
	fmt.Println(app)

	fmt.Println("running update on the application resource.")
	err = api.Rest.Update(context.TODO(), app, &client.UpdateOptions{})
	if err != nil {
		fmt.Println("error getting the custom resource. %w", err)
	}
}
*/

/*
// Retrieve the existing custom resource object
	customResource, err := api.Rest.
		Dynamic().
		Resource(app).
		Namespace(ns).
		Get(context.Background(), "ResourceName", v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Error getting custom resource:", err)
	}
*/
