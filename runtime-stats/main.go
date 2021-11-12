package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/oasisprotocol/oasis-core/go/common"
	"github.com/oasisprotocol/oasis-core/go/common/cbor"
	"github.com/oasisprotocol/oasis-core/go/common/crypto/signature"
	consensusAPI "github.com/oasisprotocol/oasis-core/go/consensus/api"
	"github.com/oasisprotocol/oasis-core/go/consensus/api/transaction"
	genesisFile "github.com/oasisprotocol/oasis-core/go/genesis/file"
	cmdCommon "github.com/oasisprotocol/oasis-core/go/oasis-node/cmd/common"
	cmdCommonFlags "github.com/oasisprotocol/oasis-core/go/oasis-node/cmd/common/flags"
	cmdGrpc "github.com/oasisprotocol/oasis-core/go/oasis-node/cmd/common/grpc"
	registryAPI "github.com/oasisprotocol/oasis-core/go/registry/api"
	roothashAPI "github.com/oasisprotocol/oasis-core/go/roothash/api"
	"github.com/oasisprotocol/oasis-core/go/roothash/api/block"
	"github.com/oasisprotocol/oasis-core/go/roothash/api/commitment"
	schedulerAPI "github.com/oasisprotocol/oasis-core/go/scheduler/api"
)

var queryCmd = &cobra.Command{
	Use:   "<runtime-id>",
	Short: "query runtime stats",
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

type stats struct {
	// Rounds.
	rounds uint64
	// Successful rounds.
	successfulRounds uint64
	// Failed rounds.
	failedRounds uint64
	// Rounds failed due to proposer timeouts.
	proposerTimeoutedRounds uint64
	// Epoch transition rounds.
	epochTransitionRounds uint64
	// Suspended rounds.
	suspendedRounds uint64

	// Discrepancies.
	discrepancyDetected        uint64
	discrepancyDetectedTimeout uint64

	// Per-entity stats.
	entities map[signature.PublicKey]*entityStats

	entitiesOutput [][]string
	entitiesHeader []string
}

type entityStats struct {
	// Rounds entity node was elected.
	roundsElected uint64
	// Rounds entity node was elected as primary executor worker.
	roundsPrimary uint64
	// Rounds entity node was elected as primary executor worker and workers were invoked.
	roundsPrimaryRequired uint64
	// Rounds entity node was elected as a backup executor worker.
	roundsBackup uint64
	// Rounds entity node was elected as a backup executor worker
	// and backup workers were invoked.
	roundsBackupRequired uint64
	// Rounds entity node was a proposer.
	roundsProposer uint64

	// How many times entity node proposed a timeout.
	proposedTimeout uint64

	// How many good blocks committed while being primary worker.
	committeedGoodBlocksPrimary uint64
	// How many bad blocs committed while being primary worker.
	committeedBadBlocksPrimary uint64
	// How many good blocks committed while being backup worker.
	committeedGoodBlocksBackup uint64
	// How many bad blocks committed while being backup worker.
	committeedBadBlocksBackup uint64

	// How many rounds missed committing a block while being a primary worker.
	missedPrimary uint64
	// How many rounds missed committing a block while being a backup worker (and discrepancy detection was invoked).
	missedBackup uint64
	// How many rounds proposer timeout was triggered while being the proposer.
	missedProposer uint64

	// Round at which the entity first joined the committee.
	joinedAt uint64
}

func (s *stats) prepareEntitiesOutput() {
	s.entitiesOutput = make([][]string, 0)

	s.entitiesHeader = []string{
		"Entity ID",
		"Joined at",
		"Elected",
		"Primary",
		"Backup",
		"Proposer",
		"Primary invoked",
		"Primary Good commit",
		"Prim Bad commmit",
		"Bckp invoked",
		"Bckp Good commit",
		"Bckp Bad commit",
		"Primary missed",
		"Bckp missed",
		"Proposer missed",
		"Proposed timeout",
	}

	for entity, stats := range s.entities {
		var line []string
		line = append(line,
			entity.String(),
			strconv.FormatUint(stats.joinedAt, 10),
			strconv.FormatUint(stats.roundsElected, 10),
			strconv.FormatUint(stats.roundsPrimary, 10),
			strconv.FormatUint(stats.roundsBackup, 10),
			strconv.FormatUint(stats.roundsProposer, 10),
			strconv.FormatUint(stats.roundsPrimaryRequired, 10),
			strconv.FormatUint(stats.committeedGoodBlocksPrimary, 10),
			strconv.FormatUint(stats.committeedBadBlocksPrimary, 10),
			strconv.FormatUint(stats.roundsBackupRequired, 10),
			strconv.FormatUint(stats.committeedGoodBlocksBackup, 10),
			strconv.FormatUint(stats.committeedBadBlocksBackup, 10),
			strconv.FormatUint(stats.missedPrimary, 10),
			strconv.FormatUint(stats.missedBackup, 10),
			strconv.FormatUint(stats.missedProposer, 10),
			strconv.FormatUint(stats.proposedTimeout, 10),
		)
		s.entitiesOutput = append(s.entitiesOutput, line)
	}
}

func (s *stats) printStats() {
	fmt.Printf("Runtime rounds: %d\n", s.rounds)
	fmt.Printf("Successful rounds: %d\n", s.successfulRounds)
	fmt.Printf("Epoch transition rounds: %d\n", s.epochTransitionRounds)
	fmt.Printf("Proposer timeouted rounds: %d\n", s.proposerTimeoutedRounds)
	fmt.Printf("Failed rounds: %d\n", s.failedRounds)
	fmt.Printf("Discrepancies: %d\n", s.discrepancyDetected)
	fmt.Printf("Discrepancies (timeout): %d\n", s.discrepancyDetectedTimeout)
	fmt.Printf("Suspended: %d\n", s.suspendedRounds)

	fmt.Println("Entity stats")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetHeader(s.entitiesHeader)
	table.AppendBulk(s.entitiesOutput)
	table.Render()
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

	if len(args) != 3 {
		cmdCommon.EarlyLogAndExit(fmt.Errorf("need exactly three arguments (runtimeID, start height, end height)"))
	}
	if err = runtimeID.UnmarshalHex(args[0]); err != nil {
		cmdCommon.EarlyLogAndExit(fmt.Errorf("malformed runtime ID: %s", args[0]))
	}
	startHeight, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}
	endHeight, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}
	conn := doConnect(cmd)

	// Clients.
	consensus := consensusAPI.NewConsensusClient(conn)
	roothash := roothashAPI.NewRootHashClient(conn)
	reg := registryAPI.NewRegistryClient(conn)

	// Take start height from genesis, if not provided.
	if startHeight == 0 {
		startHeight = uint64(doc.Height)
	}

	// Take end height from latest block, if not provided.
	if endHeight == 0 {
		blk, err := consensus.GetBlock(ctx, consensusAPI.HeightLatest)
		if err != nil {
			cmdCommon.EarlyLogAndExit(err)
		}
		endHeight = uint64(blk.Height)
	}

	stats := &stats{
		entities: make(map[signature.PublicKey]*entityStats),
	}

	var currentRound uint64
	var currentCommittee *schedulerAPI.Committee
	var currentScheduler *schedulerAPI.CommitteeNode
	var roundDiscrepancy bool
	nodeToEntity := make(map[signature.PublicKey]signature.PublicKey)
	for height := int64(startHeight); height < int64(endHeight); height++ {
		if height%1000 == 0 {
			fmt.Println("At height:", height)
		}
		// Update node to entity map.
		nodes, err := reg.GetNodes(ctx, height)
		if err != nil {
			cmdCommon.EarlyLogAndExit(err)
		}
		for _, node := range nodes {
			nodeToEntity[node.ID] = node.EntityID
		}

		// Query latest roothash block and events.
		blk, err := roothash.GetLatestBlock(ctx, &roothashAPI.RuntimeRequest{RuntimeID: runtimeID, Height: height})
		if errors.Is(err, roothashAPI.ErrInvalidRuntime) {
			// Runtime not yet registered at current height.
			continue
		}
		if err != nil {
			cmdCommon.EarlyLogAndExit(err)
		}
		evs, err := roothash.GetEvents(ctx, height)
		if err != nil {
			cmdCommon.EarlyLogAndExit(err)
		}

		var proposerTimeout bool
		if currentRound != blk.Header.Round && currentCommittee != nil {
			// If new round, check for proposer timeout.
			// Need to look at submitted transactions if round failure was caused by a proposer timeout.
			rsp, err := consensus.GetTransactionsWithResults(ctx, height)
			if err != nil {
				cmdCommon.EarlyLogAndExit(err)
			}
			for i := 0; i < len(rsp.Transactions); i++ {
				// Ignore failed txs.
				if !rsp.Results[i].IsSuccess() {
					continue
				}
				var sigTx transaction.SignedTransaction
				if err = cbor.Unmarshal(rsp.Transactions[i], &sigTx); err != nil {
					cmdCommon.EarlyLogAndExit(err)
				}
				var tx transaction.Transaction
				if err = sigTx.Open(&tx); err != nil {
					cmdCommon.EarlyLogAndExit(err)
				}
				// Ignore non proposer timeout txs.
				if tx.Method != roothashAPI.MethodExecutorProposerTimeout {
					continue
				}
				var xc roothashAPI.ExecutorProposerTimeoutRequest
				if err = cbor.Unmarshal(tx.Body, &xc); err != nil {
					cmdCommon.EarlyLogAndExit(err)
				}
				// Ignore txs of other runtimes.
				if xc.ID != runtimeID {
					continue
				}
				// Proposer timeout triggered the round failure, update stats.
				stats.entities[nodeToEntity[sigTx.Signature.PublicKey]].proposedTimeout++
				stats.entities[nodeToEntity[currentScheduler.PublicKey]].missedProposer++
				proposerTimeout = true
				break
			}
		}

		// Go over events before updating potential new round committee info.
		// Even if round transition happened at this height, all events emitted
		// at this height belong to the previous round.
		for _, ev := range evs {
			// Skip events for initial height where we don't have round info yet.
			if height == int64(startHeight) {
				break
			}
			// Skip events for other runtimes.
			if ev.RuntimeID != runtimeID {
				continue
			}
			switch {
			case ev.ExecutorCommitted != nil:
				// Nothing to do here. We use Finalized event Good/Bad Compute node
				// fields to process commitments.
			case ev.ExecutionDiscrepancyDetected != nil:
				if ev.ExecutionDiscrepancyDetected.Timeout {
					stats.discrepancyDetectedTimeout++
				} else {
					stats.discrepancyDetected++
				}
				roundDiscrepancy = true
			case ev.Finalized != nil:
				ev := ev.Finalized
				// Skip the empty finalized event that is triggered on initial round.
				if len(ev.GoodComputeNodes) == 0 && len(ev.BadComputeNodes) == 0 && currentCommittee == nil {
					continue
				}
				// Skip if epoch transition or suspended blocks.
				if blk.Header.HeaderType == block.EpochTransition || blk.Header.HeaderType == block.Suspended {
					continue
				}
				// Skip if proposer timeout.
				if proposerTimeout {
					continue
				}

				// Update stats.
			OUTER:
				for _, member := range currentCommittee.Members {
					entity := nodeToEntity[member.PublicKey]
					// Primary workers are always required.
					if member.Role == schedulerAPI.RoleWorker {
						stats.entities[nodeToEntity[member.PublicKey]].roundsPrimaryRequired++
					}
					// In case of discrepancies backup workers were invoked as well.
					if roundDiscrepancy && member.Role == schedulerAPI.RoleBackupWorker {
						stats.entities[nodeToEntity[member.PublicKey]].roundsBackupRequired++
					}

					// Go over good commitments.
					for _, g := range ev.GoodComputeNodes {
						if member.PublicKey == g && member.Role == schedulerAPI.RoleWorker {
							stats.entities[entity].committeedGoodBlocksPrimary++
							continue OUTER
						}
						if member.PublicKey == g && roundDiscrepancy && member.Role == schedulerAPI.RoleBackupWorker {
							stats.entities[entity].committeedGoodBlocksBackup++
							continue OUTER
						}
					}
					// Go over bad commitments.
					for _, g := range ev.BadComputeNodes {
						if member.PublicKey == g && member.Role == schedulerAPI.RoleWorker {
							stats.entities[entity].committeedBadBlocksPrimary++
							continue OUTER
						}
						if member.PublicKey == g && roundDiscrepancy && member.Role == schedulerAPI.RoleBackupWorker {
							stats.entities[entity].committeedBadBlocksBackup++
							continue OUTER
						}
					}
					// Neither good nor bad - missed commitment.
					if member.Role == schedulerAPI.RoleWorker {
						stats.entities[entity].missedPrimary++
					}
					if roundDiscrepancy && member.Role == schedulerAPI.RoleBackupWorker {
						stats.entities[entity].missedBackup++
					}
				}
			}
		}

		// New round.
		if currentRound != blk.Header.Round {
			currentRound = blk.Header.Round
			stats.rounds++

			switch blk.Header.HeaderType {
			case block.Normal:
				stats.successfulRounds++
			case block.EpochTransition:
				stats.epochTransitionRounds++
			case block.RoundFailed:
				if proposerTimeout {
					stats.proposerTimeoutedRounds++
				} else {
					stats.failedRounds++
				}
			case block.Suspended:
				stats.suspendedRounds++
				currentCommittee = nil
				currentScheduler = nil
				continue
			default:
				cmdCommon.EarlyLogAndExit(fmt.Errorf("unexpected header type: %v", blk.Header.HeaderType))
			}

			// Query runtime state and setup committee info for the round.
			state, err := roothash.GetRuntimeState(ctx, &roothashAPI.RuntimeRequest{RuntimeID: runtimeID, Height: height})
			if err != nil {
				cmdCommon.EarlyLogAndExit(err)
			}
			if state.ExecutorPool == nil {
				// No committee - shouldn't have happened unless runtime was suspended (handled above).
				cmdCommon.EarlyLogAndExit(fmt.Errorf("no committee at height: %d", height))
			}
			// Set committee info.
			currentCommittee = state.ExecutorPool.Committee
			currentScheduler, err = commitment.GetTransactionScheduler(currentCommittee, currentRound)
			if err != nil {
				cmdCommon.EarlyLogAndExit(err)
			}
			roundDiscrepancy = false

			// Update election stats.
			seen := make(map[signature.PublicKey]bool)
			for _, member := range currentCommittee.Members {
				entity := nodeToEntity[member.PublicKey]
				if _, ok := stats.entities[entity]; !ok {
					// New entity.
					stats.entities[entity] = &entityStats{
						joinedAt: currentRound,
					}
				}

				// Multiple records for same node in case the node has
				// multiple roles. Only count it as elected once.
				if !seen[member.PublicKey] {
					stats.entities[entity].roundsElected++
				}
				seen[member.PublicKey] = true

				if member.Role == schedulerAPI.RoleWorker {
					stats.entities[entity].roundsPrimary++
				}
				if member.Role == schedulerAPI.RoleBackupWorker {
					stats.entities[entity].roundsBackup++
				}
				if member.PublicKey == currentScheduler.PublicKey {
					stats.entities[entity].roundsProposer++
				}
			}
		}
	}

	// Prepare and printout stats.
	stats.prepareEntitiesOutput()
	stats.printStats()

	// Also save entity stats in a csv.
	fout, err := os.Create(fmt.Sprintf("runtime-%s-%d-%d-stats.csv", runtimeID, startHeight, endHeight))
	if err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}
	defer fout.Close()
	w := csv.NewWriter(fout)
	if err = w.Write(stats.entitiesHeader); err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}
	if err = w.WriteAll(stats.entitiesOutput); err != nil {
		cmdCommon.EarlyLogAndExit(err)
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
