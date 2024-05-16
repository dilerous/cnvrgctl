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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Executes backup commands on the postgres pod",
	Long: `This command will initiate a pg_dump of the postgres database
and save it to a file in the current directory.

Examples:

# Backups the default postgres database in the cnvrg namespace.
  cnvrgctl migrate backup -n cnvrg

# Specify namespace, deployment label key, and deployment name.
  cnvrgctl migrate backup --target postgres-ha --label app.kubernetes.io/name -n cnvrg`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("called the migrate backup command function")

		// target deployment of the postgres backup
		targetFlag, _ := cmd.Flags().GetString("target")

		// grab the namespace from the -n flag if not specified default is used
		nsFlag, _ := cmd.Flags().GetString("namespace")

		// grab the namespace from the -n flag if not specified default is used
		labelFlag, _ := cmd.Flags().GetString("label")

		// connect to kubernetes and define clientset and rest client
		api, err := connectToK8s()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error connecting to the cluster, check your connectivity. %v", err)
			log.Fatalf("error connecting to the cluster, check your connectivity. %v", err)
		}

		err = scaleDeploy(api, nsFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error scaling the deployment. %v", err)
			log.Fatalf("error scaling the deployment. %v", err)
		}

		podName, err := getDeployPod(api, targetFlag, nsFlag, labelFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting the pod name check the deployment label, namespace and target. %v", err)
			log.Fatalf("error getting the pod name check the deployment label, namespace and target. %v", err)
		}

		// execute the backup of the target postgres deployment
		err = executeBackup(api, podName, nsFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error executing the backup, check the logs. %v", err)
			log.Fatalf("error executing the backup, check the logs. %v", err)
		}

	},
}

func init() {
	migrateCmd.AddCommand(backupCmd)

	// flag to define the release name
	backupCmd.PersistentFlags().StringP("target", "t", "postgres", "name of postgres deployment to backup.")

	// flag to define the app label key
	backupCmd.PersistentFlags().StringP("label", "l", "app", "modify the key of the deployment label. example: app.kubernetes.io/name")

}

func scaleDeploy(api *KubernetesAPI, nsFlag string) error {

	var (
		clientset = api.Client
		// Set the deployment name and namespace
		deployNames = []string{"app", "sidekiq", "systemkiq", "searchkiq", "cnvrg-operator"}
		namespace   = nsFlag
	)

	// Get the deployment
	for _, deployName := range deployNames {

		s, err := clientset.AppsV1().Deployments(nsFlag).GetScale(context.TODO(), deployName, v1.GetOptions{})
		if err != nil {
			fmt.Printf("there was an error getting the number of replicas for deployment %v, check the namespace specified is correct.\n %v", deployName, err)
			return fmt.Errorf("there was an error getting the number of replicas for deployment %v, check the namespace specified is correct. %w", deployName, err)
		}
		sc := *s
		sc.Spec.Replicas = 0

		// Scale the deployment to 0
		scale, err := clientset.AppsV1().Deployments(namespace).UpdateScale(context.TODO(), deployName, &sc, v1.UpdateOptions{})
		if err != nil {
			fmt.Printf("there was an issue scaling the deployment %v.\n%v", deployName, err)

			return fmt.Errorf("there was an issue scaling the deployment %v. %w", deployName, err)

		}
		fmt.Printf("Scaled deployment %s to 0 replicas.\n%v", scale, deployName)
	}
	return nil
}

// get the pod name from the deployment this will be passed to executeBackup function
func getDeployPod(api *KubernetesAPI, targetFlag string, nsFlag string, labelTag string) (string, error) {
	var (
		// set the clientset, namespace, deployment name and label key
		clientset  = api.Client
		namespace  = nsFlag
		deployName = targetFlag
		label      = labelTag
	)

	// Get the Pods associated with the deployment
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{
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

// Executes a pg dump of the postgres database by getting the postgres pod name then running
// pg_dump on the postgres pod
func executeBackup(api *KubernetesAPI, pod string, nsFlag string) error {

	// set variables for the clientset and pod name
	var (
		clientset = api.Client
		podName   = pod
		namespace = nsFlag
	)

	// this is the command passed when connecting to the pod
	command := []string{
		"sh",
		"-c",
		"echo $HOME; ls -l && echo hello",
	}

	// rest request to send command to pod
	req := clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: command,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		}, scheme.ParameterCodec)

	// Execute the command in the pod
	executor, err := remotecommand.NewSPDYExecutor(api.Config, "POST", req.URL())
	if err != nil {
		fmt.Printf("here was an error executing the commands in the pod. %v\n", err)
		return fmt.Errorf("here was an error executing the commands in the pod. %w", err)
	}

	// Prepare the streams for stdout and stderr
	stdout := os.Stdout
	stderr := os.Stderr

	// stream the output of the command to stdout and stderr
	err = executor.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    false,
	})
	if err != nil {
		fmt.Printf("there was an error streaming the output of the command to stdout, stderr. %v\n", err)
		return fmt.Errorf("there was an error streaming the output of the command to stdout, stderr. %w", err)
	}
	return nil
}

// confirm the deployment exists
//deploymentName, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deployName, v1.GetOptions{})
//if err != nil {
//	fmt.Printf("there was an error getting the deployment %v.\n%v", deploymentName, err)
//	return fmt.Errorf("there was an error getting the deployment %v. %w", deploymentName, err)
//}

// get the current replica value so we can scale down if needed
