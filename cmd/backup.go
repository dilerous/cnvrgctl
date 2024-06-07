/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
	Short: "Executes backup commands on the postgres pod and files storage",
	Long: `This command will initiate a pg_dump of the postgres database
and save it to a file in the current directory. It will also backup the files from
the minio cnvrg-storage bucket.

Examples:

# Backups the default postgres database and files in the cnvrg namespace.
  cnvrgctl migrate backup -n cnvrg

# Specify namespace, deployment label key, and deployment name.
  cnvrgctl migrate backup --target postgres-ha --label app.kubernetes.io/name -n cnvrg`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("called the migrate backup command function")

		result := false

		// target deployment of the postgres backup
		targetFlag, _ := cmd.Flags().GetString("target")

		// grab the namespace from the -n flag if not specified default is used
		nsFlag, _ := cmd.Flags().GetString("namespace")

		// grab the namespace from the -n flag if not specified default is used
		labelFlag, _ := cmd.Flags().GetString("label")

		// grab the namespace from the -n flag if not specified default is used
		s3SecretName, _ := cmd.Flags().GetString("secret-name")

		// connect to kubernetes and define clientset and rest client
		api, err := connectToK8s()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error connecting to the cluster, check your connectivity. %v", err)
			log.Fatalf("error connecting to the cluster, check your connectivity. %v", err)
		}

		err = scaleDeployDown(api, nsFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error scaling the deployment. %v", err)
			log.Fatalf("error scaling the deployment. %v\n", err)
		}

		podName, err := getDeployPod(api, targetFlag, nsFlag, labelFlag)
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
		err = copyDBLocally(api, nsFlag, podName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error copying the database file. %v", err)
			log.Fatalf("error copying the database file. %v\n", err)
		}

		// get the object data and store in the ObjectStorage struct
		objectData, err := getObjectSecret(api, s3SecretName, nsFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get the S3 secret. %v", err)
			log.Fatalf("failed to get the S3 secret. %v\n", err)
		}

		// check which type of bucket either minio or s3
		//TODO: reduce all the if statements
		if objectData.Type == "minio" {
			err := connectToMinio(objectData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get the S3 secret. %v", err)
				log.Fatalf("failed to get the S3 secret. %v\n", err)
			}
			result, err = backupMinioBucketLocal(objectData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error backing up the bucket, check the logs. %v", err)
				log.Fatalf("error backing up the bucket, check the logs. %v\n", err)
			}

		} else {
			err = connectToS3(objectData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to connect to the s3 bucket. %v", err)
				log.Fatalf("failed to connect to the s3 bucket. %v", err)
			}
		}

		//If the backup is successful, scale back up the pods
		if result {
			scaleDeployUp(api, nsFlag)
		}

	},
}

func init() {
	migrateCmd.AddCommand(backupCmd)

	// flag to define the release name
	backupCmd.Flags().StringP("target", "t", "postgres", "Name of postgres deployment to backup.")

	// flag to define the app label key
	backupCmd.Flags().StringP("label", "l", "app", "Define the key of the deployment label for the postgres deployment. example: app.kubernetes.io/name")

	// flag to define the release name
	backupCmd.Flags().StringP("secret-name", "", "cp-object-storage", "Define the secret name for the S3 bucket credentials.")

}

// scales the following deployments "app", "sidekiq", "systemkiq", "searchkiq", "cnvrg-operator" in the namespace specified
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

// get the pod name from the deployment this will be passed to executeBackup function
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

// Executes a pg dump of the postgres database by getting the postgres pod name then running
// pg_dump on the postgres pod
func executePostgresBackup(api *KubernetesAPI, pod string, nsFlag string) error {
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

func copyDBLocally(api *KubernetesAPI, nsFlag string, pod string) error {
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
		return fmt.Errorf("error execuuting the remote command. %w", err)
	}

	// set the variables to type byte and stream the output to those variables
	var stdout, stderr bytes.Buffer
	exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	// Create a local file to write to
	localFile, err := os.Create(filePath + backupFile)
	if err != nil {
		log.Printf("error creating local file. %v\n", err)
		return fmt.Errorf("error creating local file. %w", err)
	}
	defer localFile.Close()

	// open the file that was just created
	file, err := os.OpenFile(filePath+backupFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("opening the file failed. %s\n", err)
		return fmt.Errorf("opening the file failed. %w", err)
	}
	defer file.Close()

	// copy the stream output from the cat command to the file.
	_, err = io.Copy(file, &stdout)
	if err != nil {
		log.Printf("the copy failed. %v", err)
		return fmt.Errorf("the copy failed. %w", err)
	}

	return nil
}

func connectToS3(o *ObjectStorage) error {
	log.Println("connectToS3 function called.")

	// Initialize a session that the SDK will use to load
	// credentials from the shared credentials file ~/.aws/credentials
	sess, err := session.NewSession(&aws.Config{Region: aws.String(o.Region)}, nil)
	if err != nil {
		log.Printf("the copy failed. %v", err)
		return fmt.Errorf("the copy failed. %w", err)
	}

	// Create S3 client
	s3Client := s3.New(sess)

	// List buckets
	buckets, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		log.Printf("error listing the buckets, check your credentials. %v", err)
		return fmt.Errorf("the copy failed. %w", err)
	}

	log.Println("Buckets:")
	for _, bucket := range buckets.Buckets {
		log.Println(*bucket.Name)
	}

	// Get object from bucket
	obj, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(o.BucketName),
		Key:    aws.String(o.AccessKey),
	})
	if err != nil {
		log.Printf("the copy failed. %v", err)
		return fmt.Errorf("the copy failed. %w", err)
	}

	log.Println("Object:")
	log.Println(obj)
	return nil
}

// TODO: check if useSSL = false, conslidate with get bucket function
func backupMinioBucketLocal(o *ObjectStorage) (bool, error) {
	log.Println("backupMinioBucketLocal function called.")

	// Initialize a new MinIO client
	useSSL := false

	//TODO: remove this section is already used in connectToMinio
	uWithoutHttp := strings.Replace(o.Endpoint, "http://", "", 1)
	log.Println(uWithoutHttp)

	minioClient, err := minio.New(uWithoutHttp, &minio.Options{
		Creds:  credentials.NewStaticV4(o.AccessKey, o.SecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Printf("error connecting to minio. %v", err)
		return false, fmt.Errorf("error connecting to minio. %w", err)
	}

	// grabs all the objects and copies them to the local folder ./cnvrg-storage
	allObjects := minioClient.ListObjects(context.Background(), o.BucketName, minio.ListObjectsOptions{Recursive: true})
	for object := range allObjects {
		log.Println(object.Key)
		fmt.Println(object.Key)
		minioClient.FGetObject(context.Background(), o.BucketName, object.Key, "./cnvrg-storage/"+object.Key, minio.GetObjectOptions{})
	}

	fmt.Println("Successfully copied objects!")
	return true, nil
}
