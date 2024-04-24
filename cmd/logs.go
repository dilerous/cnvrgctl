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
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("logs called")
		connectToK8s()
		podList := getPods("cnvrg")
		getLogs(podList)
		//		for _, pod := range podList {
		//			fmt.Println(pod.Name)
		//		}
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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
			fmt.Fprintf(os.Stderr, "error building kubeconfig: %v\n", err)
			os.Exit(1)
		}
	}

	// Create Kubernetes client
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating kubernetes client: %v\n", err)
		os.Exit(1)
	}

}

func getPods(ns string) []corev1.Pod {
	// List Pods
	pods, err := clientset.CoreV1().Pods(ns).List(context.Background(), v1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing pods: %v\n", err)
		os.Exit(1)
	}

	// Print and return the pod name and namespace
	fmt.Println("Pods:")
	for _, pod := range pods.Items {
		fmt.Printf("Namespace: %s, Name: %s\n", pod.Namespace, pod.Name)
	}
	return pods.Items
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // Windows
}

func getLogs(pods []corev1.Pod) {
	fmt.Println("Pod Logs:")

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
		podLogs, err := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).DoRaw(context.Background())
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

/*
func createLogs(pods []corev1.Pod) {
	fmt.Println("Capturing logs in a text file:")
	for _, pod := range pods {
		file, err := os.Create(pod.Name + ".txt")
		if err != nil {
			log.Fatalf("Error creating file: %v", err)
		}
	}
	defer file.Close()
}*/
