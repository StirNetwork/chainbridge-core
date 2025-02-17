package calls

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/StirNetwork/chainbridge-core/chains/evm/calls/consts"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
)

func DeployErc20(c ChainClient, txFabric TxFabric, name, symbol string) (common.Address, error) {
	parsed, err := abi.JSON(strings.NewReader(consts.ERC20PresetMinterPauserABI))
	if err != nil {
		return common.Address{}, err
	}
	address, err := deployContract(c, parsed, common.FromHex(consts.ERC20PresetMinterPauserBin), txFabric, name, symbol)
	if err != nil {
		return common.Address{}, err
	}
	return address, nil
}

func DeployBridge(c ChainClient, txFabric TxFabric, chainID uint8, relayerAddrs []common.Address, initialRelayerThreshold *big.Int) (common.Address, error) {
	parsed, err := abi.JSON(strings.NewReader(consts.BridgeABI))
	if err != nil {
		return common.Address{}, err
	}
	address, err := deployContract(c, parsed, common.FromHex(consts.BridgeBin), txFabric, chainID, relayerAddrs, initialRelayerThreshold, big.NewInt(0), big.NewInt(100))
	if err != nil {
		return common.Address{}, err
	}
	return address, nil
}

func DeployErc20Handler(c ChainClient, txFabric TxFabric, bridgeAddress common.Address) (common.Address, error) {
	log.Debug().Msgf("Deployng ERC20 Handler with params: %s", bridgeAddress.String())
	parsed, err := abi.JSON(strings.NewReader(consts.ERC20HandlerABI))
	if err != nil {
		return common.Address{}, err
	}
	address, err := deployContract(c, parsed, common.FromHex(consts.ERC20HandlerBin), txFabric, bridgeAddress, [][32]byte{}, []common.Address{}, []common.Address{})
	if err != nil {
		return common.Address{}, err
	}
	return address, nil
}

func deployContract(client ChainClient, abi abi.ABI, bytecode []byte, txFabric TxFabric, params ...interface{}) (common.Address, error) {
	gp, err := client.GasPrice()
	if err != nil {
		return common.Address{}, err
	}
	client.LockNonce()
	n, err := client.UnsafeNonce()
	if err != nil {
		return common.Address{}, err
	}
	input, err := abi.Pack("", params...)
	if err != nil {
		return common.Address{}, err
	}
	tx := txFabric(n.Uint64(), nil, big.NewInt(0), consts.DefaultDeployGasLimit, gp, append(bytecode, input...))
	hash, err := client.SignAndSendTransaction(context.TODO(), tx)
	if err != nil {
		return common.Address{}, err
	}
	time.Sleep(2 * time.Second)
	_, err = client.WaitAndReturnTxReceipt(tx.Hash())
	if err != nil {
		return common.Address{}, err
	}
	log.Debug().Str("hash", hash.String()).Uint64("nonce", n.Uint64()).Msgf("Contract deployed")
	address := crypto.CreateAddress(client.From(), n.Uint64())
	err = client.UnsafeIncreaseNonce()
	if err != nil {
		return common.Address{}, err
	}
	client.UnlockNonce()
	// checks bytecode at address
	// nil is latest block
	if code, err := client.CodeAt(context.Background(), address, nil); err != nil {
		return common.Address{}, err
	} else if len(code) == 0 {
		return common.Address{}, fmt.Errorf("no code at provided address %s", address.String())
	}
	return address, nil
}
