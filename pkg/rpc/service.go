package rpc

import (
	"context"
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/diagnostics"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/stubs"
)

//go:generate openrpc-codegen -spec ../../openrpc.json -output ./types.go
//go:generate goimports -w ./types.go

type Service struct {
	ctx                    context.Context
	oracle                 *capacity.ResourceOracle
	versionMonitorStub     *stubs.VersionMonitorStub
	provisionStub          *stubs.ProvisionStub
	networkerStub          *stubs.NetworkerStub
	statisticsStub         *stubs.StatisticsStub
	storageStub            *stubs.StorageModuleStub
	performanceMonitorStub *stubs.PerformanceMonitorStub
	diagnosticsManager     *diagnostics.DiagnosticsManager
	farmerID               uint32
}

func NewService(ctx context.Context, manager substrate.Manager, client zbus.Client, msgBrokerCon string) (*Service, error) {
	sub, err := manager.Substrate()
	if err != nil {
		return nil, err
	}
	defer sub.Close()

	diagnosticsManager, err := diagnostics.NewDiagnosticsManager(msgBrokerCon, client)
	if err != nil {
		return nil, err
	}

	storageModuleStub := stubs.NewStorageModuleStub(client)

	server := Service{
		oracle:                 capacity.NewResourceOracle(storageModuleStub),
		versionMonitorStub:     stubs.NewVersionMonitorStub(client),
		provisionStub:          stubs.NewProvisionStub(client),
		networkerStub:          stubs.NewNetworkerStub(client),
		statisticsStub:         stubs.NewStatisticsStub(client),
		storageStub:            storageModuleStub,
		performanceMonitorStub: stubs.NewPerformanceMonitorStub(client),
		diagnosticsManager:     diagnosticsManager,
		ctx:                    ctx,
	}

	farm, err := sub.GetFarm(uint32(environment.MustGet().FarmID))
	if err != nil {
		return nil, fmt.Errorf("failed to get farm: %w", err)
	}
	farmer, err := sub.GetTwin(uint32(farm.TwinID))
	if err != nil {
		return nil, err
	}
	server.farmerID = uint32(farmer.ID)
	return &server, nil
}

var _ ZosRpcApi = (*Service)(nil)

func Run(ctx context.Context, port uint, manager substrate.Manager, client zbus.Client, msgBrokerCon string) error {
	service, err := NewService(ctx, manager, client, msgBrokerCon)
	if err != nil {
		return fmt.Errorf("failed to create api service: %w", err)
	}

	if err := rpc.RegisterName("zos", service); err != nil {
		return fmt.Errorf("failed to register api service: %w", err)
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %v: %w", port, err)
	}

	log.Info().Uint("port", port).Msg("rpc server started")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Error().Err(err).Send()
			continue
		}

		// SetTwinId(ctx, GetTwinFromIp(conn.RemoteAddr().String()))
		log.Info().Uint32("twinId", 0).Msg("got rpc request")
		go jsonrpc.ServeConn(conn)
	}
}
