/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	cfgFile string
	//clientset *kubernetes.Clientset
)

type KubernetesAPI struct {
	Suffix string
	Client kubernetes.Interface
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cnvrgctl",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cnvrgctl.yaml)")

	// Persistent flag to define the namespace
	rootCmd.PersistentFlags().StringP("namespace", "n", "default", "If present, the namespace scope for this CLI request")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// Persistent flag for setting the kubeconfig
	rootCmd.PersistentFlags().StringP("kubeconfig", "", "", "Path to the kubeconfig file to use for CLI requests")

	// Persistent flag for setting the context
	rootCmd.PersistentFlags().StringP("context", "", "", "The name of the kubeconfig context to use")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cnvrgctl" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cnvrgctl")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func connectToK8s() (*KubernetesAPI, error) {

	kubeContextFlag, err := rootCmd.Flags().GetString("context")
	if err != nil {
		return nil, fmt.Errorf("error reading the kubeconfig context. %w", err)
	}

	kubeConfigFlag, err := rootCmd.Flags().GetString("kubeconfig")
	if err != nil {
		return nil, fmt.Errorf("error getting the kubeconfig path. %w", err)
	}

	// If KUBECONFIG is not set, use default path
	//TODO look at making this a case statement
	envKubeConfig := os.Getenv("KUBECONFIG")
	kubeconfig := kubeConfigFlag

	if kubeConfigFlag == "" {
		kubeconfig = envKubeConfig
		if kubeconfig == "" {
			kubeconfig = homeDir() + "/.kube/config"
		}
	}

	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		// If building config fails, try in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("error building the kubeconfig. %w", err)
		}
	}

	// Use context inputed by context flag
	if kubeConfigFlag != "" {
		config, err = buildConfigWithContextFromFlags(kubeContextFlag, kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("the context doesn't exists. %w", err)
		}
	}

	// Create Kubernetes client
	client := KubernetesAPI{}
	client.Client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating kubernetes client, exiting. %w", err)
	}
	return &client, nil
}

// Gets the home env variable for linux/windows
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // Windows
}

// Build the clientconfig when a context is specified.
func buildConfigWithContextFromFlags(context string, kubeconfigPath string) (*rest.Config, error) {
	fmt.Println(kubeconfigPath)
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}
