/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Pull logs from all running pods in the namespace defined",
	Long: `Capture the logs for every container in a specified namespace and save the 
files to ./<log-dir>/<pod-name>.txt. 

Usage:
  cnvrgctl logs [flags]
	
Examples:
  # Gather all container logs in the cnvrg namespace.
  cnvrgctl -n cnvrg logs 

  # Gather all container logs in the cnvrg namespace and select the last 10 lines.
  cnvrgctl -n cnvrg logs -l=10

  # Gather all container logs and tar the log files into a tar.gz
  cnvrgctl -n cnvrg logs --tar
  
  # Gather all container logs and specify the directory the files are saved to.
  cnvrgctl -n cnvrg logs --log-dir=my-log-folder `,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("called the logs command.")

		// Pass a namespace to the logs command
		ns, _ := cmd.Flags().GetString("namespace")

		// Pass a namespace to the logs command
		logDir, _ := cmd.Flags().GetString("log-dir")

		// Pass the number of lines to gather when grabbing logs
		lines, _ := cmd.Flags().GetInt("lines")

		// calls connect function to set the clientset for kubectl access
		api, err := ConnectToK8s()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error connecting to cluster, check your connectivity. %v", err)
		}

		// return a list all pods in the cnvrg namespace
		podList, err := getPods(ns, api.Client)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting the list of pods. %v", err)
			return
		}

		// takes the podlist and gathers logs for each pod and saves to txt file
		err = getLogs(podList, lines, logDir, api.Client)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error gathering logs. %v", err)
		}

		// Tars the log files
		tarFlag, _ := cmd.Flags().GetBool("tar")
		if tarFlag {
			err = createTar()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error creating the tar file. %v", err)
			}
		}

	},
}

func init() {
	// Adds the log command to the cli tool
	RootCmd.AddCommand(logsCmd)

	// Adds the flag -t --tar to the logs command this is local
	logsCmd.Flags().BoolP("tar", "t", false, "Tarball the log files")

	// Add the flag -n --number to select the number of logs to grab
	logsCmd.Flags().IntP("lines", "l", 100, "Define the number of lines in the log to return")

	// Add the flag --log-dir to define the log directory
	logsCmd.PersistentFlags().StringP("log-dir", "", "./logs", "Define the directory logs are saved too.")
}

func getPods(ns string, clientset kubernetes.Interface) ([]corev1.Pod, error) {
	// List Pods
	pods, err := clientset.CoreV1().Pods(ns).List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting the list of pods failed. %w", err)
	}

	// Print and return the pod name and namespace
	fmt.Println("Pods: ")
	for _, pod := range pods.Items {
		fmt.Printf("pod name: %s, namespace: %s\n", pod.Name, pod.Namespace)
	}
	return pods.Items, nil
}

func getLogs(pods []corev1.Pod, num int, logdir string, clientset kubernetes.Interface) error {
	fmt.Println("Grabbing the following pod logs:")

	tailLines := int64(num)

	logsPath := logdir
	err := os.MkdirAll(logsPath, 0755)
	if err != nil {
		return fmt.Errorf("error creating the folder. %w", err)
	}

	// change to the logs directory
	err = os.Chdir(logsPath)
	if err != nil {
		return fmt.Errorf("error changing directory. %w", err)
	}

	for _, pod := range pods {
		time.Sleep(1 * time.Second)
		podLogs, err := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{TailLines: &tailLines}).DoRaw(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting logs for pod %s: %v\n", pod.Name, err)
			continue
		}

		file, err := os.Create(pod.Name + ".txt")
		if err != nil {
			return fmt.Errorf("error creating the file. %w", err)
		}
		defer file.Close()

		fmt.Printf("Pod: %s\n%s\n", pod.Name, string(podLogs))
		fmt.Fprint(file, string(podLogs))
		file.Close()
	}

	// changing back to the root directory
	err = os.Chdir("../")
	if err != nil {
		return fmt.Errorf("error changing directory to root. %w", err)
	}
	return nil
}

// TODO: create a flag that lets you define the folder the files live in; default ./logs
func createTar() error {
	log.Println("You called the tar flag")

	tarFile := "logs.tar.gz"
	dir := "./"

	err := createTarGz(dir, tarFile)
	if err != nil {
		return fmt.Errorf("error creating the tar file, %v. %w", tarFile, err)
	}

	fmt.Printf("tar file created successfully, %v\n", tarFile)
	return nil
}

func createTarGz(source string, target string) error {
	// Create the target file logs.tar.gz
	file, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("file creation failed, the name of the file is %v. %w", target, err)
	}
	defer file.Close()

	// Create a gzip writer
	gzwFile := gzip.NewWriter(file)
	defer gzwFile.Close()

	// Create a new tar writer
	twFile := tar.NewWriter(gzwFile)
	defer twFile.Close()

	// Walk through the source directory
	err = filepath.Walk(source, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking the file path. %w", err)
		}

		// Only include .txt files
		if !strings.HasSuffix(fi.Name(), ".txt") {
			return nil
		}

		//TODO: wy is it opening file not tarfile
		// Open the file
		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("error opening the file %v. %w", file, err)
		}
		defer f.Close()

		// Create a new tar header
		hdr := &tar.Header{
			Name:    strings.TrimPrefix(file, source+"/"),
			Size:    fi.Size(),
			Mode:    int64(fi.Mode()),
			ModTime: fi.ModTime(),
		}

		// Write the header to the tar archive
		_ = twFile.WriteHeader(hdr)

		// Copy the file contents to the tar archive
		if _, err := io.Copy(twFile, f); err != nil {
			return fmt.Errorf("error copying the file contents to the tar archive. %w", err)
		}

		return nil
	})
	return err
}
