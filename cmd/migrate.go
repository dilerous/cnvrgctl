/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ObjectStorage struct {
	AccessKey  string
	SecretKey  string
	Region     string
	Endpoint   string
	Type       string
	BucketName string
	Namespace  string
}

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Command to manage cnvrg.io installation migrations",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("migrate called")
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// migrateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// migrateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// grabs the secret, key and endpoing from the cp-object-secret
func getObjectSecret(api *KubernetesAPI, name string, namespace string) (*ObjectStorage, error) {
	object := ObjectStorage{}

	// Get the Secret
	secret, err := api.Client.CoreV1().Secrets(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		log.Printf("error getting the secret, does it exist? %v", err)
		return &object, fmt.Errorf("error getting the secret, does it exist? %w ", err)
	}

	// Get the Secret data
	endpoint, ok := secret.Data["CNVRG_STORAGE_ENDPOINT"]
	object.Endpoint = string(endpoint)
	if !ok {
		log.Printf("error getting the key CNVRG_STORAGE_ENDPOINT, does it exist? %v", err)
		return nil, fmt.Errorf("error getting the key CNVRG_STORAGE_ENDPOINT, does it exist? %w ", err)
	}

	key, ok := secret.Data["CNVRG_STORAGE_ACCESS_KEY"]
	object.AccessKey = string(key)
	if !ok {
		log.Printf("error getting the key CNVRG_STORAGE_ACCESS_KEY, does it exist? %v", err)
		return nil, fmt.Errorf("error getting the key CNVRG_STORAGE_ACCESS_KEY, does it exist? %w ", err)
	}

	secretKey, ok := secret.Data["CNVRG_STORAGE_SECRET_KEY"]
	object.SecretKey = string(secretKey)
	if !ok {
		log.Printf("error getting the key CNVRG_STORAGE_SECRET_KEY, does it exist? %v", err)
		return nil, fmt.Errorf("error getting the key CNVRG_STORAGE_SECRET_KEY, does it exist? %w ", err)
	}

	region, ok := secret.Data["CNVRG_STORAGE_REGION"]
	object.Region = string(region)
	if !ok {
		log.Printf("error getting the key CNVRG_STORAGE_REGION, does it exist? %v", err)
		return nil, fmt.Errorf("error getting the key CNVRG_STORAGE_REGION, does it exist? %w ", err)
	}

	storageType, ok := secret.Data["CNVRG_STORAGE_TYPE"]
	object.Type = string(storageType)
	if !ok {
		log.Printf("error getting the key CNVRG_STORAGE_TYPE, does it exist? %v", err)
		return nil, fmt.Errorf("error getting the key CNVRG_STORAGE_TYPE, does it exist? %w ", err)
	}

	bucketName, ok := secret.Data["CNVRG_STORAGE_BUCKET"]
	object.BucketName = string(bucketName)
	if !ok {
		log.Printf("error getting the key CNVRG_STORAGE_BUCKET, does it exist? %v", err)
		return nil, fmt.Errorf("error getting the key CNVRG_STORAGE_BUCKET, does it exist? %w ", err)
	}

	return &object, nil
}

// connect to minio storage
// TODO Create generic and move to migrate
func connectToMinio(o *ObjectStorage) error {
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
