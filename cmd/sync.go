/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/mona-actions/gh-migrate-releases/pkg/sync"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// syncCmd represents the export command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Recreates releases, membership, and team repo roles from a source organization to a target organization",
	Long:  "Recreates releases, membership, and team repo roles from a source organization to a target organization",
	Run: func(cmd *cobra.Command, args []string) {
		// Get parameters
		sourceOrganization := cmd.Flag("source-organization").Value.String()
		targetOrganization := cmd.Flag("target-organization").Value.String()
		sourceToken := cmd.Flag("source-token").Value.String()
		targetToken := cmd.Flag("target-token").Value.String()
		ghHostname := cmd.Flag("source-hostname").Value.String()
		repository := cmd.Flag("repository").Value.String()

		// Set ENV variables
		os.Setenv("GHMT_SOURCE_ORGANIZATION", sourceOrganization)
		os.Setenv("GHMT_TARGET_ORGANIZATION", targetOrganization)
		os.Setenv("GHMT_SOURCE_TOKEN", sourceToken)
		os.Setenv("GHMT_TARGET_TOKEN", targetToken)
		os.Setenv("GHMT_SOURCE_HOSTNAME", ghHostname)
		os.Setenv("GHMT_REPOSITORY", repository)

		// Bind ENV variables in Viper
		viper.BindEnv("SOURCE_ORGANIZATION")
		viper.BindEnv("TARGET_ORGANIZATION")
		viper.BindEnv("SOURCE_TOKEN")
		viper.BindEnv("TARGET_TOKEN")
		viper.BindEnv("SOURCE_HOSTNAME")
		viper.BindEnv("REPOSITORY")
		// Call syncreleases
		sync.SyncReleases()
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Flags
	syncCmd.Flags().StringP("source-organization", "s", "", "Source Organization to sync releases from")
	syncCmd.MarkFlagRequired("source-organization")

	syncCmd.Flags().StringP("target-organization", "t", "", "Target Organization to sync releases from")
	syncCmd.MarkFlagRequired("target-organization")

	syncCmd.Flags().StringP("source-token", "a", "", "Source Organization GitHub token. Scopes: read:org, read:user, user:email")
	syncCmd.MarkFlagRequired("source-token")

	syncCmd.Flags().StringP("target-token", "b", "", "Target Organization GitHub token. Scopes: admin:org")
	syncCmd.MarkFlagRequired("target-token")

	syncCmd.Flags().StringP("repository", "r", "", "repository to export/import releases from/to")
	syncCmd.MarkFlagRequired("repository")

	syncCmd.Flags().StringP("source-hostname", "u", "", "GitHub Enterprise source hostname url (optional) Ex. https://github.example.com")

}
