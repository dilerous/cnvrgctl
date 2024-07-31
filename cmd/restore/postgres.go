/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package restore

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	root "github.com/dilerous/cnvrgctl/cmd"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
)

// postgresCmd represents the postgres command
var postgresCmd = &cobra.Command{
	Use:   "postgres",
	Short: "Restore the Postgres database backup.",
	Long: `This command will scale down the application and supporting pods and 
restore the Postgres database. 

Examples:

# Restores the Postgres database to the postgres pod specified by the namespace.
  cnvrgctl restore postgres -n cnvrg

# Specify namespace, deployment label key, and deployment name.
  cnvrgctl restore postgres --target postgres-ha --label app.kubernetes.io/name -n cnvrg`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("postgres command called")

		svcPort := "5432"

		// target deployment of the postgres backup
		targetFlag, _ := cmd.Flags().GetString("target")

		// grab the namespace from the -n flag if not specified default is used
		nsFlag, _ := cmd.Flags().GetString("namespace")

		// grab the namespace from the -n flag if not specified default is used
		labelFlag, _ := cmd.Flags().GetString("selector")

		// Define the local location to save the file
		fileLocationFlag, _ := cmd.Flags().GetString("file-location")

		// flag to define backup file name
		fileNameFlag, _ := cmd.Flags().GetString("file-name")

		// connect to the kubernetes api and set clientset and rest client
		api, err := root.ConnectToK8s()
		if err != nil {
			fmt.Printf("error connecting to the cluster, check your connectivity. %v", err)
			log.Printf("error connecting to the cluster, check your connectivity. %v", err)
		}

		// get the postgres pod name
		podName, err := root.GetDeployPod(api, targetFlag, nsFlag, labelFlag)
		if err != nil {
			fmt.Printf("error getting pod name, %v", err)
			log.Printf("error getting pod name, %v", err)
		}

		// scale down the app, operator and kiq pods
		err = root.ScaleDeployDown(api, nsFlag)
		if err != nil {
			fmt.Printf("there was a problem with scaling down the pods. %v ", err)
			log.Printf("there was a problem with scaling down the pods. %v", err)
		}

		// copy the local sql backup to the postgres pod
		copyDBRemotely(api, nsFlag, podName, fileLocationFlag, fileNameFlag)
		if err != nil {
			fmt.Printf("there was a problem copying the local backup to the pod. %v ", err)
			log.Printf("there was a problem copying the local backup to the pod. %v", err)
		}

		// forward the postgres service and execute the sql commands
		err = portForwardSvc(api, nsFlag, podName, svcPort)
		if err != nil {
			fmt.Printf("error forwarding the service. %v ", err)
			log.Printf("error forwarding the service. %v", err)
		}

		// restore the postgres backup from the dump file
		err = restoreDBBackup(api, nsFlag, podName)
		if err != nil {
			fmt.Printf("error restoring the backup, check the logs. %v ", err)
			log.Printf("error restoring the backup, check the logs. %v", err)
		}

		err = root.ScaleDeployUp(api, nsFlag)
		if err != nil {
			fmt.Printf("there was a problem with scaling up the pods. %v ", err)
			log.Printf("there was a problem with scaling up the pods. %v", err)
		}
	},
}

func init() {
	restoreCmd.AddCommand(postgresCmd)

	// flag to define the release name
	postgresCmd.Flags().StringP("target", "t", "postgres", "Name of postgres deployment to retore.")

	// flag to define the app label key
	postgresCmd.Flags().StringP("selector", "l", "app", "Define the deployment label for the postgres deployment. example: app.kubernetes.io/name")

	// flag to define redis backup file name
	postgresCmd.Flags().StringP("file-name", "", "cnvrg-db-backup.sql", "Name of the redis backup file.")

	// flag to define restore location
	postgresCmd.Flags().StringP("file-location", "f", ".", "Local location of the postgres backup file.")
}

func dropPgDB(a root.KubernetesAPI, n string, name string) error {
	log.Println("dropPgDB function called.")

	var (
		api       = a
		podName   = name
		namespace = n
	)

	// TODO: reference the struct for targetFlag and labelFlag
	// Connect to the PostgreSQL database
	db, err := connectToPostgreSQL(&api, namespace, podName)
	if err != nil {
		log.Printf("error connecting to postgresql. %v", err)
		return fmt.Errorf("error connecting to postgresql. %w", err)
	}
	defer db.Close()

	// Execute SQL commands
	sqlCommands := []string{
		"UPDATE pg_database SET datallowconn = 'false' WHERE datname = 'cnvrg_production';",
		"ALTER DATABASE cnvrg_production CONNECTION LIMIT 0;",
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'cnvrg_production';",
		"DROP DATABASE cnvrg_production;",
		"CREATE DATABASE cnvrg_production;",
	}
	for _, cmd := range sqlCommands {
		if _, err := db.Exec(cmd); err != nil {
			fmt.Fprintf(os.Stderr, "error executing SQL command: %v", err)
			log.Fatalf("error executing SQL command: %v", err)
		}
	}
	fmt.Println("SQL commands executed successfully.")
	return nil
}

// Connect to the PostgreSQL database
func connectToPostgreSQL(clientset *root.KubernetesAPI, namespace, podName string) (*sql.DB, error) {
	log.Println("connectToPostgreSQL function called.")

	api := clientset
	// Get PostgreSQL pod IP address
	pod, err := api.Client.CoreV1().Pods(namespace).Get(context.Background(), podName, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %v", err)
	}

	podIP := pod.Status.PodIP
	fmt.Println(podIP)

	ip := "localhost"

	// Connect to PostgreSQL database
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s dbname=postgres user=cnvrg sslmode=disable", ip))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	//fmt.Println(string(output))
	return db, nil
}

// forwards the service so the database copy can take place
// arguments are namespace "ns", "n" name of pod and "p" for the port number of the svc
func portForwardSvc(api *root.KubernetesAPI, ns string, n string, p string) error {
	log.Println("portForwardSvc function called")

	var (
		namespace = ns
		podName   = n
		localPort = p
	)
	fmt.Println("the pod name is " + podName)

	// Port forward the service
	stopCh := make(chan struct{})
	readyChan := make(chan struct{})
	errChan := make(chan error, 1)
	go func() {

		restClient := api.Client.CoreV1().RESTClient()
		req := restClient.Post().
			Resource("pods").
			Name(podName).
			Namespace(namespace).
			SubResource("portforward")

		url := req.URL()

		roundTripper, upgrader, err := spdy.RoundTripperFor(api.Config)
		if err != nil {
			errChan <- fmt.Errorf("error creating round tripper: %v", err)
			return
		}

		var (
			dialer = spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, url)
			port   = []string{localPort}
		)

		// Create the port forwarder object
		pf, err := portforward.New(dialer, port, stopCh, readyChan, os.Stdout, os.Stderr)
		if err != nil {
			errChan <- fmt.Errorf("error creating the forwarder object: %v", err)
			return
		}

		// Start port forwarding in the background
		fmt.Println("Starting the port forwarding...")
		err = pf.ForwardPorts()
		if err != nil {
			errChan <- fmt.Errorf("error in port forwarding: %v", err)
			return
		}
	}()
	time.Sleep(5 * time.Second)

	// execute the sql commands against the postgres DB
	if strings.Contains(podName, "postgres") {
		err := dropPgDB(*api, namespace, podName)
		if err != nil {
			fmt.Printf("error restoring the postgres database. %v ", err)
			log.Printf("error restoring the postgres database. %v", err)
		}
	}

	// notify user the connection is being closed
	fmt.Println("Changes made closing the connection...")

	// close the channel
	close(stopCh)

	select {
	case err := <-errChan:
		if err != nil {
			log.Printf("error port forwarding: %v", err)
			return fmt.Errorf("error port forwarding. %w", err)
		}
		fmt.Println("Port forwarding completed successfully.")
		log.Println("Port forwarding completed successfully.")
	case <-readyChan:
		fmt.Println("forwarding successfully closed.")
		log.Println("forwarding successfully closed.")
	}
	return nil
}

// TODO: add flags to define the backup file name and path
// takes the arguments namespace "ns" pod name "p" the file path "f" and the backup file name "n"
func copyDBRemotely(api *root.KubernetesAPI, ns string, p string, f string, n string) error {
	log.Println("copyDBLocally function called.")

	//TODO: add flag to specify location of file
	var ( // Set the pod and namespace
		podName    = p
		namespace  = ns
		filePath   = f
		backupFile = n
		clientset  = api.Client
		command    = []string{"cp", "/dev/stdin", "/opt/app-root/src/cnvrg-db-backup.sql"}
		config     = api.Config
	)

	// open the file that was just created
	file, err := os.Open(filePath + backupFile)
	if err != nil {
		log.Printf("opening the file failed. %s\n", err)
		return fmt.Errorf("opening the file failed. %w", err)
	}
	defer file.Close()

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
			Stdin:   true,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		}, scheme.ParameterCodec)

	// execute the command
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		log.Printf("error executing the remote command. %v\n", err)
		return fmt.Errorf("error execuuting the remote command. %w", err)
	}

	// set the variables to type byte and stream the output to those variables
	var stdout, stderr bytes.Buffer

	// Create a waitgroup to synchronize the completion of streaming
	var wg sync.WaitGroup
	wg.Add(1)

	// execute the command
	go func() {
		exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
			Stdin:  file,
			Stdout: &stdout,
			Stderr: &stderr,
		})
		wg.Done() // Signal that streaming is complete
	}()

	// Wait for streaming to finish
	wg.Wait()

	//no errors return nil
	return nil
}
