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
		log.Println("restore files command called")

		// set result to false until a successfull backup
		result := false

		//define the empty ObjectStorage struct
		o := root.ObjectStorage{}

		// get the name of the secret that has the object storage credentials
		o.SecretName, _ = cmd.Flags().GetString("secret-name")
		s3SecretName, _ := cmd.Flags().GetString("secret-name")

		// grab the namespace from the -n flag if not specified default is used
		o.Namespace, _ = cmd.Flags().GetString("namespace")
		nsFlag, _ := cmd.Flags().GetString("namespace")

		// define the url to the minio api
		o.Endpoint, _ = cmd.Flags().GetString("bucket-url")

		// define the target bucket to restore the files too.
		o.BucketName, _ = cmd.Flags().GetString("bucket")

		// set the secret key if defined
		o.SecretKey, _ = cmd.Flags().GetString("secret-key")

		// set the access key if defined
		o.AccessKey, _ = cmd.Flags().GetString("access-key")

		// set the session key if defined
		o.SessionKey, _ = cmd.Flags().GetString("session-key")

		if o.Endpoint == "s3.amazonaws.com" {
			o.UseSSL = true
			listS3Bucket(o)
		}

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
			//err = connectToS3(objectData)
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
			root.ScaleDeployUp(api, o.Namespace)
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

	// flag to define the secret for the object storage credentials
	filesCmd.Flags().StringP("secret-key", "k", "", "define the secret key for the S3 bucket credentials. (required if bucket, access-key and minio-url is set)")

	// flag to define the secret for the object storage credentials
	filesCmd.Flags().StringP("access-key", "a", "", "define the access key for the S3 bucket credentials. (required if secret-key, bucket and minio-url is set)")

	// flag to define the session key for the object storage credentials
	filesCmd.Flags().StringP("session-key", "", "", "define the session key for the S3 bucket credentials.")

	// flag to define the backup bucket target
	filesCmd.Flags().StringP("bucket", "b", "cnvrg-storage", "define the bucket to restore the files from. (required if secret-key, access-key and minio-url is set)")

	// flag to define the backup bucket target
	filesCmd.Flags().StringP("bucket-url", "u", "s3.amazonaws.com", "define the url to the bucket api. (required if secret-key, access-key and bucket is set)")

	// flag to define the source files
	filesCmd.Flags().StringP("source", "s", "cnvrg-storage", "define the source folder to backup files too locally.")

	// if any of the flags defined are set, they all must be set
	filesCmd.MarkFlagsRequiredTogether("secret-key", "access-key", "bucket", "bucket-url")
}

// list S3 bucket for testing
// TODO: create function that sets useSSL based on url
func listS3Bucket(o root.ObjectStorage) {
	log.Println("listS3Bucket function called")

	// Define your bucket credentials
	accessKeyID := o.AccessKey
	secretAccessKey := o.SecretKey
	sessionKey := o.SessionKey
	endpoint := o.Endpoint // For AWS S3, use "s3.amazonaws.com". For MinIO, use your MinIO server address.
	useSSL := o.UseSSL     // For secure connection use true, otherwise false.
	bucketName := o.BucketName

	// Initialize MinIO client object
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, sessionKey),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// List objects in the bucket
	objectCh := minioClient.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	fmt.Println("Objects in bucket", bucketName)
	for object := range objectCh {
		if object.Err != nil {
			log.Fatalln(object.Err)
		}
		fmt.Println("Bucket Name: ", bucketName)
		fmt.Println("Name:         ", object.Key)
		fmt.Println("Last modified:", object.LastModified)
		fmt.Println("Size:         ", object.Size)
		fmt.Println("Storage class:", object.StorageClass)
		fmt.Println("")
	}
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
