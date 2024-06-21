/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package backup

import (
	"log"

	root "github.com/dilerous/cnvrgctl/cmd"
	"github.com/spf13/cobra"
)

// databaseCmd represents the database command
var databaseCmd = &cobra.Command{
	Use:   "database",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("database called")
		root.ConnectToK8s()
	},
}

func init() {
	root.RootCmd.AddCommand(databaseCmd)
}

/*
func NewCmdDatabase() *cobra.Command {
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
*/
