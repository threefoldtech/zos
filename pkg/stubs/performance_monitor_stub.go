// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type PerformanceMonitorStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewPerformanceMonitorStub(client zbus.Client) *PerformanceMonitorStub {
	return &PerformanceMonitorStub{
		client: client,
		module: "node",
		object: zbus.ObjectID{
			Name:    "performance-monitor",
			Version: "0.0.1",
		},
	}
}

func (s *PerformanceMonitorStub) Get(ctx context.Context, arg0 string) (ret0 pkg.TaskResult, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Get", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *PerformanceMonitorStub) GetAll(ctx context.Context) (ret0 []pkg.TaskResult, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetAll", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *PerformanceMonitorStub) GetAllTaskResult(ctx context.Context) (ret0 pkg.AllTaskResult, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetAllTaskResult", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *PerformanceMonitorStub) GetCpuBenchTaskResult(ctx context.Context) (ret0 pkg.CpuBenchTaskResult, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetCpuBenchTaskResult", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *PerformanceMonitorStub) GetHealthTaskResult(ctx context.Context) (ret0 pkg.HealthTaskResult, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetHealthTaskResult", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *PerformanceMonitorStub) GetIperfTaskResult(ctx context.Context) (ret0 pkg.IperfTaskResult, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetIperfTaskResult", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *PerformanceMonitorStub) GetPublicIpTaskResult(ctx context.Context) (ret0 pkg.PublicIpTaskResult, ret1 error) {
	args := []interface{}{}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetPublicIpTaskResult", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}
