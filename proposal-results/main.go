package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"

	metadata "github.com/oasisprotocol/metadata-registry-tools"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/oasisprotocol/oasis-core/go/common/crypto/signature"
	"github.com/oasisprotocol/oasis-core/go/common/quantity"
	consensus "github.com/oasisprotocol/oasis-core/go/consensus/api"
	governance "github.com/oasisprotocol/oasis-core/go/governance/api"
	cmdCommon "github.com/oasisprotocol/oasis-core/go/oasis-node/cmd/common"
	cmdGrpc "github.com/oasisprotocol/oasis-core/go/oasis-node/cmd/common/grpc"
	registry "github.com/oasisprotocol/oasis-core/go/registry/api"
	staking "github.com/oasisprotocol/oasis-core/go/staking/api"
)

var queryCmd = &cobra.Command{
	Use:   "<proposal-id>",
	Short: "proposal id",
	Run:   doQuery,
}

func exitErr(err error) {
	if err != nil {
		fmt.Println("err:", err)
		os.Exit(1)
	}
}

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

type EntityMeta struct {
	EntityID signature.PublicKey
	Name     string
}

func (e *EntityMeta) Address() staking.Address {
	return staking.NewAddress(e.EntityID)
}

func getMetadataRegistryMeta(ctx context.Context) (map[staking.Address]*EntityMeta, error) {
	gp, err := metadata.NewGitProvider(metadata.NewGitConfig())
	if err != nil {
		return nil, err
	}

	entities, err := gp.GetEntities(ctx)
	if err != nil {
		return nil, err
	}

	meta := make(map[staking.Address]*EntityMeta, len(entities))
	for id, ent := range entities {
		em := &EntityMeta{
			EntityID: id,
			Name:     ent.Name,
		}
		meta[em.Address()] = em
	}

	return meta, nil
}

func name(registry *EntityMeta, oscan *EntityMeta) string {
	// Prefer registry meta.
	if registry != nil {
		return registry.Name
	}
	if oscan != nil && oscan.Name != "" {
		return oscan.Name + " (from oasisscan)"
	}
	return "<none>"
}

func sortByStake(keyvals map[staking.Address]quantity.Quantity) KeyVals {
	pl := make(KeyVals, len(keyvals))
	i := 0
	for k, v := range keyvals {
		pl[i] = &KeyVal{k, v}
		i++
	}
	sort.Sort(sort.Reverse(pl))
	return pl
}

type KeyVal struct {
	Key   staking.Address
	Value quantity.Quantity
}

type KeyVals []*KeyVal

func (p KeyVals) Len() int           { return len(p) }
func (p KeyVals) Less(i, j int) bool { return p[i].Value.Cmp(&p[j].Value) < 0 }
func (p KeyVals) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func doQuery(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	if len(args) != 1 {
		cmdCommon.EarlyLogAndExit(fmt.Errorf("need exactly one argument (proposal id)"))
	}
	pid, err := strconv.Atoi(args[0])
	exitErr(err)
	proposalID := uint64(pid)
	conn := doConnect(cmd)

	cons := consensus.NewConsensusClient(conn)
	gov := cons.Governance()
	sched := cons.Scheduler()
	reg := cons.Registry()
	stake := cons.Staking()
	beac := cons.Beacon()

	// Get latest height.
	blk, err := cons.GetBlock(ctx, consensus.HeightLatest)
	exitErr(err)
	latestHeight := blk.Height

	p, err := gov.Proposal(ctx, &governance.ProposalQuery{Height: latestHeight, ProposalID: proposalID})
	exitErr(err)

	switch p.State {
	case governance.StateActive:
		epoch, err := beac.GetEpoch(ctx, latestHeight)
		exitErr(err)

		// Simulate proposal closing.
		// https://github.com/oasisprotocol/oasis-core/blob/5a88c9bb64ccbfb98b07e177adfba3d338f172e0/go/consensus/tendermint/apps/governance/governance.go#L391-L431

		totalVotingStake := quantity.NewQuantity()
		validatorEntitiesEscrow := make(map[staking.Address]*quantity.Quantity)
		currentValidators, err := sched.GetValidators(ctx, latestHeight)
		exitErr(err)

		oasisScanMeta, err := getOasisscanMeta(ctx)
		if err != nil {
			oasisScanMeta = make(map[staking.Address]*EntityMeta)
			fmt.Errorf("error loading oasiscan meta: %w", err)
		}
		registryMeta, err := getMetadataRegistryMeta(ctx)
		if err != nil {
			registryMeta = make(map[staking.Address]*EntityMeta)
			fmt.Errorf("error loading metadata-registry meta: %w", err)
		}

		voters := make(map[staking.Address]quantity.Quantity)
		nonVoters := make(map[staking.Address]quantity.Quantity)

		for _, valID := range currentValidators {
			node, err := reg.GetNode(ctx, &registry.IDQuery{Height: latestHeight, ID: valID.ID})
			exitErr(err)

			entityAddr := staking.NewAddress(node.EntityID)
			// If there are multiple nodes in the validator set belonging to the same entity,
			// only count the entity escrow once.
			if validatorEntitiesEscrow[entityAddr] != nil {
				continue
			}

			account, err := stake.Account(ctx, &staking.OwnerQuery{Height: latestHeight, Owner: entityAddr})
			exitErr(err)

			validatorEntitiesEscrow[entityAddr] = &account.Escrow.Active.Balance
			err = totalVotingStake.Add(&account.Escrow.Active.Balance)
			exitErr(err)
			nonVoters[entityAddr] = account.Escrow.Active.Balance
		}

		// Tally the votes.
		votes, err := gov.Votes(ctx, &governance.ProposalQuery{Height: latestHeight, ProposalID: proposalID})
		exitErr(err)

		p.Results = make(map[governance.Vote]quantity.Quantity)
		for _, vote := range votes {
			escrow, ok := validatorEntitiesEscrow[vote.Voter]
			if !ok {
				// Voter not in current validator set - invalid vote.
				p.InvalidVotes++
				continue
			}

			currentVotes := p.Results[vote.Vote]
			newVotes := escrow.Clone()
			if err := newVotes.Add(&currentVotes); err != nil {
				exitErr(fmt.Errorf("failed to add votes: %w", err))
			}
			p.Results[vote.Vote] = *newVotes

			delete(nonVoters, vote.Voter)
			voters[vote.Voter] = *escrow.Clone()
		}

		// Query governance parameters.
		params, err := gov.ConsensusParameters(ctx, latestHeight)
		exitErr(err)

		// Try closing the proposal.
		err = p.CloseProposal(*totalVotingStake.Clone(), params.Quorum, params.Threshold)
		exitErr(err)

		fmt.Println("Proposal active, vote outcome if ended now:", p.State)
		fmt.Printf("Voting ends in %d epochs\n", p.ClosesAt-epoch)

		// Calculate voting percentages.
		votedStake, err := p.VotedSum()
		exitErr(err)

		voteStakePercentage := new(big.Float).SetInt(votedStake.Clone().ToBigInt())
		voteStakePercentage = voteStakePercentage.Mul(voteStakePercentage, new(big.Float).SetInt64(100))
		voteStakePercentage = voteStakePercentage.Quo(voteStakePercentage, new(big.Float).SetInt(totalVotingStake.ToBigInt()))
		fmt.Printf("\nVoted stake: %s (%.2f%%), total voting stake: %s, quorum: %d%%\n", votedStake, voteStakePercentage, totalVotingStake, params.Quorum)

		votedYes := p.Results[governance.VoteYes]
		votedYesPercentage := new(big.Float).SetInt(votedYes.Clone().ToBigInt())
		votedYesPercentage = votedYesPercentage.Mul(votedYesPercentage, new(big.Float).SetInt64(100))
		if votedStake.Cmp(quantity.NewFromUint64(0)) > 0 {
			votedYesPercentage = votedYesPercentage.Quo(votedYesPercentage, new(big.Float).SetInt(votedStake.ToBigInt()))
		}
		fmt.Printf("Voted yes stake: %s (%.2f%%), voted stake: %s, threshold: %d%%\n", votedYes, votedYesPercentage, votedStake, params.Threshold)

		fmt.Println("\nValidators voted:")
		votersList := sortByStake(voters)
		for _, val := range votersList {
			name := name(registryMeta[val.Key], oasisScanMeta[val.Key])
			stakePercentage := new(big.Float).SetInt(val.Value.Clone().ToBigInt())
			stakePercentage = stakePercentage.Mul(stakePercentage, new(big.Float).SetInt64(100))
			stakePercentage = stakePercentage.Quo(stakePercentage, new(big.Float).SetInt(totalVotingStake.ToBigInt()))
			fmt.Printf("%s,%s,%s (%.2f%%)\n", val.Key, name, val.Value, stakePercentage)
		}
		fmt.Println("\nValidators not voted:")
		nonVotersList := sortByStake(nonVoters)
		for _, val := range nonVotersList {
			name := name(registryMeta[val.Key], oasisScanMeta[val.Key])
			stakePercentage := new(big.Float).SetInt(val.Value.Clone().ToBigInt())
			stakePercentage = stakePercentage.Mul(stakePercentage, new(big.Float).SetInt64(100))
			stakePercentage = stakePercentage.Quo(stakePercentage, new(big.Float).SetInt(totalVotingStake.ToBigInt()))
			fmt.Printf("%s,%s,%s (%.2f%%)\n", val.Key, name, val.Value, stakePercentage)
		}

	case governance.StatePassed:
		fmt.Println("proposal passed, results: ", p.Results)
	case governance.StateFailed:
		fmt.Println("proposal failed, results: ", p.Results)
	case governance.StateRejected:
		fmt.Println("proposal rejected, results: ", p.Results)
	default:
		fmt.Println("unexpected proposal state: ", p.State)
		os.Exit(1)
	}
}

func main() {
	if err := queryCmd.Execute(); err != nil {
		cmdCommon.EarlyLogAndExit(err)
	}
}

func init() {
	queryCmd.PersistentFlags().AddFlagSet(cmdGrpc.ClientFlags)
}
