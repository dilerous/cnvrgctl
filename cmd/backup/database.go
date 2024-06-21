/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package backup

import (
	"fmt"

	"github.com/dilerous/cnvrgctl/cmd"
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
		fmt.Println("database called")

		//	cmd.ConnectToK8s
	},
}

func init() {
	cmd.RootCmd.AddCommand(databaseCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// databaseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// databaseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
