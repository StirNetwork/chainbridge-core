package utils

import (
	"fmt"
	"math/big"

	"github.com/StirNetwork/chainbridge-core/chains/evm/calls"
	"github.com/StirNetwork/chainbridge-core/chains/evm/cli/flags"
	"github.com/StirNetwork/chainbridge-core/chains/evm/evmclient"
	"github.com/StirNetwork/chainbridge-core/chains/evm/evmtransaction"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var simulateCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Simulate transaction invocation",
	Long:  "Replay a failed transaction by simulating invocation; not state-altering",
	RunE: func(cmd *cobra.Command, args []string) error {
		txFabric := evmtransaction.NewTransaction
		return SimulateCmd(cmd, args, txFabric)
	},
}

func BindSimulateCmdFlags(cli *cobra.Command) {
	cli.Flags().String("txHash", "", "transaction hash")
	cli.Flags().String("blockNumber", "", "block number")
	cli.Flags().String("fromAddress", "", "address of sender")
}

func init() {
	BindSimulateCmdFlags(simulateCmd)
}

func SimulateCmd(cmd *cobra.Command, args []string, txFabric calls.TxFabric) error {
	txHash := cmd.Flag("txHash").Value.String()
	blockNumber := cmd.Flag("blockNumber").Value.String()
	fromAddress := cmd.Flag("fromAddress").Value.String()

	// fetch global flag values
	url, _, gasPrice, senderKeyPair, err := flags.GlobalFlagValues(cmd)
	if err != nil {
		return fmt.Errorf("could not get global flags: %v", err)
	}

	// convert string block number to big.Int
	// ignore success bool
	blockNumberBigInt, _ := new(big.Int).SetString(blockNumber, 10)

	log.Debug().Msgf(`
Simulating transaction
Tx hash: %s
Block number: %v
From address: %s`,
		txHash, blockNumberBigInt, fromAddress)

	ethClient, err := evmclient.NewEVMClientFromParams(url, senderKeyPair.PrivateKey(), gasPrice)
	if err != nil {
		log.Error().Err(fmt.Errorf("eth client intialization error: %v", err))
		return err
	}

	data, err := ethClient.Simulate(blockNumberBigInt, common.HexToHash(txHash), common.HexToAddress(fromAddress))
	if err != nil {
		log.Error().Err(fmt.Errorf("[utils] simulate transact error: %v", err))
		return err
	}

	log.Debug().Msgf("data: %s", string(data))

	return nil
}
