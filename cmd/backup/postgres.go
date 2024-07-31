/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package backup

import (
	"context"
	"fmt"
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
	Short: "Backup the postgres database",
	Long: `Backs up the postgres database by performing a pg_dump
on the running postgres pod. This command will scale down the cnvrg.io
application, so use during a downtime window.

Examples:

# Backups the default postgres database and files in the cnvrg namespace.
  cnvrgctl backup postgres -n cnvrg`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("postgress command called")

		// set result to false until a successfull backup
		result := false

		// target deployment of the postgres backup
		targetFlag, _ := cmd.Flags().GetString("target")

		// grab the namespace from the -n flag if not specified default is used
		nsFlag, _ := cmd.Flags().GetString("namespace")

		// Define the key of the deployment label for the postgres deployment
		labelFlag, _ := cmd.Flags().GetString("label")

		// flag to disable scaling the pods before the backup
		disableScaleFlag, _ := cmd.Flags().GetBool("disable-scale")

		// flag to disable scaling the pods before the backup
		fileLocationFlag, _ := cmd.Flags().GetString("file-location")

		// flag to define backup file name
		fileNameFlag, _ := cmd.Flags().GetString("file-name")

		// connect to kubernetes and define clientset and rest client
		api, err := root.ConnectToK8s()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error connecting to the cluster, check your connectivity. %v", err)
			log.Fatalf("error connecting to the cluster, check your connectivity. %v", err)
		}

		// scale down the application pods to prepare for backups
		if !disableScaleFlag {
			err = root.ScaleDeployDown(api, nsFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error scaling the deployment. %v", err)
				log.Fatalf("error scaling the deployment. %v\n", err)
			}
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
		result, err = copyDBLocally(api, nsFlag, podName, fileLocationFlag, fileNameFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error copying the database file. %v", err)
			log.Fatalf("error copying the database file. %v\n", err)
		} else {
			fmt.Printf("backup %s saved to '%s'\n", fileNameFlag, fileLocationFlag)
			log.Printf("backup %s saved to '%s'\n", fileNameFlag, fileLocationFlag)

		}

		//If the backup is successful and disable-scale flag is false, scale back up the pods
		if result && !disableScaleFlag {
			root.ScaleDeployUp(api, nsFlag)
		}

	},
}

func init() {
	backupCmd.AddCommand(postgresCmd)

	// flag to define the release name
	postgresCmd.Flags().StringP("target", "t", "postgres", "Name of postgres deployment to backup.")

	// flag to disable scaling the pods before the backup
	postgresCmd.Flags().BoolP("disable-scale", "", false, "Disable scaling the app, cnvrg-operator and 'kiq' pods to 0 before the backup.")

	// flag to define the app label key
	postgresCmd.Flags().StringP("label", "l", "app", "Define the key of the deployment label for the postgres deployment. example: app.kubernetes.io/name")

	// flag to define restore location
	postgresCmd.Flags().StringP("file-location", "f", ".", "Local location to save the postgres backup file.")

	// flag to define backup file name
	postgresCmd.Flags().StringP("file-name", "", "cnvrg-db-backup.sql", "Name of the postgres backup file.")
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
		"export PGPASSWORD=$POSTGRESQL_PASSWORD; echo postgres password is, $POSTGRESQL_PASSWORD; pg_dump -h postgres -U cnvrg -d cnvrg_production -Fc > cnvrg-db-backup.sql",
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
