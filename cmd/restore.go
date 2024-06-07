/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore file backups to target bucket.",
	Long: `This command will restore file backups to a bucket you specify. By default the credentials,
bucket, and keys will be gathered from the cp-object-storage secret for the restore. You can manually
specify these values using flags.

Examples:
	
# Restore the backups to the bucket 'cnvrg-backups'.
  cnvrgctl migrate restore -a minio -k minio123 -u minio.aws.dilerous.cloud -b cnvrg-backups`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	migrateCmd.AddCommand(restoreCmd)
}
