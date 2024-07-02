package zos_rpc

import (
	"context"
	"os/exec"
	"strings"
)

// var _ GoOpenRPCService = (*ApiService)(nil)

func (s *ApiService) SystemVersion() (*SystemVersionResult, error) {
	ctx := context.Background()

	output, err := exec.CommandContext(ctx, "zinit", "-V").CombinedOutput()
	var zInitVer string
	if err != nil {
		zInitVer = err.Error()
	} else {
		zInitVer = strings.TrimSpace(strings.TrimPrefix(string(output), "zinit"))
	}

	version := Version{
		Zos:   s.versionMonitorStub.GetVersion(ctx).String(),
		Zinit: zInitVer,
	}

	return &SystemVersionResult{
		Version: version,
	}, nil
}
