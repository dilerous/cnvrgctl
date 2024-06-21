package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// grabs the secret, key and endpoing from the cp-object-secret
func GetObjectSecret(api *KubernetesAPI, name string, namespace string) (*ObjectStorage, error) {
	object := ObjectStorage{}

	// Get the Secret
	secret, err := api.Client.CoreV1().Secrets(namespace).Get(context.Background(), name, v1.GetOptions{})
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

// Gathers the name of the pod based on the label and deployment name passed
// TODO: make sense to make a struct for this?
func GetDeployPod(api *KubernetesAPI, targetFlag string, nsFlag string, labelTag string) (string, error) {
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

// get the pod name from the deployment this will be passed to executeBackup function
// used by back and restore commands
func ScaleDeployUp(api *KubernetesAPI, ns string) error {
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

// scales the following deployments "app", "sidekiq", "systemkiq", "searchkiq", "cnvrg-operator" in the namespace specified
// used in back and restore commands
func ScaleDeployDown(api *KubernetesAPI, ns string) error {
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
