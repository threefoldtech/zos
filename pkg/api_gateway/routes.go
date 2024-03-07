package apigateway

import (
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func (g *apiGateway) setupRoutes(router peer.Router) {

	r := router.SubRoute("zos")
	system := r.SubRoute("system")
	system.WithHandler("version", g.systemVersionHandler)
	system.WithHandler("dmi", g.systemDMIHandler)
	system.WithHandler("hypervisor", g.systemHypervisorHandler)
	system.WithHandler("diagnostics", g.systemDiagnosticsHandler)

	perf := r.SubRoute("perf")
	perf.WithHandler("get", g.perfGetHandler)
	perf.WithHandler("get_all", g.perfGetAllHandler)

	gpu := r.SubRoute("gpu")
	gpu.WithHandler("list", g.gpuListHandler)

	storage := r.SubRoute("storage")
	storage.WithHandler("pools", g.storagePoolsHandler)

	network := r.SubRoute("network")
	network.WithHandler("list_wg_ports", g.networkListWGPortsHandler)
	network.WithHandler("public_config_get", g.networkPublicConfigGetHandler)
	network.WithHandler("interfaces", g.networkInterfacesHandler)
	network.WithHandler("has_ipv6", g.networkHasIPv6Handler)
	network.WithHandler("list_public_ips", g.networkListPublicIPsHandler)
	network.WithHandler("list_private_ips", g.networkListPrivateIPsHandler)

	statistics := r.SubRoute("statistics")
	statistics.WithHandler("get", g.statisticsGetHandler)

	deployment := r.SubRoute("deployment")
	deployment.WithHandler("deploy", g.deploymentDeployHandler)
	deployment.WithHandler("update", g.deploymentUpdateHandler)
	deployment.WithHandler("delete", g.deploymentDeleteHandler)
	deployment.WithHandler("get", g.deploymentGetHandler)
	deployment.WithHandler("list", g.deploymentListHandler)
	deployment.WithHandler("changes", g.deploymentChangesHandler)

	admin := r.SubRoute("admin")
	// not working!
	// admin.Use(g.authorized)
	admin.WithHandler("interfaces", g.adminInterfacesHandler)
	admin.WithHandler("set_public_nic", g.adminSetPublicNICHandler)
	admin.WithHandler("get_public_nic", g.adminGetPublicNICHandler)
}
