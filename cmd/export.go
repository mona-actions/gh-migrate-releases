/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/mona-actions/gh-migrate-releases/pkg/export"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Creates a JSON file of the releases tied to a repository",
	Long:  "Creates a JSON file of the releases tied to a repository",
	Run: func(cmd *cobra.Command, args []string) {
		// Get parameters
		organization := cmd.Flag("organization").Value.String()
		token := cmd.Flag("token").Value.String()
		filePrefix := cmd.Flag("file-prefix").Value.String()
		ghHostname := cmd.Flag("hostname").Value.String()
		repository := cmd.Flag("repository").Value.String()

		if filePrefix == "" {
			filePrefix = fmt.Sprintf("%s-%s", organization, repository)
		}

		// Set ENV variables
		os.Setenv("GHMT_SOURCE_ORGANIZATION", organization)
		os.Setenv("GHMT_SOURCE_TOKEN", token)
		os.Setenv("GHMT_OUTPUT_FILE", filePrefix)
		os.Setenv("GHMT_SOURCE_HOSTNAME", ghHostname)
		os.Setenv("GHMT_REPOSITORY", repository)

		// Bind ENV variables in Viper
		viper.BindEnv("SOURCE_ORGANIZATION")
		viper.BindEnv("SOURCE_TOKEN")
		viper.BindEnv("OUTPUT_FILE")
		viper.BindEnv("SOURCE_HOSTNAME")
		viper.BindEnv("REPOSITORY")

		// Call exportCSV
		export.CreateJSONs()
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	// Flags
	exportCmd.Flags().StringP("organization", "o", "", "Organization of the repository")
	exportCmd.MarkFlagRequired("organization")

	exportCmd.Flags().StringP("token", "t", "", "GitHub token")
	exportCmd.MarkFlagRequired("token")

	exportCmd.Flags().StringP("repository", "r", "", "repository to export")
	exportCmd.MarkFlagRequired("repository")

	exportCmd.Flags().StringP("file-prefix", "f", "", "Output filenames prefix")

	exportCmd.Flags().StringP("hostname", "u", "", "GitHub Enterprise hostname url (optional) Ex. github.example.com")

}
