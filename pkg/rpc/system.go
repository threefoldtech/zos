package rpc

import (
	"fmt"
	"os/exec"
	"strings"
)

func (s *Service) SystemVersion(arg any, reply *Version) error {
	output, err := exec.CommandContext(s.ctx, "zinit", "-V").CombinedOutput()
	var zInitVer string
	if err != nil {
		zInitVer = err.Error()
	} else {
		zInitVer = strings.TrimSpace(strings.TrimPrefix(string(output), "zinit"))
	}

	reply.Zos = s.versionMonitorStub.GetVersion(s.ctx).String()
	reply.Zinit = zInitVer

	return nil
}

func (s *Service) SystemHypervisor(arg any, reply *string) error {
	hv, err := s.oracle.GetHypervisor()
	if err != nil {
		return fmt.Errorf("failed to get hypervisor: %w", err)
	}

	*reply = hv
	return nil
}

func (s *Service) SystemDiagnostics(arg any, reply *Diagnostics) error {
	dia, err := s.diagnosticsManager.GetSystemDiagnostics(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to get diagnostics: %w", err)
	}
	return convert(dia, reply)
}

func (s *Service) SystemDmi(arg any, reply *DMI) error {
	dmi, err := s.oracle.DMI()
	if err != nil {
		return fmt.Errorf("failed to get dmi: %w", err)
	}
	convertDmi(dmi, reply)
	return nil
}
