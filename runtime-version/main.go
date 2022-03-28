package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/oasisprotocol/oasis-core/go/common"
	"github.com/oasisprotocol/oasis-core/go/common/crypto/signature"
	consensusAPI "github.com/oasisprotocol/oasis-core/go/consensus/api"
	genesisFile "github.com/oasisprotocol/oasis-core/go/genesis/file"
	cmdCommon "github.com/oasisprotocol/oasis-core/go/oasis-node/cmd/common"
	cmdCommonFlags "github.com/oasisprotocol/oasis-core/go/oasis-node/cmd/common/flags"
	cmdGrpc "github.com/oasisprotocol/oasis-core/go/oasis-node/cmd/common/grpc"
	registryAPI "github.com/oasisprotocol/oasis-core/go/registry/api"
	staking "github.com/oasisprotocol/oasis-core/go/staking/api"

	metadataRegistry "github.com/oasisprotocol/metadata-registry-tools"
)

var queryCmd = &cobra.Command{
	Use:   "<runtime-id>",
	Short: "query runtime versions",
	Run:   doQuery,
}

var runtimeID common.Namespace

func doConnect(cmd *cobra.Command) *grpc.ClientConn {
	if err := cmdCommon.Init(); err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}

	conn, err := cmdGrpc.NewClient(cmd)
	if err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}

	return conn
}

func doQuery(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	genesis, err := genesisFile.DefaultFileProvider()
	if err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}
	doc, err := genesis.GetGenesisDocument()
	if err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}
	doc.SetChainContext()

	if len(args) != 1 {
		cmdCommon.EarlyLogAndExit(fmt.Errorf("need exactly one argument (runtimeID)"))
	}
	if err = runtimeID.UnmarshalHex(args[0]); err != nil {
		cmdCommon.EarlyLogAndExit(fmt.Errorf("malformed runtime ID: %s", args[0]))
	}
	conn := doConnect(cmd)

	consensus := consensusAPI.NewConsensusClient(conn)
	reg := registryAPI.NewRegistryClient(conn)

	// Take height from latest block
	blk, err := consensus.GetBlock(ctx, consensusAPI.HeightLatest)
	if err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}
	height := blk.Height

	// Get nodes
	nodes, err := reg.GetNodes(ctx, height)
	if err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}

	entityVersions := make(map[signature.PublicKey]map[string]bool)
	versionCounts := make(map[string]int)
	totalNodes := 0

	for _, node := range nodes {
		for _, runtime := range node.Runtimes {
			if runtime.ID == runtimeID {
				versionString := runtime.Version.String()
				versionCounts[versionString] = versionCounts[versionString] + 1
				totalNodes += 1

				// Store runtime version info for current entity.
				if _, ok := entityVersions[node.EntityID]; !ok {
					entityVersions[node.EntityID] = make(map[string]bool)
				}
				entityVersions[node.EntityID][versionString] = true
			}
		}
	}

	// Node version stats
	fmt.Println("Total nodes:", totalNodes)
	versionKeys := make([]string, 0, len(versionCounts))
	for key, _ := range versionCounts {
		versionKeys = append(versionKeys, key)
	}
	sort.Strings(versionKeys)
	for _, key := range versionKeys {
		fmt.Printf("%s: %d\n", key, versionCounts[key])
	}

	// Entities running latest version
	gp, err := metadataRegistry.NewGitProvider(metadataRegistry.NewGitConfig())

	latestVersion := versionKeys[len(versionKeys)-1]
	updatedEntities := make([]string, 0)
	for entity, versions := range entityVersions {
		if versions[latestVersion] {
			name := "<none>"
			meta, err := gp.GetEntity(ctx, entity)
			if err == nil {
				name = meta.Name
			}
			addr := staking.NewAddress(entity)
			updatedEntities = append(updatedEntities, addr.String()+" "+name)
		}
	}
	sort.Strings(updatedEntities)
	fmt.Printf("\nTotal entities running %s: %d\n", latestVersion, len(updatedEntities))
	for _, entity := range updatedEntities {
		fmt.Println(entity)
	}
}

func main() {
	if err := queryCmd.Execute(); err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}
}

func init() {
	queryCmd.PersistentFlags().AddFlagSet(cmdGrpc.ClientFlags)
	queryCmd.PersistentFlags().AddFlagSet(cmdCommonFlags.GenesisFileFlags)
}
