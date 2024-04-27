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
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var clientset *kubernetes.Clientset

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "The command will grab logs from all running pods in the namespace defined",
	Long: `Capture the logs for every container in a specified namespace and save the 
files to ./logs/<pod-name>.txt. 
	
Examples:
  # Gather all container logs in the cnvrg namespace.
  cnvrgctl -n cnvrg logs 

  # Gather all container logs and tar in the a .tar files
  cnvrgctl -n cnvrg logs --tar`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("logs called")

		// testing how to add a local flag
		tarFlag, _ := cmd.Flags().GetBool("tar")
		if tarFlag {
			createTar()
		}

		// testing how to pass a namespace from the flag
		ns, _ := cmd.Flags().GetString("namespace")
		fmt.Printf("The namespace that was defined is %s", ns)

		// calls connect function to set the clientset for kubectl access
		connectToK8s()

		// return a list all pods in the cnvrg namespace
		// TODO: add a flag to select the namespace
		podList, _ := getPods(ns)

		// takes the podlist and gathers logs for each pod and saves to txt file
		getLogs(podList)
	},
}

func init() {
	// Adds the log command to the cli tool
	rootCmd.AddCommand(logsCmd)

	// Adds the flag -t --tar to the logs command this is local
	logsCmd.Flags().BoolP("tar", "t", false, "tarball the log files")
}

func connectToK8s() {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		// If KUBECONFIG is not set, use default path
		if home := homeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		// If building config fails, try in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error building the kubeconfig, exiting %v\n", err)
			os.Exit(1)
		}
	}

	// Create Kubernetes client
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating kubernetes client, exiting %v\n", err)
		os.Exit(1)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // Windows
}

func getPods(ns string) ([]corev1.Pod, error) {
	// List Pods
	pods, err := clientset.CoreV1().Pods(ns).List(context.Background(), v1.ListOptions{})
	if err != nil {
		//		fmt.Fprintf(os.Stderr, "error listing pods: %v\n", err)
		return nil, fmt.Errorf("error getting list of pods check connectivity. %w", err)
	}

	// Print and return the pod name and namespace
	fmt.Println("Pods:")
	for _, pod := range pods.Items {
		fmt.Printf("Namespace: %s, Name: %s\n", pod.Namespace, pod.Name)
	}
	return pods.Items, nil
}

func getLogs(pods []corev1.Pod) {
	fmt.Println("Pod Logs:")

	tailLines := int64(1)

	logsPath := "./logs"
	err := os.MkdirAll(logsPath, 0755)
	if err != nil {
		fmt.Println("error creating folder:", err)
		return
	}

	err = os.Chdir(logsPath)
	if err != nil {
		fmt.Println("error changing directory:", err)
		return
	}

	for _, pod := range pods {
		time.Sleep(2 * time.Second)
		podLogs, err := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{TailLines: &tailLines}).DoRaw(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting logs for pod %s: %v\n", pod.Name, err)
			continue
		}

		file, err := os.Create(pod.Name + ".txt")
		if err != nil {
			log.Fatalf("error creating file: %v", err)
		}
		defer file.Close()

		fmt.Printf("Pod: %s\n%s\n", pod.Name, string(podLogs))
		fmt.Fprint(file, string(podLogs))
		file.Close()
	}
}

func createTar() {
	fmt.Println("You called the tar flag")
}
