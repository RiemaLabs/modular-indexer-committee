package main

import (
	"log"

	"github.com/RiemaLabs/nubit-da-sdk/constant"
	"github.com/spf13/cobra"
)

type RuntimeArguments struct {
	// EnableService: Provide APIs.
	EnableService bool
	// EnableCommittee: Upload Checkpoints.
	EnableCommittee bool
	// EnableStateRootCache: Store StateRoot as Cache.
	EnableStateRootCache bool
	// EnableTest: Test.
	EnableTest bool
	// BlockHeight: blockheight.
	LatestBlockHeight uint
	// NetWork: Network.
	NetWork string
}

func NewRuntimeArguments() *RuntimeArguments {
	return &RuntimeArguments{NetWork: constant.TestNet}
}

func (arguments *RuntimeArguments) MakeCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use: "Nubit Committee Indexer",
		Short: "Activates the Nubit Committee Indexer with optional services.",
		Long: `
		Committee Indexer command initiates the Committee Indexer process, an essential component of the Modular Indexer architecture. This command offers multiple flags to tailor the indexer's functionality according to the user's needs. The indexer operates on a fully user-verified execution layer for meta-protocols on Bitcoin, leveraging Bitcoin's immutable and decentralized nature to provide a Turing-complete execution layer. 
		
		Flags:
		- "--service/-s": Activates the web service API, allowing the indexer to respond to incoming queries.
		- "--committee": Enables the committee indexer service, which is responsible for reading each block of Bitcoin, calculating protocol states, and summarizing these states.
		- "--cache": Activates the StateRoot cache, improving the efficiency of verkle tree storage and the initialization speed of the indexer. This flag is enabled by default.
		`,
		
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
			if arguments.EnableTest {
				log.Println("Test mode is enabled.")
			} else {
				log.Println("Test mode cache is disabled.")
			}
			log.Println("Network:", arguments.NetWork)
			log.Println("LatestBlockHeight fixed:", arguments.LatestBlockHeight)
		},
	}

	rootCmd.Flags().BoolVarP(&arguments.EnableService, "service", "s", false, "Enable this flag to provide API service")
	rootCmd.Flags().BoolVarP(&arguments.EnableCommittee, "committee", "", false, "Enable this flag to provide committee indexer service")
	rootCmd.Flags().BoolVarP(&arguments.EnableStateRootCache, "cache", "", true, "Enable this flag to cache State Root")
	rootCmd.Flags().BoolVarP(&arguments.EnableTest, "test", "", true, "Enable this flag to hijack the blockheight to test the service")
	rootCmd.Flags().StringVarP(&arguments.NetWork, "network", "", constant.TestNet, "Enable this flag to cache State Root")
	rootCmd.Flags().UintVarP(&arguments.LatestBlockHeight, "blockheight", "b", 781000, "Latest Block Height")

	return rootCmd
}
