package example

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ChainSafe/chainbridgev2/lvldatabase"

	"github.com/ChainSafe/chainbridgev2/chains/evm"
	"github.com/ChainSafe/chainbridgev2/chains/evm/client"
	"github.com/ChainSafe/chainbridgev2/chains/evm/listener"
	"github.com/ChainSafe/chainbridgev2/example/keystore"
	"github.com/ChainSafe/chainbridgev2/relayer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var AliceKp = keystore.TestKeyRing.EthereumKeys[keystore.AliceKey]
var BobKp = keystore.TestKeyRing.EthereumKeys[keystore.BobKey]
var EveKp = keystore.TestKeyRing.EthereumKeys[keystore.EveKey]

var (
	DefaultRelayerAddresses = []common.Address{
		common.HexToAddress(keystore.TestKeyRing.EthereumKeys[keystore.AliceKey].Address()),
		common.HexToAddress(keystore.TestKeyRing.EthereumKeys[keystore.BobKey].Address()),
		common.HexToAddress(keystore.TestKeyRing.EthereumKeys[keystore.CharlieKey].Address()),
		common.HexToAddress(keystore.TestKeyRing.EthereumKeys[keystore.DaveKey].Address()),
		common.HexToAddress(keystore.TestKeyRing.EthereumKeys[keystore.EveKey].Address()),
	}
)

const DefaultGasLimit = 6721975
const DefaultGasPrice = 20000000000

const TestEndpoint = "ws://localhost:8545"
const TestEndpoint2 = "ws://localhost:8546"

//Bridge:             0x62877dDCd49aD22f5eDfc6ac108e9a4b5D2bD88B
//Erc20 Handler:      0x3167776db165D8eA0f51790CA2bbf44Db5105ADF
func Run(ctx *cli.Context) error {
	errChn := make(chan error)
	stopChn := make(chan struct{})

	c, err := client.NewClient(TestEndpoint, false, AliceKp, stopChn, errChn, common.HexToAddress("0x62877dDCd49aD22f5eDfc6ac108e9a4b5D2bD88B"))
	if err != nil {
		panic(err)
	}

	eventListener := listener.NewEVMListener(c, "0x62877dDCd49aD22f5eDfc6ac108e9a4b5D2bD88B", 1)

	eventListener.RegisterHandler("0x3167776db165D8eA0f51790CA2bbf44Db5105ADF", listener.HandleErc20DepositedEvent)

	lvldb, err := lvldatabase.NewLvlDB("./lvldbdata")
	if err != nil {
		panic(err)
	}

	chain := evm.NewEVMChain(eventListener, nil, lvldb, c)
	if err != nil {
		panic(err)
	}

	r := relayer.NewRelayer([]relayer.RelayedChain{chain})

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
		log.Info().Msgf("terminating got [%v] signal", sig)
		return nil
	}
}