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
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("backup called")

		api, err := connectToK8s()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error connecting to the cluster, check your connectivity. %v", err)
			log.Fatalf("error connecting to the cluster, check your connectivity. %v", err)
		}
		err = executeBackup(api)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error executing the backup, check the logs. %v", err)
			log.Fatalf("error executing the backup, check the logs. %v", err)
		}
	},
}

func init() {
	migrateCmd.AddCommand(backupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// backupCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// backupCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func executeBackup(api *KubernetesAPI) error {

	var (
		// set the clientset
		clientset = api.Client
		// Define the namespace and deployment name
		namespace      = "cnvrg"
		deploymentName = "postgres"
	)

	// Get the Pods associated with the Deployment
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{
		LabelSelector: labels.Set{"app": deploymentName}.AsSelector().String(),
	})
	if err != nil {
		panic(err)
	}

	// Choose the first Pod from the list
	if len(pods.Items) == 0 {
		fmt.Println("No Pods found for the Deployment.")
		return fmt.Errorf("there was no pods found for this deployment %w", err)
	}
	podName := pods.Items[0].Name

	// Define the command to execute in the pod
	//command := []string{"ls", "-l"}

	// Convert command slice to string
	//cmdStr := strings.Join(command, " ")
	//fmt.Println(cmdStr)

	command := []string{
		"sh",
		"-c",
		"echo $HOME; ls -l && echo hello",
	}

	// Set up the exec request
	/*
		req := clientset.CoreV1().RESTClient().
			Post().
			Resource("pods").
			Name(podName).
			Namespace(namespace).
			SubResource("exec").
			Param("container", pods.Items[0].Spec.Containers[0].Name).
			Param("command", cmdStr).
			Param("stdin", "false").
			Param("stdout", "true").
			Param("stderr", "true").
			Param("tty", "false").
			VersionedParams(&_v1.PodExecOptions{})
	*/

	req := clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{ // Use metav1.PodExecOptions
			Command: command,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		}, scheme.ParameterCodec)

	// Execute the command in the pod
	executor, err := remotecommand.NewSPDYExecutor(api.Config, "POST", req.URL())
	if err != nil {
		panic(err)
	}

	// Prepare the streams for stdout and stderr
	stdout := os.Stdout
	stderr := os.Stderr

	// Execute the command
	err = executor.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    false,
	})
	if err != nil {
		panic(err)
	}
	return nil
}
