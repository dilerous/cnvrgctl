/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package backup

import (
	"fmt"
	"log"

	"github.com/dilerous/cnvrgctl/cmd"
	"github.com/spf13/cobra"
)

// bucketCmd represents the bucket command
var bucketCmd = &cobra.Command{
	Use:   "bucket",
	Short: "Backup the S3 bucket",
	Long: `The command will backup the bucket used to host the files and 
datasets in the cnvrg environment. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("bucket called")
	},
}

func init() {
	cmd.RootCmd.AddCommand(bucketCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// bucketCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// bucketCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func NewCmdBucket() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subcommand1",
		Short: "This is subcommand1",
		Long:  "This is a longer description for subcommand1",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Executing subcommand1")
		},
	}

	// Add flags, arguments, etc., specific to subcommand1 if needed

	return cmd
}
