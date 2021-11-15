package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/StirNetwork/chainbridge-core/chains/evm"
	evmCLI "github.com/StirNetwork/chainbridge-core/chains/evm/cli"
	"github.com/StirNetwork/chainbridge-core/chains/evm/evmclient"
	"github.com/StirNetwork/chainbridge-core/chains/evm/evmtransaction"
	"github.com/StirNetwork/chainbridge-core/chains/evm/listener"
	"github.com/StirNetwork/chainbridge-core/chains/evm/voter"
	"github.com/StirNetwork/chainbridge-core/config"
	"github.com/StirNetwork/chainbridge-core/lvldb"
	"github.com/StirNetwork/chainbridge-core/relayer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	rootCMD = &cobra.Command{
		Use: "",
	}
	runCMD = &cobra.Command{
		Use:   "run",
		Short: "Run app",
		Long:  "Run app",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := Run(); err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	config.BindFlags(runCMD)
}

func Execute() {
	rootCMD.AddCommand(runCMD, evmCLI.EvmRootCLI)
	if err := rootCMD.Execute(); err != nil {
		log.Fatal().Err(err).Msg("failed to execute root cmd")
	}
}

func Run() error {
	errChn := make(chan error)
	stopChn := make(chan struct{})

	db, err := lvldb.NewLvlDB(viper.GetString(config.BlockstoreFlagName))
	if err != nil {
		panic(err)
	}

	// ===== BSC setup =====
	// bsc is evm compatible. so we can utilize evm module
	bscClient := evmclient.NewEVMClient()
	err = bscClient.Configurate(viper.GetString(config.ConfigFlagName), "bsc.json")
	if err != nil {
		panic(err)
	}

	bscConfig := bscClient.GetConfig()
	bscEventHandler := listener.NewETHEventHandler(common.HexToAddress(bscConfig.SharedEVMConfig.Bridge), bscClient)
	bscEventHandler.RegisterEventHandler(bscConfig.SharedEVMConfig.Erc20Handler, listener.Erc20EventHandler)
	bscListener := listener.NewEVMListener(bscClient, bscEventHandler, common.HexToAddress(bscConfig.SharedEVMConfig.Bridge))

	bscMessageHandler := voter.NewEVMMessageHandler(bscClient, common.HexToAddress(bscConfig.SharedEVMConfig.Bridge))
	bscMessageHandler.RegisterMessageHandler(common.HexToAddress(bscConfig.SharedEVMConfig.Erc20Handler), voter.ERC20MessageHandler)
	bscVoter := voter.NewVoter(bscMessageHandler, bscClient, evmtransaction.NewTransaction)

	bscChain := evm.NewEVMChain(bscListener, bscVoter, db, *bscConfig.SharedEVMConfig.GeneralChainConfig.Id, &bscConfig.SharedEVMConfig)

	// ===== Shiden setup =====
	// sdn is evm compatible. so we can utilize evm module
	sdnClient := evmclient.NewEVMClient()
	err = sdnClient.Configurate(viper.GetString(config.ConfigFlagName), "sdn.json")
	if err != nil {
		panic(err)
	}

	sdnConfig := sdnClient.GetConfig()
	sdnEventHandler := listener.NewETHEventHandler(common.HexToAddress(sdnConfig.SharedEVMConfig.Bridge), sdnClient)
	sdnEventHandler.RegisterEventHandler(sdnConfig.SharedEVMConfig.Erc20Handler, listener.Erc20EventHandler)
	sdnListener := listener.NewEVMListener(sdnClient, sdnEventHandler, common.HexToAddress(sdnConfig.SharedEVMConfig.Bridge))

	sdnMessageHandler := voter.NewEVMMessageHandler(sdnClient, common.HexToAddress(sdnConfig.SharedEVMConfig.Bridge))
	sdnMessageHandler.RegisterMessageHandler(common.HexToAddress(sdnConfig.SharedEVMConfig.Erc20Handler), voter.ERC20MessageHandler)
	sdnVoter := voter.NewVoter(sdnMessageHandler, sdnClient, evmtransaction.NewTransaction)

	sdnChain := evm.NewEVMChain(sdnListener, sdnVoter, db, *sdnConfig.SharedEVMConfig.GeneralChainConfig.Id, &sdnConfig.SharedEVMConfig)

	r := relayer.NewRelayer([]relayer.RelayedChain{bscChain, sdnChain})

	go r.Start(stopChn, errChn)

	sysErr := make(chan os.Signal, 1)
	signal.Notify(sysErr,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGQUIT)

	select {
	case err := <-errChn:
		log.Error().Err(err).Msg("failed to listen and serve")
		close(stopChn)
		return err
	case sig := <-sysErr:
		log.Info().Msgf("terminating got ` [%v] signal", sig)
		return nil
	}
}
