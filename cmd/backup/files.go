/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package backup

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	root "github.com/dilerous/cnvrgctl/cmd"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
)

// bucketCmd represents the bucket command
var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "Backup the files from the cnvrg.io storage bucket",
	Long: `The command will backup the object storage bucket used to host the files and 
datasets in the cnvrg environment.

Examples:

# Backups the files stored in the bucket in the cnvrg namespace.
  cnvrgctl backup files -n cnvrg`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("restore files called")

		// set result to false until a successfull backup
		result := false

		// get the name of the secret that has the object storage credentials
		s3SecretName, _ := cmd.Flags().GetString("secret-name")

		// grab the namespace from the -n flag if not specified default is used
		nsFlag, _ := cmd.Flags().GetString("namespace")

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

		// get the object data and store in the ObjectStorage struct
		objectData, err := root.GetObjectSecret(api, s3SecretName, nsFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get the S3 secret. %v", err)
			log.Fatalf("failed to get the S3 secret. %v\n", err)
		}

		// determines the bucket type, then runs the corrispoding functions
		switch objectData.Type {
		case "minio":
			// connect to minio to verify connectivity
			err := connectToMinio(objectData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get the %s secret. %v", s3SecretName, err)
				log.Printf("failed to get the %s secret. %v\n", s3SecretName, err)
			}

			// backup the files from minio to the local drive
			result, err = backupMinioBucketLocal(objectData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error backing up the bucket, check the logs. %v", err)
				log.Printf("error backing up the bucket, check the logs. %v\n", err)
			}

		case "aws":
			err = connectToS3(objectData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to connect to the s3 bucket. %v", err)
				log.Printf("failed to connect to the s3 bucket. %v", err)
			}

		case "gcp":
			fmt.Println("this will be the location of the gcp code")

		case "azure":
			fmt.Println("this will be the location of the azure code")

		default:
			fmt.Println("object storage bucket type unknown.")
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
	backupCmd.AddCommand(filesCmd)

	// flag to define the release name
	filesCmd.Flags().StringP("secret-name", "", "cp-object-storage", "Define the secret name for the S3 bucket credentials.")
}

// connect to an AWS S3 bucket using the AWS SDK
func connectToS3(o *root.ObjectStorage) error {
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
func backupMinioBucketLocal(o *root.ObjectStorage) (bool, error) {
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

// connect to minio storage
// TODO Create generic and move to migrate
// used by backup and restore commands
func connectToMinio(o *root.ObjectStorage) error {
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
