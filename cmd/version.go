/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// Set the version of the cnvrgctl cli
var Version = "v0.0.4"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Displays the current version of the cnvrgctl cli tool",
	Long: `Shows the current version of the cli tool. This will be expanded to show the 
cnvrg.io app, operator and Kubernetes version at a later date.

Usage:
  cnvrgctl version [flags]

Examples:

# Display the current cli version.
  cnvrgctl version`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("version command called")

		// display and log the version of the cli tool.
		displayVersion(Version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}

func displayVersion(v string) {
	fmt.Println("cnvrgctl version " + v)
	log.Println("cnvrgctl version " + v)
}
