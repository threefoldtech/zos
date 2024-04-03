package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/kernel"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/upgrade"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/identity"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/threefoldtech/zos/pkg/version"
)

const (
	redisSocket = "unix:///var/run/redis.sock"
)

const (
	module = "identityd"
)

// This daemon startup has the follow flow:
// 1. Do upgrade to latest version (this might means it needs to restart itself)
// 2. Register the node to BCDB
// 3. start zbus server to serve identity interface
// 4. Start watcher for new version
// 5. On update, re-register the node with new version to BCDB

func main() {
	app.Initialize()

	var (
		broker   string
		root     string
		interval int
		ver      bool
		debug    bool

		id      bool
		net     bool
		farm    bool
		address bool
	)

	flag.StringVar(&root, "root", "/var/cache/modules/identityd", "root working directory of the module")
	flag.StringVar(&broker, "broker", redisSocket, "connection string to broker")
	flag.IntVar(&interval, "interval", 600, "interval in seconds between update checks, default to 600")
	flag.BoolVar(&ver, "v", false, "show version and exit")
	flag.BoolVar(&debug, "d", false, "when set, no self update is done before upgrading")
	flag.BoolVar(&id, "id", false, "[deprecated] prints the node ID and exits")
	flag.BoolVar(&net, "net", false, "prints the node network and exits")
	flag.BoolVar(&farm, "farm", false, "prints the node farm id and exits")
	flag.BoolVar(&address, "address", false, "prints the node ss58 address and exits")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	if farm {
		env := environment.MustGet()
		fmt.Println(env.FarmID)
		os.Exit(0)
	} else if net {
		env := environment.MustGet()
		fmt.Println(env.RunningMode.String())
		os.Exit(0)
	} else if id || address {
		ctx := context.Background()
		client, err := zbus.NewRedisClient(broker)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to zbus")
		}
		stub := stubs.NewIdentityManagerStub(client)

		if id {
			fmt.Println(stub.NodeID(ctx))
		} else { // address
			add, err := stub.Address(ctx)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to get node address")
			}
			fmt.Println(add)
		}
		os.Exit(0)
	}

	if err := os.MkdirAll(root, 0750); err != nil {
		log.Fatal().Err(err).Str("root", root).Msg("failed to create root directory")
	}

	// 2. Register the node to BCDB
	// at this point we are running latest version
	idMgr, err := getIdentityMgr(root, debug)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create identity manager")
	}

	upgrader, err := upgrade.NewUpgrader(root, upgrade.NoZosUpgrade(debug))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize upgrader")
	}

	monitor := newVersionMonitor(10*time.Second, upgrader.Version())
	// 3. start zbus server to serve identity interface
	log.Info().Stringer("version", monitor.GetVersion()).Msg("current")

	server, err := zbus.NewRedisServer(module, broker, 1)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v\n", err)
	}

	server.Register(zbus.ObjectID{Name: "manager", Version: "0.0.1"}, idMgr)
	server.Register(zbus.ObjectID{Name: "monitor", Version: "0.0.1"}, monitor)

	ctx, cancel := utils.WithSignal(context.Background())
	// register the cancel function with defer if the process stops because of a update
	defer cancel()

	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("unexpected error")
		}
	}()

	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("received a termination signal")
	})

	err = manageSSHKeys()
	if err != nil {
		log.Error().Err(err).Msg("failed to configure ssh users")
	}

	err = upgrader.Run(ctx)
	if errors.Is(err, upgrade.ErrRestartNeeded) {
		return
	} else if err != nil {
		log.Error().Err(err).Msg("error during update")
		os.Exit(1)
	}
}

func getIdentityMgr(root string, debug bool) (pkg.IdentityManager, error) {
	manager, err := identity.NewManager(root, debug)
	if err != nil {
		return nil, err
	}

	env, err := environment.Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse node environment")
	}

	nodeID := manager.NodeID()
	log.Info().
		Str("identity", nodeID.Identity()).
		Msg("node identity loaded")

	log.Info().
		Bool("orphan", env.Orphan).
		Uint64("farmer_id", uint64(env.FarmID)).
		Msg("farmer identified")

	return manager, nil
}

func manageSSHKeys() error {
	extraUser, found := kernel.GetParams().GetOne("ssh-user")

	authorizedKeysPath := filepath.Join("/", "root", ".ssh", "authorized_keys")
	err := os.Remove(authorizedKeysPath)
	if err != nil {
		return err
	}

	env := environment.MustGet()
	config, err := environment.GetConfig()
	if err != nil {
		return err
	}

	authorizedUsers := config.Users.Authorized
	if env.RunningMode == environment.RunningMain {
		// use authorized users of testing if production has no authorized users
		if len(authorizedUsers) == 0 {
			config, err = environment.GetConfigForMode(environment.RunningTest)
			if err != nil {
				return err
			}

			authorizedUsers = config.Users.Authorized
		}

		// disable the extra user of any farm other than freefarm on production
		if env.FarmID != 1 {
			found = false
		}
	}

	if found {
		authorizedUsers = append(authorizedUsers, extraUser)
	}

	file, err := os.OpenFile(authorizedKeysPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	for _, user := range authorizedUsers {
		featchKeys := func() error {
			res, err := http.Get(fmt.Sprintf("https://github.com/%s.keys", user))
			if res.StatusCode == http.StatusNotFound {
				return backoff.Permanent(fmt.Errorf("failed to get user keys for user (%s): keys not found", user))
			}

			if err != nil {
				return err
			}

			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to get user keys for user (%s) with status code %d", user, res.StatusCode)
			}

			_, err = io.Copy(file, res.Body)
			return err
		}

		err = backoff.Retry(featchKeys, backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), 3))
		if err != nil {
			// skip user if failed to load the keys multiple times
			// this means the username is not correct and need to be skipped
			log.Error().Err(err).Send()
		}
	}

	return nil
}
