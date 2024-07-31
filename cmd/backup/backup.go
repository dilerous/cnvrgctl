/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	root "github.com/dilerous/cnvrgctl/cmd"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backs up the cnvrg.io postgres database or files",
	Long: `Backup either the postgres database or the files used by cnvrg.io.

Examples:

# Backups the default postgres database cnvrg namespace.
  cnvrgctl backup postgres -n cnvrg

# Specify namespace, deployment label key, and deployment name.
  cnvrgctl backup postgres --target postgres-ha --label app.kubernetes.io/name -n cnvrg
  
# Backups the default object storage bucket in the cnvrg namespace.
  cnvrgctl backup files -n cnvrg`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("called the backup command")
	},
}

func init() {
	root.RootCmd.AddCommand(backupCmd)
}

// copies the backup file from the pod to the local machine
// Function takes the namespace "ns", the pod name "p", local file location "l" and the file name "f"
func copyDBLocally(api *root.KubernetesAPI, ns string, p string, l string, f string) (bool, error) {
	log.Println("copyDBLocally function called.")

	//TODO: add flag to specify location of file
	var ( // Set the pod and namespace
		podName    = p
		namespace  = ns
		filePath   = l + "/"
		backupFile = f
		clientset  = api.Client
		command    = []string{"cat", backupFile}
		config     = api.Config
	)

	// check if the backup file is the redis database, set the correct path
	if f == "dump.rdb" {
		command = []string{"cat", "/data/" + f}
	}

	// If the file path is not the local directory, create the directory
	if filePath != "./" {
		err := createDirectory(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating the directory. %v", err)
			log.Printf("error creating the directory. %v\n", err)
		}
	}

	// Create a REST client
	log.Println("Creating the rest client call")
	req := clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: command,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		}, scheme.ParameterCodec)

	// execute the command
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		log.Printf("error executing the remote command. %v\n", err)
		return false, fmt.Errorf("error execuuting the remote command. %w", err)
	}

	// set the variables to type byte and stream the output to those variables
	var stdout, stderr bytes.Buffer

	// execute the command
	exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	// Create a local file to write to
	localFile, err := os.Create(filePath + backupFile)
	if err != nil {
		log.Printf("error creating local file. %v\n", err)
		return false, fmt.Errorf("error creating local file. %w", err)
	}
	defer localFile.Close()

	// open the file that was just created
	file, err := os.OpenFile(filePath+backupFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("opening the file failed. %s\n", err)
		return false, fmt.Errorf("opening the file failed. %w", err)
	}
	defer file.Close()

	// copy the stream output from the cat command to the file.
	_, err = io.Copy(file, &stdout)
	if err != nil {
		log.Printf("the copy failed. %v", err)
		return false, fmt.Errorf("the copy failed. %w", err)
	}

	return true, nil
}

// create a directory
func createDirectory(dirName string) error {

	// Create a directory and any necessary parents
	err := os.MkdirAll(dirName, 0755) // 0755 is the permission mode
	if err != nil {
		log.Printf("failed to create directory. %v", err)
		return fmt.Errorf("failed to create directory. %w", err)
	} else {
		log.Printf("directorie(s) created successfully. %v", err)
		fmt.Println("directorie(s) created successfully.")
	}
	return nil
}
