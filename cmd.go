package main

import (
	"log"

	"github.com/spf13/cobra"
)

type RuntimeArguments struct {
	// EnableService: Provide APIs.
	EnableService bool
	// EnableCommittee: Upload Checkpoints.
	EnableCommittee bool
	// EnableStateRootCache: Store StateRoot as Cache.
	EnableStateRootCache bool
}

func NewRuntimeArguments() *RuntimeArguments {
	return &RuntimeArguments{}
}

func (arguments *RuntimeArguments) MakeCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use: "Nubit Committee Indexer",
		// TODO: Urgent. Add descriptions.
		Short: "",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			if arguments.EnableService {
				log.Println("Service mode is enabled.")
			} else {
				log.Println("Service mode is disabled.")
			}
			if arguments.EnableCommittee {
				log.Println("Committee mode is enabled.")
			} else {
				log.Println("Committee mode is disabled.")
			}
			if arguments.EnableStateRootCache {
				log.Println("StateRoot cache is enabled.")
			} else {
				log.Println("StateRoot cache is disabled.")
			}
		},
	}

	rootCmd.Flags().BoolVarP(&arguments.EnableService, "service", "s", false, "Enable this flag to provide API service")
	rootCmd.Flags().BoolVarP(&arguments.EnableCommittee, "committee", "", false, "Enable this flag to provide committee indexer service")
	rootCmd.Flags().BoolVarP(&arguments.EnableStateRootCache, "cache", "", true, "Enable this flag to cache State Root")

	return rootCmd
}
