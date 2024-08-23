package rpc

import (
	"context"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func (s *Service) DeploymentChanges(arg uint64, reply *Workloads) error {
	wls, err := s.provisionStub.Changes(context.Background(), GetTwinID(s.ctx), arg)
	if err != nil {
		return err
	}

	for _, wl := range wls {
		var workload Workload
		if err := convert(wl, &workload); err != nil {
			return err
		}
		reply.Workloads = append(reply.Workloads, workload)
	}

	return nil
}

func (s *Service) DeploymentList(arg any, reply *Deployments) error {
	deps, err := s.provisionStub.List(s.ctx, GetTwinID(s.ctx))
	if err != nil {
		return err
	}

	for _, dep := range deps {
		var deployment Deployment
		if err := convert(dep, &deployment); err != nil {
			return err
		}
		reply.Deployments = append(reply.Deployments, deployment)
	}

	return nil
}

func (s *Service) DeploymentGet(arg uint64, reply *Deployment) error {
	dep, err := s.provisionStub.Get(s.ctx, GetTwinID(s.ctx), arg)
	if err != nil {
		return err
	}

	return convert(dep, reply)
}

func (s *Service) DeploymentUpdate(arg Deployment, reply *any) error {
	var deployment gridtypes.Deployment
	if err := convert(arg, &deployment); err != nil {
		return err
	}
	return s.provisionStub.CreateOrUpdate(s.ctx, GetTwinID(s.ctx), deployment, true)
}

func (s *Service) DeploymentDeploy(arg Deployment, reply *any) error {
	var deployment gridtypes.Deployment
	if err := convert(arg, &deployment); err != nil {
		return err
	}
	return s.provisionStub.CreateOrUpdate(s.ctx, GetTwinID(s.ctx), deployment, false)
}
