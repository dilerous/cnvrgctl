/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	root "github.com/dilerous/cnvrgctl/cmd"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// databaseCmd represents the database command
var postgresCmd = &cobra.Command{
	Use:   "postgres",
	Short: "Will backup the postgres database",
	Long: `Backs up the postgres database by performing a pg_dump
on the running postgres pod. This command will scale down the cnvrg.io
application, so use during a downtime window.

Examples:

# Backups the default postgres database and files in the cnvrg namespace.
  cnvrgctl backup postgres -n cnvrg`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("database called")

		// set result to false until a successfull backup
		result := false

		// target deployment of the postgres backup
		targetFlag, _ := cmd.Flags().GetString("target")

		// grab the namespace from the -n flag if not specified default is used
		nsFlag, _ := cmd.Flags().GetString("namespace")

		// Define the key of the deployment label for the postgres deployment
		labelFlag, _ := cmd.Flags().GetString("label")

		// connect to kubernetes and define clientset and rest client
		api, err := root.ConnectToK8s()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error connecting to the cluster, check your connectivity. %v", err)
			log.Fatalf("error connecting to the cluster, check your connectivity. %v", err)
		}

		// scale down the application pods to prepare for backups
		err = root.ScaleDeployDown(api, nsFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error scaling the deployment. %v", err)
			log.Fatalf("error scaling the deployment. %v\n", err)
		}

		// get the pod name from the deployment defined
		podName, err := root.GetDeployPod(api, targetFlag, nsFlag, labelFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting the pod name check the deployment label, namespace and target. %v", err)
			log.Fatalf("error getting the pod name check the deployment label, namespace and target. %v", err)
		}

		// execute the backup of the target postgres deployment
		err = executePostgresBackup(api, podName, nsFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error executing the backup, check the logs. %v", err)
			log.Fatalf("error executing the backup, check the logs. %v\n", err)
		}

		// copy the postgres backup to the local machine
		result, err = copyDBLocally(api, nsFlag, podName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error copying the database file. %v", err)
			log.Fatalf("error copying the database file. %v\n", err)
		}

		//If the backup is successful, scale back up the pods
		if result {
			root.ScaleDeployUp(api, nsFlag)
		} else {
			log.Println("backup result was set to false, there was a problem. result value: ", result)
			fmt.Println("there was a problem with the backup, check the logs.")
		}

	},
}

func init() {
	backupCmd.AddCommand(postgresCmd)

	// flag to define the release name
	postgresCmd.Flags().StringP("target", "t", "postgres", "Name of postgres deployment to backup.")

	// flag to define the app label key
	postgresCmd.Flags().StringP("label", "l", "app", "Define the key of the deployment label for the postgres deployment. example: app.kubernetes.io/name")
}

// Executes a pg dump of the postgres database by getting the postgres pod name then running
// pg_dump on the postgres pod
func executePostgresBackup(api *root.KubernetesAPI, pod string, nsFlag string) error {
	log.Println("executePostgresBackup function called.")
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
		"export PGPASSWORD=$POSTGRESQL_PASSWORD; echo $POSTGRESQL_PASSWORD; pg_dump -h postgres -U cnvrg -d cnvrg_production -Fc > cnvrg-db-backup.sql",
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
		log.Printf("here was an error executing the commands in the pod. %v\n", err)
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
		log.Printf("there was an error streaming the output of the command to stdout, stderr. %v\n", err)
		return fmt.Errorf("there was an error streaming the output of the command to stdout, stderr. %w", err)
	}

	//TODO add in a check if the file exits here cnvrg-db-backup.sql
	fmt.Println("Postgres DB Backup successful!")
	return nil
}

// copies the postgres-backup file from the pod to the local machine
func copyDBLocally(api *root.KubernetesAPI, nsFlag string, pod string) (bool, error) {
	log.Println("copyDBLocally function called.")

	//TODO: add flag to specify location of file
	var ( // Set the pod and namespace
		podName    = pod
		namespace  = nsFlag
		filePath   = "./"
		backupFile = "cnvrg-db-backup.sql"
		clientset  = api.Client
		command    = []string{"cat", backupFile}
		config     = api.Config
	)

	// Create a REST client
	log.Println("Creating the rest client call")
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

	// execute the command
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		log.Printf("error executing the remote command. %v\n", err)
		return false, fmt.Errorf("error execuuting the remote command. %w", err)
	}

	// set the variables to type byte and stream the output to those variables
	var stdout, stderr bytes.Buffer

	// execute the command
	exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	// Create a local file to write to
	localFile, err := os.Create(filePath + backupFile)
	if err != nil {
		log.Printf("error creating local file. %v\n", err)
		return false, fmt.Errorf("error creating local file. %w", err)
	}
	defer localFile.Close()

	// open the file that was just created
	file, err := os.OpenFile(filePath+backupFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("opening the file failed. %s\n", err)
		return false, fmt.Errorf("opening the file failed. %w", err)
	}
	defer file.Close()

	// copy the stream output from the cat command to the file.
	_, err = io.Copy(file, &stdout)
	if err != nil {
		log.Printf("the copy failed. %v", err)
		return false, fmt.Errorf("the copy failed. %w", err)
	}

	return true, nil
}
