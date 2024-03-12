package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

type RuntimeArguments struct {
	EnableService bool
}

func NewRuntimeArguments() *RuntimeArguments {
	return &RuntimeArguments{}
}

func (arguments *RuntimeArguments) MakeCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use: "Nubit Committee Indexer",
		// TODO: Add descriptions.
		Short: "",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			if arguments.EnableService {
				fmt.Println("Service mode is enabled. Starting to provide service...")
				// Add your service logic here
			} else {
				fmt.Println("Service mode is disabled.")
			}
		},
	}

	rootCmd.Flags().BoolVarP(&arguments.EnableService, "service", "s", false, "Enable this flag to start providing service")

	return rootCmd
}
