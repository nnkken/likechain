package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/likecoin/likechain/app"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/cosmos/cosmos-sdk/x/staking"

	"github.com/likecoin/likechain/ip"
)

// liked custom flags
const flagInvCheckPeriod = "inv-check-period"
const flagGetIP = "get-ip"

var invCheckPeriod uint
var shouldGetIP bool

func persistentPreRunEFn(ctx *server.Context) func(cmd *cobra.Command, args []string) error {
	originalFn := server.PersistentPreRunEFn(ctx)
	return func(cmd *cobra.Command, args []string) error {
		err := originalFn(cmd, args)
		if err != nil {
			return err
		}
		if shouldGetIP {
			laddr, err := url.Parse(ctx.Config.P2P.ListenAddress)
			if err != nil {
				return errors.New("cannot parse p2p.laddr")
			}
			port := laddr.Port()
			if port == "" {
				return errors.New("cannot get port from p2p.laddr")
			}
			fmt.Println("getting external IP address")
			ip, err := ip.RunProviders(ip.IPGetters, ip.DefaultTimeout)
			if err != nil {
				fmt.Println("Get IP failed, ignoring")
				return nil
			}
			fmt.Printf("Got external IP: %s\n", ip)
			ctx.Config.P2P.ExternalAddress = fmt.Sprintf("tcp://%s:%s", ip, laddr.Port())
			fmt.Printf("p2p.external_address = %s\n", ctx.Config.P2P.ExternalAddress)
		}
		return nil
	}
}

func main() {
	cdc := app.MakeCodec()

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(sdk.Bech32PrefixAccAddr, sdk.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(sdk.Bech32PrefixValAddr, sdk.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(sdk.Bech32PrefixConsAddr, sdk.Bech32PrefixConsPub)
	config.Seal()

	ctx := server.NewDefaultContext()
	cobra.EnableCommandSorting = false
	rootCmd := &cobra.Command{
		Use:               "liked",
		Short:             "LikeChain Daemon (server)",
		PersistentPreRunE: persistentPreRunEFn(ctx),
	}

	rootCmd.AddCommand(genutilcli.InitCmd(ctx, cdc, app.ModuleBasics, app.DefaultNodeHome))
	rootCmd.AddCommand(genutilcli.CollectGenTxsCmd(ctx, cdc, auth.GenesisAccountIterator{}, app.DefaultNodeHome))
	rootCmd.AddCommand(genutilcli.MigrateGenesisCmd(ctx, cdc))
	rootCmd.AddCommand(genutilcli.GenTxCmd(ctx, cdc, app.ModuleBasics, staking.AppModuleBasic{},
		auth.GenesisAccountIterator{}, app.DefaultNodeHome, app.DefaultCLIHome))
	rootCmd.AddCommand(genutilcli.ValidateGenesisCmd(ctx, cdc, app.ModuleBasics))

	server.AddCommands(ctx, cdc, rootCmd, newApp, exportAppStateAndTMValidators)
	rootCmd.PersistentFlags().BoolVar(&shouldGetIP, flagGetIP, false, "Get external IP for Tendermint")

	// prepare and add flags
	executor := cli.PrepareBaseCmd(rootCmd, "GA", app.DefaultNodeHome)
	rootCmd.PersistentFlags().UintVar(&invCheckPeriod, flagInvCheckPeriod,
		0, "Assert registered invariants every N blocks")
	err := executor.Execute()
	if err != nil {
		panic(err)
	}
}

func newApp(logger log.Logger, db dbm.DB, traceStore io.Writer) abci.Application {
	return app.NewLikeApp(
		logger, db, traceStore, true, invCheckPeriod,
		baseapp.SetPruning(store.NewPruningOptionsFromString(viper.GetString("pruning"))),
		baseapp.SetMinGasPrices(viper.GetString(server.FlagMinGasPrices)),
		baseapp.SetHaltHeight(uint64(viper.GetInt(server.FlagHaltHeight))),
	)
}

func exportAppStateAndTMValidators(
	logger log.Logger, db dbm.DB, traceStore io.Writer, height int64, forZeroHeight bool, jailWhiteList []string,
) (json.RawMessage, []tmtypes.GenesisValidator, error) {

	if height != -1 {
		gApp := app.NewLikeApp(logger, db, traceStore, false, uint(1))
		err := gApp.LoadHeight(height)
		if err != nil {
			return nil, nil, err
		}
		return gApp.ExportAppStateAndValidators(forZeroHeight, jailWhiteList)
	}
	gApp := app.NewLikeApp(logger, db, traceStore, true, uint(1))
	return gApp.ExportAppStateAndValidators(forZeroHeight, jailWhiteList)
}
