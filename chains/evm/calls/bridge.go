package calls

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/consts"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/rs/zerolog/log"
)

func PrepareSetBurnableInput(handler, tokenAddress common.Address) ([]byte, error) {
	a, err := abi.JSON(strings.NewReader(consts.BridgeABI))
	if err != nil {
		return []byte{}, err
	}
	input, err := a.Pack("adminSetBurnable", handler, tokenAddress)
	if err != nil {
		return []byte{}, err
	}
	return input, nil
}

func PrepareAdminSetResourceInput(handler common.Address, rId [32]byte, addr common.Address) ([]byte, error) {
	log.Debug().Msgf("ResourceID %x", rId)
	a, err := abi.JSON(strings.NewReader(consts.BridgeABI))
	if err != nil {
		return []byte{}, err
	}
	input, err := a.Pack("adminSetResource", handler, rId, addr)
	if err != nil {
		return []byte{}, err
	}
	return input, nil
}

func PrepareAdminSetGenericResourceInput(
	handler common.Address,
	rId [32]byte,
	addr common.Address,
	depositFunctionSig [4]byte,
	executeFunctionSig [4]byte,
) ([]byte, error) {
	a, err := abi.JSON(strings.NewReader(consts.BridgeABI))
	if err != nil {
		return []byte{}, err
	}
	input, err := a.Pack("adminSetGenericResource", handler, rId, addr, depositFunctionSig, executeFunctionSig)
	if err != nil {
		return []byte{}, err
	}
	return input, nil
}

func PrepareAddRelayerInput(relayer common.Address) ([]byte, error) {
	a, err := abi.JSON(strings.NewReader(consts.BridgeABI))
	if err != nil {
		return []byte{}, err
	}
	input, err := a.Pack("adminAddRelayer", relayer)
	if err != nil {
		return []byte{}, err
	}
	return input, nil
}

func PrepareIsRelayerInput(address common.Address) ([]byte, error) {
	a, err := abi.JSON(strings.NewReader(consts.BridgeABI))
	if err != nil {
		return nil, err
	}

	data, err := a.Pack("isRelayer", address)
	if err != nil {
		log.Error().Err(fmt.Errorf("unpack output error: %v", err))
		return nil, err
	}
	return data, nil
}

func ParseIsRelayerOutput(output []byte) (bool, error) {
	a, err := abi.JSON(strings.NewReader(consts.BridgeABI))
	if err != nil {
		return false, err
	}

	res, err := a.Unpack("isRelayer", output)
	if err != nil {
		log.Error().Err(fmt.Errorf("unpack output error: %v", err))
		return false, err
	}

	b := abi.ConvertType(res[0], new(bool)).(*bool)
	return *b, nil
}

func ConstructErc20DepositData(destRecipient []byte, amount *big.Int) []byte {
	var data []byte
	data = append(data, math.PaddedBigBytes(amount, 32)...)
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len(destRecipient))), 32)...)
	data = append(data, destRecipient...)
	return data
}

func ConstructGenericDepositData(metadata []byte) []byte {
	var data []byte
	data = append(data, math.PaddedBigBytes(big.NewInt(int64(len(metadata))), 32)...)
	data = append(data, metadata...)
	return data
}

func PrepareDepositInput(destDomainID uint8, resourceID [32]byte, data []byte) ([]byte, error) {
	a, err := abi.JSON(strings.NewReader(consts.BridgeABI))
	if err != nil {
		return []byte{}, err
	}
	input, err := a.Pack("deposit", destDomainID, resourceID, data)
	if err != nil {
		return []byte{}, err
	}
	return input, nil
}

func Deposit(client ChainClient, fabric TxFabric, bridgeAddress common.Address, resourceID [32]byte, destDomainID uint8, data []byte) error {
	input, err := PrepareDepositInput(destDomainID, resourceID, data)
	if err != nil {
		return err
	}

	gasLimit := uint64(2000000)
	h, err := Transact(client, fabric, &bridgeAddress, input, gasLimit)
	if err != nil {
		return fmt.Errorf("deposit failed %w", err)
	}
	log.Debug().Str("hash", h.String()).Msgf("Deposit sent")
	return nil
}
