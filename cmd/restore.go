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
	Short: "Restore file backups to target bucket.",
	Long: `This command will restore file backups to a bucket you specify.

Examples:
	
# Restore the backups to the bucket 'cnvrg-backups'.
  cnvrgctl migrate restore -a minio -k minio123 -u minio.aws.dilerous.cloud -b cnvrg-backups`,
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

		sourceFlag, _ := cmd.Flags().GetString("source")

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
			uploadFilesMinio(objectData, sourceFlag)
		}
		// upload files from minio using info from the flags
		uploadFilesMinio(&o, sourceFlag)
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

	// flag to define the source files
	restoreCmd.Flags().StringP("source", "s", "cnvrg-storage", "define the source files to restore.")
}

// TODO: check if useSSL = false, conslidate with get bucket function
// TODO: add in getting cp-object-secret
func uploadFilesMinio(o *ObjectStorage, s string) error {
	log.Println("uploadFiles Minio function called.")
	// Walk all the subdirectories of a directory
	dir := s
	useSSL := false

	//remove https from the endpoint url
	url := strings.TrimSuffix(o.Endpoint, "https://")

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
			log.Fatalf("unable to walk the file path. %v\n", err)
			return fmt.Errorf("unable to walk the file path. %w", err)
		}

		if !info.IsDir() {
			// trim the bucket name from the path
			pathTrimmed := strings.TrimPrefix(path, dir)
			fmt.Println("The path trimmed is: ", pathTrimmed)

			// Upload all files
			ui, err := minioClient.FPutObject(context.TODO(), o.BucketName, pathTrimmed, path, minio.PutObjectOptions{})
			if err != nil {
				log.Fatalf("failed to upload files to minio bucket. %v\n", err)
				return fmt.Errorf("failed to upload files to minio bucket. %w", err)

			}
			fmt.Println(ui.Key)
			log.Println(ui.Key)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to get the S3 secret. %v\n", err)
		return fmt.Errorf("failed to get the S3 secret. %w", err)

	}
	return nil
}
