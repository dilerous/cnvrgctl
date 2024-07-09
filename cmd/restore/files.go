/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package restore

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	root "github.com/dilerous/cnvrgctl/cmd"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
)

// filesCmd represents the files command
var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "restore files to the cnvrg-storage bucket",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("restore command called")

		//define the empty object struct
		o := root.ObjectStorage{}

		// set success to false until backup completes successfully
		success := false

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

		// define the source folder to backup too
		sourceFlag, _ := cmd.Flags().GetString("source")

		// set the secret key if defined
		skFlag, _ := cmd.Flags().GetString("secret-key")
		if skFlag != "" {
			o.SecretKey = skFlag
		}

		// set the access key if defined
		akFlag, _ := cmd.Flags().GetString("access-key")
		if akFlag != "" {
			o.AccessKey = akFlag
		}

		// connect to kubernetes and define clientset and rest client
		api, err := root.ConnectToK8s()
		if err != nil {
			fmt.Printf("error connecting to the cluster, check your connectivity. %v", err)
			log.Printf("error connecting to the cluster, check your connectivity. %v", err)
		}
		// get the object data and store in the ObjectStorage struct
		if skFlag == "" && akFlag == "" {
			objectData, err := root.GetObjectSecret(api, s3SecretName, nsFlag)
			if err != nil {
				fmt.Printf("failed to get the S3 secret. %v ", err)
				log.Printf("failed to get the S3 secret. %v", err)
			}
			success, err = uploadFilesMinio(objectData, sourceFlag)
			if err != nil {
				log.Printf("failed to upload files. %v", err)
				fmt.Printf("failed to upload files. %v ", err)
			}
		}

		// upload files from minio using info from the flags
		if !success {
			_, err = uploadFilesMinio(&o, sourceFlag)
			if err != nil {
				log.Printf("failed to upload files. %v", err)
				fmt.Printf("failed to upload files. %v ", err)
			}
		}
	},
}

func init() {
	restoreCmd.AddCommand(filesCmd)

	// flag to define the secret for the object storage credentials
	filesCmd.Flags().StringP("secret-name", "", "cp-object-storage", "define the Kubernetes secret name for the S3 bucket credentials.")

	// flag to define the secret for the object storage credentials
	filesCmd.Flags().StringP("secret-key", "k", "", "define the secret key for the S3 bucket credentials. (required if bucket, access-key and minio-url is set)")

	// flag to define the secret for the object storage credentials
	filesCmd.Flags().StringP("access-key", "a", "", "define the access key for the S3 bucket credentials. (required if secret-key, bucket and minio-url is set)")

	// flag to define the backup bucket target
	filesCmd.Flags().StringP("bucket", "b", "cnvrg-storage", "define the bucket to restore the files too. (required if secret-key, access-key and minio-url is set)")

	// flag to define the backup bucket target
	filesCmd.Flags().StringP("minio-url", "u", "", "define the url to the minio api.(required if secret-key, access-key and bucket is set)")

	// flag to define the source files
	filesCmd.Flags().StringP("source", "s", "cnvrg-storage", "define the source folder to restore.")

	// if any of the flags defined are set, they all must be set
	filesCmd.MarkFlagsRequiredTogether("secret-key", "access-key", "bucket", "minio-url")
}

// TODO: check if useSSL = false, conslidate with get bucket function
// TODO: add in getting cp-object-secret
func uploadFilesMinio(o *root.ObjectStorage, s string) (bool, error) {
	log.Println("uploadFiles Minio function called.")
	// Walk all the subdirectories of a directory
	dir := s
	useSSL := false

	//remove https from the endpoint url
	url := strings.TrimPrefix(o.Endpoint, "https://")
	url = strings.TrimPrefix(url, "http://")
	log.Println(url)

	minioClient, err := minio.New(url, &minio.Options{
		Creds:  credentials.NewStaticV4(o.AccessKey, o.SecretKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		log.Printf("failed to configure minio client. %v", err)
		return false, fmt.Errorf("failed to configure minio client. %w", err)
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("unable to walk the file path. %v\n", err)
			return fmt.Errorf("unable to walk the file path. %w", err)
		}

		if !info.IsDir() {
			// trim the bucket name from the path
			pathTrimmed := strings.TrimPrefix(path, dir)
			log.Println("The path trimmed is: ", pathTrimmed)

			// Upload all files
			ui, err := minioClient.FPutObject(context.Background(), o.BucketName, pathTrimmed, path, minio.PutObjectOptions{})
			if err != nil {
				log.Printf("failed to upload files to minio bucket. %v\n", err)
				return fmt.Errorf("failed to upload files to minio bucket. %w", err)

			}
			fmt.Println("file " + ui.Key + " was uploaded successfully.")
			log.Println("file " + ui.Key + " was uploaded successfully.")
		}
		return nil
	})
	if err != nil {
		log.Printf("failed to get the S3 secret. %v", err)
		return false, fmt.Errorf("failed to get the S3 secret. %w", err)
	}
	fmt.Println("Files uploaded successfully!")
	return true, nil
}
