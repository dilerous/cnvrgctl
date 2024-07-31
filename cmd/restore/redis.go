/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package restore

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	root "github.com/dilerous/cnvrgctl/cmd"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// redisCmd represents the redis command
var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Restore the Redis database backup.",
	Long: `This command will scale down the application and supporting pods and 
restore the Redis database. 

Examples:

# Restores the Redis database to the Redis pod specified by the namespace.
  cnvrgctl restore redis -n cnvrg`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("redis command called")

		var (
			svcPort     = "6379"
			redisSecret = root.RedisSecret{}
		)

		// target deployment of the postgres backup
		targetFlag, _ := cmd.Flags().GetString("target")

		// grab the namespace from the -n flag if not specified default is used
		redisSecretName, _ := cmd.Flags().GetString("secret-name")

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

		// capture the redis password
		secret, err := root.GetRedisPassword(api, redisSecretName, nsFlag, &redisSecret)
		if err != nil {
			fmt.Printf("error capturing the redis secret, check the namespace and secret exists. %v", err)
			log.Printf("error capturing the redis secret, check the namespace and secret exists. %v", err)
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

		// copy the local redis backup to the redis pod
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

		// restore the redis backup from the dump file
		err = restoreDBBackup(api, nsFlag, podName)
		if err != nil {
			fmt.Printf("error restoring the backup, check the logs. %v ", err)
			log.Printf("error restoring the backup, check the logs. %v", err)
		}

		// update the appendonly value to "no" in the redis creds secret for the value of redis.conf
		err = updateAppendValue(api, secret, nsFlag, redisSecretName)
		if err != nil {
			fmt.Printf("error updating the redis.conf appendonly value in the redis secret. %v ", err)
			log.Printf("error updating the redis.conf appendonly value in the redis secret. %v ", err)
		}

		// delete the redis pod for the appendonly setting to take effect
		err = root.DeletePod(api, nsFlag, podName)
		if err != nil {
			fmt.Printf("error deleting the pod %s, check the logs. %v ", podName, err)
			log.Printf("error deleting the pod %s, check the logs. %v", podName, err)
		}

		// scale up the deployments for app, the kiq pods and the operator
		err = root.ScaleDeployUp(api, nsFlag)
		if err != nil {
			fmt.Printf("there was a problem with scaling up the pods. %v ", err)
			log.Printf("there was a problem with scaling up the pods. %v", err)
		}
	},
}

func init() {
	restoreCmd.AddCommand(redisCmd)

	// flag to define the release name
	redisCmd.Flags().StringP("target", "t", "redis", "Name of redis deployment to retore.")

	// flag to define the app label key
	redisCmd.Flags().StringP("selector", "l", "app", "Define the deployment label for the redis deployment. example: app.kubernetes.io/name")

	// flag to define redis backup file name
	redisCmd.Flags().StringP("file-name", "", "dump.rdb", "Name of the redis backup file.")

	// flag to define restore location
	redisCmd.Flags().StringP("file-location", "f", ".", "Local location of the redis backup file.")

	// flag to define the release name
	redisCmd.Flags().StringP("secret-name", "", "redis-creds", "Define the secret name for the Redis credentials.")
}

// takes the redis.conf and updates appendonly to "no"
// arguments are the "api" for the clientset, "s" for the RedisSecret struct,
// "ns" for namespace and "n" for secret name
func updateAppendValue(api *root.KubernetesAPI, s *root.RedisSecret, ns string, n string) error {

	// set the variables for the secret, namespace, secret name and api
	var (
		redisSecret = s
		clientset   = api.Client
		namespace   = ns
		secretName  = n
		green       = "\033[32m"
		reset       = "\033[0m"
	)

	// decode the redis.conf value so it can be updated
	decodedRedisConf, err := base64.StdEncoding.DecodeString(redisSecret.RedisConf)
	if err != nil {
		log.Printf("failed to decode the base64 value. %v", err)
		return fmt.Errorf("failed to decode Base64 value. %v", err)
	}

	// Convert the byte slice to a string to make changes
	stringRedisConf := string(decodedRedisConf)

	// find append value of "yes" and change to "no"
	updatedRedisConf := strings.Replace(stringRedisConf, "yes", "no", -1)

	// Encode the string to []byte
	encodedRedisConf := []byte(updatedRedisConf)

	// Fetch the existing secret
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, v1.GetOptions{})
	if err != nil {
		log.Printf("error fetching secret. %v\n", err)
		return fmt.Errorf("error fetching secret. %v", err)
	}

	//update the redis.conf key with the encoded string
	secret.Data["redis.conf"] = encodedRedisConf

	// Update the secret in Kubernetes
	_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, v1.UpdateOptions{})
	if err != nil {
		log.Printf("error updating secret. %v\n", err)
		return fmt.Errorf("error updating secret. %v", err)
	}

	// print the secret was successfully updated
	fmt.Println(string(green), "secret was successfully updated!", string(reset))
	log.Println("secret was successfully updated!")

	// no errors return nil
	return nil
}
