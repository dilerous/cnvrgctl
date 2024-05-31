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
		log.Println("restore called")
		o := ObjectStorage{}

		// grab the namespace from the -n flag if not specified default is used
		s3SecretName, _ := cmd.Flags().GetString("secret-name")

		// grab the namespace from the -n flag if not specified default is used
		nsFlag, _ := cmd.Flags().GetString("namespace")

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

	},
}

func init() {
	migrateCmd.AddCommand(restoreCmd)

	// flag to define the secret for the object storage credentials
	restoreCmd.Flags().StringP("secret-name", "", "cp-object-storage", "define the secret name for the S3 bucket credentials.")

	// flag to define the secret for the object storage credentials
	restoreCmd.Flags().StringP("secret-key", "s", "cp-object-storage", "define the secret key for the S3 bucket credentials.")

	// flag to define the secret for the object storage credentials
	restoreCmd.Flags().StringP("access-key", "a", "cp-object-storage", "define the access key for the S3 bucket credentials.")
}

// TODO: check if useSSL = false, conslidate with get bucket function
func uploadFilesMinio(o *ObjectStorage) error {
	// Walk all the subdirectories of a directory
	dir := "./cnvrg-storage"
	useSSL := false

	//TODO: remove this section is already used in connectToMinio
	uWithoutHttp := strings.Replace(o.Endpoint, "http://", "", 1)
	fmt.Println(uWithoutHttp)

	minioClient, err := minio.New(uWithoutHttp, &minio.Options{
		Creds:  credentials.NewStaticV4(o.AccessKey, o.SecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("failed to get the S3 secret. %v\n", err)
		return fmt.Errorf("failed to get the S3 secret. %w", err)
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return err
		}

		// Upload all files
		ui, err := minioClient.FPutObject(context.TODO(), o.BucketName, path, path, minio.PutObjectOptions{})
		if err != nil {
			log.Fatalf("failed to get the S3 secret. %v\n", err)
			return fmt.Errorf("failed to get the S3 secret. %w", err)

		}
		fmt.Printf("info.Name = %v\n", info.Name())
		fmt.Printf("Path = %v\n", path)
		fmt.Printf(ui.Key)
		return nil
	})
	if err != nil {
		log.Fatalf("failed to get the S3 secret. %v\n", err)
		return fmt.Errorf("failed to get the S3 secret. %w", err)

	}
	return nil
}
