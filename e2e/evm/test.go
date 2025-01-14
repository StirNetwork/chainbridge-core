package evm

import (
	"context"
	"math/big"
	"time"

	"github.com/StirNetwork/chainbridge-core/chains/evm/calls/consts"
	"github.com/StirNetwork/chainbridge-core/crypto/secp256k1"

	"github.com/StirNetwork/chainbridge-core/chains/evm/calls"
	"github.com/StirNetwork/chainbridge-core/chains/evm/evmclient"
	"github.com/StirNetwork/chainbridge-core/keystore"
	"github.com/StirNetwork/chainbridge-core/relayer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

type TestClient interface {
	calls.ChainClient
	LatestBlock() (*big.Int, error)
	FetchEventLogs(ctx context.Context, contractAddress common.Address, event string, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error)
}

func PreSetupTestSuite(fabric1, fabric2 calls.TxFabric, endpoint1, endpoint2 string, adminKey *secp256k1.Keypair) *IntegrationTestSuite {
	return &IntegrationTestSuite{
		fabric1:   fabric1,
		fabric2:   fabric2,
		endpoint1: endpoint1,
		endpoint2: endpoint2,
		adminKey:  adminKey,
	}
}

type IntegrationTestSuite struct {
	suite.Suite
	client            TestClient
	client2           TestClient
	bridgeAddr        common.Address
	erc20HandlerAddr  common.Address
	erc20ContractAddr common.Address
	fabric1           calls.TxFabric
	fabric2           calls.TxFabric
	endpoint1         string
	endpoint2         string
	adminKey          *secp256k1.Keypair
}

func (s *IntegrationTestSuite) SetupSuite()    {}
func (s *IntegrationTestSuite) TearDownSuite() {}
func (s *IntegrationTestSuite) SetupTest() {
	ethClient, err := evmclient.NewEVMClientFromParams(s.endpoint1, s.adminKey.PrivateKey(), big.NewInt(consts.DefaultGasPrice))
	if err != nil {
		panic(err)
	}
	ethClient2, err := evmclient.NewEVMClientFromParams(s.endpoint2, s.adminKey.PrivateKey(), big.NewInt(consts.DefaultGasPrice))
	if err != nil {
		panic(err)
	}
	b, err := ethClient.LatestBlock()
	if err != nil {
		panic(err)
	}
	log.Debug().Msgf("Latest block %s", b.String())
	bridgeAddr, erc20Addr, erc20HandlerAddr, err := PrepareEVME2EEnv(ethClient, s.fabric1, 1, big.NewInt(1), s.adminKey.CommonAddress())
	if err != nil {
		panic(err)
	}
	s.client = ethClient
	s.client2 = ethClient2
	s.bridgeAddr = bridgeAddr
	s.erc20ContractAddr = erc20Addr
	s.erc20HandlerAddr = erc20HandlerAddr
	//Contract addresses should be the same
	_, _, _, err = PrepareEVME2EEnv(ethClient2, s.fabric2, 2, big.NewInt(1), s.adminKey.CommonAddress())
	if err != nil {
		panic(err)
	}
}

func (s *IntegrationTestSuite) TestDeposit() {
	dstAddr := keystore.TestKeyRing.EthereumKeys[keystore.BobKey].CommonAddress()
	senderBalBefore, err := calls.GetERC20Balance(s.client, s.erc20ContractAddr, EveKp.CommonAddress())
	s.Nil(err)
	destBalanceBefore, err := calls.GetERC20Balance(s.client2, s.erc20ContractAddr, dstAddr)
	s.Nil(err)

	b, err := s.client2.LatestBlock()
	if err != nil {
		panic(err)
	}
	amountToDeposit := big.NewInt(1000000)
	resourceID := calls.SliceTo32Bytes(append(common.LeftPadBytes(s.erc20ContractAddr.Bytes(), 31), uint8(0)))
	err = calls.Deposit(s.client, s.fabric1, s.bridgeAddr, dstAddr, amountToDeposit, resourceID, 2)
	s.Nil(err)

	//Wait 120 seconds for relayer vote
	time.Sleep(120 * time.Second)
	senderBalAfter, err := calls.GetERC20Balance(s.client, s.erc20ContractAddr, s.adminKey.CommonAddress())
	s.Nil(err)
	s.Equal(-1, senderBalAfter.Cmp(senderBalBefore))

	ba, err := s.client2.LatestBlock()
	if err != nil {
		panic(err)
	}
	//wait for vote log event
	proposalEvent := "ProposalEvent(uint8,uint64,uint8,bytes32,bytes32)"
	evts, _ := s.client2.FetchEventLogs(context.Background(), s.bridgeAddr, proposalEvent, b, ba)
	var passedEventFound bool
	for _, evt := range evts {
		status := evt.Topics[3].Big().Uint64()
		if uint8(relayer.ProposalStatusPassed) == uint8(status) {
			passedEventFound = true
		}
	}
	s.True(passedEventFound)
	s.Equal(senderBalBefore.Cmp(big.NewInt(0).Add(senderBalAfter, amountToDeposit)), 0)

	//Wait 30 seconds for relayer to execute
	time.Sleep(30 * time.Second)

	ba, err = s.client2.LatestBlock()
	s.Nil(err)
	queryExecute, err := s.client2.FetchEventLogs(context.Background(), s.bridgeAddr, proposalEvent, b, ba)
	s.Nil(err)
	var executedEventFound bool
	for _, evt := range queryExecute {
		status := evt.Topics[3].Big().Uint64()
		if uint8(relayer.ProposalStatusExecuted) == uint8(status) {
			executedEventFound = true
		}
	}
	s.True(executedEventFound)

	destBalanceAfter, err := calls.GetERC20Balance(s.client2, s.erc20ContractAddr, dstAddr)
	s.Nil(err)
	//Balance has increased
	s.Equal(1, destBalanceAfter.Cmp(destBalanceBefore))
}
