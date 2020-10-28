package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"

	"github.com/likecoin/likechain/app"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/snapshots"
	"github.com/cosmos/cosmos-sdk/store"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	tmcli "github.com/tendermint/tendermint/libs/cli"

	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"

	"github.com/likecoin/likechain/ip"

	gaia "github.com/cosmos/gaia/app"
	"github.com/cosmos/gaia/app/params"
	gaiacmd "github.com/cosmos/gaia/cmd/gaiad/cmd"
)

// liked custom flags
const flagInvCheckPeriod = "inv-check-period"
const flagGetIP = "get-ip"

var shouldGetIP bool

// TODO: iterate to find the `start` command, then add `--add-ip` flag
// rootCmd.PersistentFlags().BoolVar(&shouldGetIP, flagGetIP, false, "Get external IP for Tendermint")
func persistentPreRunEFn(ctx *server.Context) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
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

func NewRootCmd() (*cobra.Command, params.EncodingConfig) {
	encodingConfig := gaia.MakeEncodingConfig()

	initClientCtx := client.Context{}.
		WithJSONMarshaler(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastBlock).
		WithHomeDir(app.DefaultNodeHome)

	rootCmd := &cobra.Command{
		Use:   "liked",
		Short: "LikeCoin chain App",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			return server.InterceptConfigsPreRunHandler(cmd)
		},
	}

	authclient.Codec = encodingConfig.Marshaler

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(sdk.Bech32PrefixAccAddr, sdk.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(sdk.Bech32PrefixValAddr, sdk.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(sdk.Bech32PrefixConsAddr, sdk.Bech32PrefixConsPub)
	config.Seal()

	rootCmd.AddCommand(
		genutilcli.InitCmd(app.ModuleBasics, app.DefaultNodeHome),
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{}, app.DefaultNodeHome),
		genutilcli.MigrateGenesisCmd(),
		genutilcli.GenTxCmd(
			app.ModuleBasics, encodingConfig.TxConfig, banktypes.GenesisBalancesIterator{},
			app.DefaultNodeHome,
		),
		genutilcli.ValidateGenesisCmd(app.ModuleBasics, encodingConfig.TxConfig),
		gaiacmd.AddGenesisAccountCmd(app.DefaultNodeHome),
		tmcli.NewCompletionCmd(rootCmd, true),
		debug.Cmd(),
	)

	server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, exportAppState)
	return rootCmd, encodingConfig
}

func Execute() error {
	rootCmd, _ := NewRootCmd()

	ctx := context.Background()
	ctx = context.WithValue(ctx, client.ClientContextKey, &client.Context{})
	ctx = context.WithValue(ctx, server.ServerContextKey, server.NewDefaultContext())

	// vvvvvvvvvvvvvvvv TESTING ONLY, REMOVE BEFORE COMMIT vvvvvvvvvvvvvvvv
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "start" {
			originalPreRunE := cmd.PreRunE
			cmd.Flags().BoolVar(
				&shouldGetIP, flagGetIP, false, "Get external IP for Tendermint p2p listen address",
			)
			cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
				if shouldGetIP {
					ctx := server.GetServerContextFromCmd(cmd)
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
				return originalPreRunE(cmd, args)
			}
			break
		}
	}

	rootCmd.AddCommand(&cobra.Command{
		Use: "dev-test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	})
	// ^^^^^^^^^^^^^^^^ TESTING ONLY, REMOVE BEFORE COMMIT ^^^^^^^^^^^^^^^^

	executor := cli.PrepareBaseCmd(rootCmd, "LIKE", app.DefaultNodeHome)
	return executor.ExecuteContext(ctx)
}

func newApp(
	logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions,
) servertypes.Application {
	var cache sdk.MultiStorePersistentCache

	if cast.ToBool(appOpts.Get(server.FlagInterBlockCache)) {
		cache = store.NewCommitKVStoreCacheManager()
	}

	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	pruningOpts, err := server.GetPruningOptionsFromFlags(appOpts)
	if err != nil {
		panic(err)
	}

	snapshotDir := filepath.Join(cast.ToString(appOpts.Get(flags.FlagHome)), "data", "snapshots")
	snapshotDB, err := sdk.NewLevelDB("metadata", snapshotDir)
	if err != nil {
		panic(err)
	}
	snapshotStore, err := snapshots.NewStore(snapshotDB, snapshotDir)
	if err != nil {
		panic(err)
	}

	return app.NewLikeApp(
		logger, db, traceStore, true, skipUpgradeHeights,
		cast.ToString(appOpts.Get(flags.FlagHome)),
		cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod)),
		gaia.MakeEncodingConfig(),
		baseapp.SetPruning(pruningOpts),
		baseapp.SetMinGasPrices(cast.ToString(appOpts.Get(server.FlagMinGasPrices))),
		baseapp.SetMinRetainBlocks(cast.ToUint64(appOpts.Get(server.FlagMinRetainBlocks))),
		baseapp.SetHaltHeight(cast.ToUint64(appOpts.Get(server.FlagHaltHeight))),
		baseapp.SetHaltTime(cast.ToUint64(appOpts.Get(server.FlagHaltTime))),
		baseapp.SetInterBlockCache(cache),
		baseapp.SetTrace(cast.ToBool(appOpts.Get(server.FlagTrace))),
		baseapp.SetIndexEvents(cast.ToStringSlice(appOpts.Get(server.FlagIndexEvents))),
		baseapp.SetSnapshotStore(snapshotStore),
		baseapp.SetSnapshotInterval(
			cast.ToUint64(appOpts.Get(server.FlagStateSyncSnapshotInterval)),
		),
		baseapp.SetSnapshotKeepRecent(
			cast.ToUint32(appOpts.Get(server.FlagStateSyncSnapshotKeepRecent)),
		),
	)
}

func exportAppState(
	logger log.Logger, db dbm.DB, traceStore io.Writer, height int64, forZeroHeight bool,
	jailAllowedAddrs []string,
) (servertypes.ExportedApp, error) {
	encodingConfig := gaia.MakeEncodingConfig()
	encodingConfig.Marshaler = codec.NewProtoCodec(encodingConfig.InterfaceRegistry)
	var likeApp *app.LikeApp
	if height != -1 {
		likeApp = app.NewLikeApp(
			logger, db, traceStore, false, map[int64]bool{}, "", uint(1), encodingConfig,
		)
		err := likeApp.LoadHeight(height)
		if err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		likeApp = app.NewLikeApp(
			logger, db, traceStore, true, map[int64]bool{}, "", uint(1), encodingConfig,
		)
	}
	return likeApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs)
}
