/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package restore

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	root "github.com/dilerous/cnvrgctl/cmd"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore file backups and database backups.",
	Long: `This command will restore file backups to a bucket you specify. By default the credentials,
bucket, and keys will be gathered from the cp-object-storage secret for the restore. You can manually
specify these values using flags. You 

Examples:
	
# Restore the backups to the bucket 'cnvrg-backups'.
  cnvrgctl migrate restore -a minio -k minio123 -u minio.aws.dilerous.cloud -b cnvrg-backups`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	root.RootCmd.AddCommand(restoreCmd)
}

// Restore the database for postgres and redis
// arguments are "api" for the clientset, "ns" for the namespace and "p" for the pod name
func restoreDBBackup(api *root.KubernetesAPI, ns string, p string) error {
	log.Println("restoreDBBackup function called.")

	// set variables for the clientset and pod name
	var (
		clientset = api.Client
		podName   = p
		namespace = ns
		command   = []string{}
		green     = "\033[32m"
		reset     = "\033[0m"
	)

	// if the pod name has postgres sets the proper restore command
	if strings.Contains(podName, "postgres") {
		command = []string{
			"sh",
			"-c",
			"export PGPASSWORD=$POSTGRESQL_PASSWORD; pg_restore -h postgres -p 5432 -U cnvrg -d cnvrg_production -j 8 --verbose cnvrg-db-backup.sql",
		}
	}

	// if the pod name matches redis set the correct commands
	if strings.Contains(podName, "redis") {
		command = []string{
			"sh",
			"-c",
			"echo running the following command: mv /data/appendonly.aof /data/appendonly.aof.old",
			"mv /data/appendonly.aof /data/appendonly.aof.old",
		}
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
	fmt.Println(string(green), "Database restore successful!", string(reset))
	log.Println("Database restore successful!")
	return nil
}
