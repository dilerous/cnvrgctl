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

// redisCmd represents the redis command
var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Backup the Redis database",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("redis command called")

		// set the redis secret to the Redis Secret struct
		var (
			redisStuct = root.RedisSecret{}
		)

		// grab the namespace from the -n flag if not specified default is used
		nsFlag, _ := cmd.Flags().GetString("namespace")

		// grab the namespace from the -n flag if not specified default is used
		redisSecretName, _ := cmd.Flags().GetString("secret-name")

		// Define the key of the deployment label for the redis deployment
		labelFlag, _ := cmd.Flags().GetString("label")

		// Name of the redis deployment
		targetFlag, _ := cmd.Flags().GetString("target")

		// Define the local location to save the file
		fileLocationFlag, _ := cmd.Flags().GetString("file-location")

		// flag to define backup file name
		fileNameFlag, _ := cmd.Flags().GetString("file-name")

		// flag to disable scaling the pods before the backup
		disableScaleFlag, _ := cmd.Flags().GetBool("disable-scale")

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
				fmt.Fprintf(os.Stderr, "error scaling the de ployment. %v", err)
				log.Fatalf("error scaling the deployment. %v\n", err)
			}
		}

		// capture the redis password
		password, err := root.GetRedisPassword(api, redisSecretName, nsFlag, &redisStuct)
		if err != nil {
			fmt.Printf("error capturing the redis password, check the namespace and secret exists. %v", err)
			log.Printf("error capturing the redis password, check the namespace and secret exists. %v", err)
		}

		// get the name of the running redis pod
		podName, err := root.GetDeployPod(api, targetFlag, nsFlag, labelFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting the pod name check the deployment label, namespace and target. %v", err)
			log.Fatalf("error getting the pod name check the deployment label, namespace and target. %v", err)
		}

		// connect to the redis pod and execute the backup
		err = executeRedisBackup(api, podName, nsFlag, password.RedisPassword)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error executing the backup, check the logs. %v", err)
			log.Fatalf("error executing the backup, check the logs. %v\n", err)
		}

		result, err := copyDBLocally(api, nsFlag, podName, fileLocationFlag, fileNameFlag)
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
	backupCmd.AddCommand(redisCmd)

	// flag to define the deployment name
	redisCmd.Flags().StringP("target", "t", "redis", "Name of redis deployment to backup.")

	// flag to disable scaling the pods before the backup
	redisCmd.Flags().BoolP("disable-scale", "", false, "Disable scaling the app, cnvrg-operator and 'kiq' pods to 0 before the backup.")

	// flag to define the app label key
	redisCmd.Flags().StringP("label", "l", "app", "Define the key of the deployment label for the redis deployment. example: app.kubernetes.io/name")

	// flag to define restore location
	redisCmd.Flags().StringP("file-location", "f", ".", "Local location to save the redis backup file.")

	// flag to define backup file name
	redisCmd.Flags().StringP("file-name", "", "dump.rdb", "Name of the redis backup file.")

	// flag to define the release name
	redisCmd.Flags().StringP("secret-name", "", "redis-creds", "Define the secret name for the Redis credentials.")
}

// Executes a backup of Redis by executing the commands from within the pod
// takes the arguments pod name "n" the namespace "ns" and the redis password "p"
func executeRedisBackup(api *root.KubernetesAPI, n string, ns string, p string) error {
	log.Println("executeRedisBackup function called.")
	// set variables for the clientset and pod name
	var (
		clientset = api.Client
		podName   = n
		namespace = ns
		password  = p
		green     = "\033[32m"
		reset     = "\033[0m"
	)

	// Command to save a redis backup of the database
	command := []string{
		"sh",
		"-c",
		fmt.Sprintf("redis-cli -a %v save", password),
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
	fmt.Println(string(green), "Redis DB Backup successful!", string(reset))
	return nil
}
