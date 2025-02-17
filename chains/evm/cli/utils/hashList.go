package utils

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/StirNetwork/chainbridge-core/chains/evm/cli/flags"
	"github.com/StirNetwork/chainbridge-core/chains/evm/evmclient"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var hashListCmd = &cobra.Command{
	Use:   "hashList",
	Short: "List tx hashes",
	Long:  "List tx hashes",
	RunE: func(cmd *cobra.Command, args []string) error {
		return HashListCmd(cmd, args)
	},
}

func BindHashListCmdFlags(cli *cobra.Command) {
	cli.Flags().String("blockNumber", "", "block number")
}

func init() {
	BindHashListCmdFlags(hashListCmd)
}

func HashListCmd(cmd *cobra.Command, args []string) error {
	blockNumber := cmd.Flag("blockNumber").Value.String()

	// fetch global flag values
	url, _, gasPrice, senderKeyPair, err := flags.GlobalFlagValues(cmd)
	if err != nil {
		return fmt.Errorf("could not get global flags: %v", err)
	}

	ethClient, err := evmclient.NewEVMClientFromParams(url, senderKeyPair.PrivateKey(), gasPrice)
	if err != nil {
		log.Error().Err(fmt.Errorf("eth client intialization error: %v", err))
		return err
	}

	blockNum, err := strconv.Atoi(blockNumber)
	if err != nil {
		log.Error().Err(fmt.Errorf("block string->int conversion error: %v", err))
		return err
	}

	blockNumStr := strconv.Itoa(blockNum)
	blockNumberBigInt, _ := new(big.Int).SetString(blockNumStr, 10)

	// check block by hash
	// see if transaction block data is there
	for i := 0; i < 50; i++ {
		log.Debug().Msgf("blockNum: %v", blockNumberBigInt)

		// convert string block number to big.Int
		// ignore success bool

		blockNumberBigInt.Add(blockNumberBigInt, big.NewInt(1))

		block, err := ethClient.BlockByNumber(context.Background(), blockNumberBigInt)
		if err != nil {
			log.Error().Err(fmt.Errorf("block by hash error: %v", err))

			// will return early and not print debug log if block not found
			// Error: not found

			// return err
		}

		log.Debug().Msgf("block: %v", block)
	}

	return nil
}
