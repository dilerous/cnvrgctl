/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("restore command called")
		o := ObjectStorage{}

		// grab the namespace from the -n flag if not specified default is used
		s3SecretName, _ := cmd.Flags().GetString("secret-name")

		// grab the namespace from the -n flag if not specified default is used
		nsFlag, _ := cmd.Flags().GetString("namespace")

		// define the url to the minio api
		urlFlag, _ := cmd.Flags().GetString("minio-url")
		o.Endpoint = urlFlag

		// define the target bucket to restore the files too.
		bucketFlag, _ := cmd.Flags().GetString("bucket")
		if bucketFlag != "cnvrg-storage" {
			o.BucketName = bucketFlag
		}

		// grab the namespace from the -n flag if not specified default is used
		skFlag, _ := cmd.Flags().GetString("secret-key")
		if skFlag != "" {
			o.SecretKey = skFlag
		}

		// grab the namespace from the -n flag if not specified default is used
		akFlag, _ := cmd.Flags().GetString("access-key")
		if akFlag != "" {
			o.AccessKey = akFlag
		}

		// connect to kubernetes and define clientset and rest client
		api, err := connectToK8s()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error connecting to the cluster, check your connectivity. %v", err)
			log.Fatalf("error connecting to the cluster, check your connectivity. %v", err)
		}
		// get the object data and store in the ObjectStorage struct
		if skFlag == "" && akFlag == "" {
			objectData, err := getObjectData(api, s3SecretName, nsFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get the S3 secret. %v", err)
				log.Fatalf("failed to get the S3 secret. %v\n", err)
			}
			uploadFilesMinio(objectData)
		}
		// upload files from minio using info from the flags
		uploadFilesMinio(&o)
	},
}

func init() {
	migrateCmd.AddCommand(restoreCmd)

	// flag to define the secret for the object storage credentials
	restoreCmd.Flags().StringP("secret-name", "", "cp-object-storage", "define the secret name for the S3 bucket credentials.")

	// flag to define the secret for the object storage credentials
	restoreCmd.Flags().StringP("secret-key", "k", "", "define the secret key for the S3 bucket credentials.")

	// flag to define the secret for the object storage credentials
	restoreCmd.Flags().StringP("access-key", "a", "", "define the access key for the S3 bucket credentials.")

	// flag to define the backup bucket target
	restoreCmd.Flags().StringP("bucket", "b", "cnvrg-storage", "define the bucket to restore the files too.")

	// flag to define the backup bucket target
	restoreCmd.Flags().StringP("minio-url", "u", "", "define the url to the minio api.")
}

// TODO: check if useSSL = false, conslidate with get bucket function
// TODO: add in getting cp-object-secret
func uploadFilesMinio(o *ObjectStorage) error {
	log.Println("uploadFiles Minio function called.")
	// Walk all the subdirectories of a directory
	dir := "./cnvrg-storage"
	useSSL := false

	url := strings.TrimSuffix(o.Endpoint, "https://")
	fmt.Println(url)

	minioClient, err := minio.New(url, &minio.Options{
		Creds:  credentials.NewStaticV4(o.AccessKey, o.SecretKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		log.Fatalf("failed to configure minio client. %v\n", err)
		return fmt.Errorf("failed to configure minio client. %w", err)
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return err
		}

		if !info.IsDir() {
			// Upload all files
			ui, err := minioClient.FPutObject(context.TODO(), o.BucketName, path, path, minio.PutObjectOptions{})
			if err != nil {
				log.Fatalf("failed to upload files to minio bucket. %v\n", err)
				return fmt.Errorf("failed to upload files to minio bucket. %w", err)

			}
			fmt.Println(ui.Key)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to get the S3 secret. %v\n", err)
		return fmt.Errorf("failed to get the S3 secret. %w", err)

	}
	return nil
}

/*
func setObjectStorageValues(o *ObjectStorage) error {
	type ObjectStorage struct {
		AccessKey  string
		SecretKey  string
		Region     string
		Endpoint   string
		Type       string
		BucketName string
		Namespace  string
	}



}
*/
