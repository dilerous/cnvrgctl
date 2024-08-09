/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// TODO: roll cfgFile into the KubernetesAPI struct
var (
	cfgFile       string
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
)

// rootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "cnvrgctl",
	Version: Version,
	Short:   "cnvrg.io delivery tool for cnvrg deployment maintenance.",
	Long: `cnvrg.io delivery tool for cnvrg deployment maintenance. The 
cnvrg.io cli tool can be used to backup, migrate and install cnvrg.io. The tool
also can be used to migrate exisiting deployments or install cnvrg.io classic.

Examples:

# Backups the default postgres database and files in the cnvrg namespace.
  cnvrgctl backup postgres -n cnvrg
  
# Backups the minio bucket in the cnvrg namespace.
  cnvrgctl backup files -n cnvrg
  
# Grab pod logs from the cnvrg namespace.
  cnvrgctl logs -n cnvrg`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cnvrgctl.yaml)")

	// Persistent flag to define the namespace
	RootCmd.PersistentFlags().StringP("namespace", "n", "default", "If present, the namespace scope for this CLI request")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// Persistent flag for setting the kubeconfig
	RootCmd.PersistentFlags().StringP("kubeconfig", "", "", "Path to the kubeconfig file to use for CLI requests")

	// Persistent flag for setting the context
	RootCmd.PersistentFlags().StringP("context", "", "", "The name of the kubeconfig context to use")

	// Start the logging for the cli
	err := setLogger()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error configuring the logger.")
		log.Printf("error configuring the logger. %v ", err)
	}
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

// Build the client config when a context is specified.
func buildConfigWithContextFromFlags(context string, kubeconfigPath string) (*rest.Config, error) {
	fmt.Println(kubeconfigPath)
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}

// TODO add flag to define logs file and path
func setLogger() error {
	LOG_FILE_PATH := "cnvrgctl-logs.txt"

	file, err := os.OpenFile(LOG_FILE_PATH, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
		return fmt.Errorf("there was an issue creating the log file. %v", err)
	}
	log.SetOutput(file)
	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	return nil
}
