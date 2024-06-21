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

// sealedCmd represents the sealed command
var sealedCmd = &cobra.Command{
	Use:   "sealed",
	Short: "Installs sealed secrets as an ArgoCD application",
	Long: `This command will deploy Sealed Secrets as an application in ArgoCD.
The default install will enable ingress for external connectivity. 

Usage:
  cnvrgctl install sealed [flags]

Examples:
# Install Sealed Secrets into the sealed namespace with the default values.
  cnvrgctl -n sealed install sealed
 
# Install Sealed Secrets with the a domain defined for the ingress route.
  cnvrgctl -n sealed install sealed --domain sealed.example.com
  
# Install Sealed Secrets with the a target version and chart defined.
  cnvrgctl -n sealed install sealed --target-version 2.0.0  --chart sealed-secrets/sealed-secrets`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("sealed command called")

		// call the Flags structs
		flags := root.Flags{}

		// grab the namespace from the -n flag if not specified default is used
		ns, _ := cmd.Flags().GetString("namespace")

		// get the namespace argocd is installed into
		flags.Argocd, _ = cmd.Flags().GetString("argocd-namespace")

		// Flag to set the chart repo url
		flags.Repo, _ = cmd.Flags().GetString("repo")

		// Flag to set the chart name
		flags.ChartName, _ = cmd.Flags().GetString("chart")

		// Flag to set the helm release name for the install
		flags.ReleaseName, _ = cmd.Flags().GetString("release")

		// Flag to set the target version of the minio deployment
		flags.TargetRevision, _ = cmd.Flags().GetString("target-version")

		// Flag to set the target version of the minio deployment
		flags.Domain, _ = cmd.Flags().GetString("domain")

		// connect to kubernetes and get the client and rest api
		api, err := root.ConnectToK8s()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error connecting to kubernetes, check your connectivity. %v", err)
			log.Fatalf("error connecting to kubernetes, check your connectivity. %v", err)
		}
		// create the Minio Application file for argocd if flag set to true
		err = createSealedApp(api, ns, flags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error applying the %s application custom resource. %v", flags.ReleaseName, err)
			log.Fatalf("error applying the %s application custom resource. %v", flags.ReleaseName, err)
		}
	},
}

func init() {
	installCmd.AddCommand(sealedCmd)

	// flag to define the path to the chart repo
	sealedCmd.Flags().StringP("repo", "", "https://bitnami-labs.github.io/sealed-secrets", "define the chart repository url")

	// flag to define the repository chart name
	sealedCmd.Flags().StringP("chart", "", "sealed-secrets", "specify the chart name in the repository defined.")

	// flag to define the release name
	sealedCmd.Flags().StringP("release", "", "sealed", "define the sealed secrets helm release name.")

	// flag to define the helm chart version
	sealedCmd.Flags().StringP("target-version", "t", "2.15.4", "define the helm chart version.")

	// flag to define the argocd domain for install
	sealedCmd.Flags().StringP("domain", "d", "sealed-secrets.local", "define the domain for the sealed secrets deployment.")

	// flag to define the argocd namespace for install
	sealedCmd.Flags().StringP("argocd-namespace", "a", "argocd", "define the namespace for the argocd deployment.")
}

// apply the ingress sealed secret application yaml
func createSealedApp(api *root.KubernetesAPI, ns string, f root.Flags) error {
	log.Println("createMinioValues function called")

	// define the application yaml
	app := &unstructured.Unstructured{}
	app.Object = map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]interface{}{
			"name":      f.ReleaseName,
			"namespace": f.Argocd,
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
						"ingress": map[string]interface{}{
							"enabled":  true,
							"hostname": f.Domain,
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
		fmt.Printf("error creating the %s application. %v\n", f.ReleaseName, err)
		log.Fatalf("error creating the %s application. %v", f.ReleaseName, err)
		return fmt.Errorf("error creating the %s application. %w", f.ReleaseName, err)
	} else {
		fmt.Printf("created the %s application.\n", f.ReleaseName)
		log.Printf("created the %s application.\n", f.ReleaseName)
		fmt.Println("to get the application status run:")
		fmt.Println("kubectl get application -A -o wide")
		return nil
	}
}
