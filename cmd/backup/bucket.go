/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package backup

import (
	"log"

	"github.com/spf13/cobra"
)

// bucketCmd represents the bucket command
var bucketCmd = &cobra.Command{
	Use:   "bucket",
	Short: "Backup the files in the cnvrg.io storage bucket",
	Long: `The command will backup the object storage bucket used to host the files and 
datasets in the cnvrg environment.

Examples:

# Backups the files stored in the bucket in the cnvrg namespace.
  cnvrgctl backup bucket -n cnvrg`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("bucket called")
	},
}

func init() {
	backupCmd.AddCommand(bucketCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// bucketCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// bucketCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
