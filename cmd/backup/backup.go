/*
Copyright © 2024 NAME HERE BRADLEY.SOPER@CNVRG.IO
*/
package backup

import (
	"log"

	root "github.com/dilerous/cnvrgctl/cmd"
	"github.com/spf13/cobra"
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
