package ingress

import "context"

type MockTunnelClient struct{}

func (MockTunnelClient) PutExposures(_ context.Context, _ []Exposure) error {
	return nil
}
