/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Command to manage cnvrg.io installation migrations",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("migrate called")
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// migrateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// migrateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// get the pod name from the deployment this will be passed to executeBackup function
// used by back and restore commands
func getDeployPod(api *KubernetesAPI, targetFlag string, nsFlag string, labelTag string) (string, error) {
	log.Println("getDeployPod function called.")

	var (
		// set the clientset, namespace, deployment name and label key
		clientset  = api.Client
		namespace  = nsFlag
		deployName = targetFlag
		label      = labelTag
	)

	// Get the Pods associated with the deployment
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), v1.ListOptions{
		LabelSelector: labels.Set{label: deployName}.AsSelector().String(),
	})
	if err != nil {
		fmt.Printf("no pods found for the deployment. %v\n", err)
		return "", fmt.Errorf("no pods found for this deployment %w", err)
	}

	// check if there is any pods in the list
	if len(pods.Items) == 0 {
		log.Println("there are no pods. check you have the correct namespace and the deployment exists.")
		return "", fmt.Errorf("there are no pods. check you have the correct namespace and the deployment exists. %w", err)
	}

	// grab the first pod name in the list
	podName := pods.Items[0].Name
	return podName, nil
}

func scaleDeployUp(api *KubernetesAPI, ns string) error {
	log.Println("scaleDeployUp function called.")

	var (
		// Set client, deployment names and namespace
		clientset   = api.Client
		deployNames = []string{"app", "sidekiq", "systemkiq", "searchkiq", "cnvrg-operator"}
		namespace   = ns
	)

	// Get the deployment
	for _, deployName := range deployNames {

		// Get the current number of replicas for the deployment
		s, err := clientset.AppsV1().Deployments(namespace).GetScale(context.Background(), deployName, v1.GetOptions{})
		if err != nil {
			fmt.Printf("there was an error getting the number of replicas for deployment %v, check the namespace specified is correct.\n %v", deployName, err)
			return fmt.Errorf("there was an error getting the number of replicas for deployment %v, check the namespace specified is correct. %w", deployName, err)
		}

		// create a v1.Scale object and set the replicas to 0
		sc := *s
		sc.Spec.Replicas = 1

		// Scale the deployment to 0
		scale, err := clientset.AppsV1().Deployments(namespace).UpdateScale(context.Background(), deployName, &sc, v1.UpdateOptions{})
		if err != nil {
			fmt.Printf("there was an issue scaling the deployment %v.\n%v", deployName, err)
			return fmt.Errorf("there was an issue scaling the deployment %v. %w", deployName, err)
		}

		// Print to screen the deployments scaled to 0
		fmt.Printf("scaled deployment %s to %d replica(s).\n", scale.Name, sc.Spec.Replicas)
		//TODO: add check for num of replicas = 0

	}
	//TODO: add a check if all replicas = 0
	fmt.Println("scaled deployments back to 1 replica(s)...")
	return nil
}

// scales the following deployments "app", "sidekiq", "systemkiq", "searchkiq", "cnvrg-operator" in the namespace specified
// used in back and restore commands
func scaleDeployDown(api *KubernetesAPI, ns string) error {
	log.Println("scaleDeployDown function called.")

	var (
		// Set client, deployment names and namespace
		clientset   = api.Client
		deployNames = []string{"app", "sidekiq", "systemkiq", "searchkiq", "cnvrg-operator"}
		namespace   = ns
	)

	// Get the deployment
	for _, deployName := range deployNames {

		// Get the current number of replicas for the deployment
		s, err := clientset.AppsV1().Deployments(namespace).GetScale(context.Background(), deployName, v1.GetOptions{})
		if err != nil {
			fmt.Printf("there was an error getting the number of replicas for deployment %v, check the namespace specified is correct.\n %v", deployName, err)
			return fmt.Errorf("there was an error getting the number of replicas for deployment %v, check the namespace specified is correct. %w", deployName, err)
		}

		// create a v1.Scale object and set the replicas to 0
		sc := *s
		sc.Spec.Replicas = 0

		// Scale the deployment to 0
		scale, err := clientset.AppsV1().Deployments(namespace).UpdateScale(context.Background(), deployName, &sc, v1.UpdateOptions{})
		if err != nil {
			fmt.Printf("there was an issue scaling the deployment %v.\n%v", deployName, err)
			return fmt.Errorf("there was an issue scaling the deployment %v. %w", deployName, err)
		}

		// Print to screen the deployments scaled to 0
		fmt.Printf("scaled deployment %s to %d replica(s).\n", scale.Name, sc.Spec.Replicas)
		//TODO: add check for num of replicas = 0

	}
	//TODO: add a check if all replicas = 0
	fmt.Println("waiting for pods to finish terminating...")
	time.Sleep(10 * time.Second)
	return nil
}

// connect to minio storage
// TODO Create generic and move to migrate
// used by backup and restore commands
func connectToMinio(o *ObjectStorage) error {
	// Initialize a new MinIO client
	useSSL := false

	uWithoutHttp := strings.Replace(o.Endpoint, "http://", "", 1)
	log.Println(uWithoutHttp)

	minioClient, err := minio.New(uWithoutHttp, &minio.Options{
		Creds:  credentials.NewStaticV4(o.AccessKey, o.SecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Printf("error connecting to minio. %v", err)
		return fmt.Errorf("error connecting to minio. %w ", err)
	}

	// List buckets
	buckets, err := minioClient.ListBuckets(context.Background())
	if err != nil {
		log.Printf("error listing the buckets. %v", err)
		return fmt.Errorf("error listing the buckets. %w ", err)
	}

	// list the buckets that are found
	for _, bucket := range buckets {
		log.Println("Buckets: " + bucket.Name)
	}

	return nil
}
