package common

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	servicecommon "github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

type PayloadResponse[T any] struct {
	Received     bool
	HTTPResponse *http.Response
	Body         []byte
	Payload      *T
}

func GetClusterTopology[T any](
	ctx context.Context,
	log *slog.Logger,
	baseURL string,
	opts []services.CallOption,
	fetch func(context.Context) (PayloadResponse[T], error),
	convert func(T) d.Topology,
) (d.Topology, error) {
	callCfg := services.ApplyCallOptions(opts)
	servicecommon.VerboseLog(ctx, callCfg, log, "requesting cluster topology", "baseURL", baseURL)

	resp, err := fetch(ctx)
	if err != nil {
		return d.Topology{}, fmt.Errorf("fetch cluster topology: %w", err)
	}
	if !resp.Received {
		return d.Topology{}, fmt.Errorf("%w: topology response is nil", d.ErrMalformedResponse)
	}
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.Topology{}, fmt.Errorf("fetch cluster topology: %w", err)
	}
	if resp.Payload == nil {
		return d.Topology{}, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(resp.Body))
	}

	topology := convert(*resp.Payload)
	servicecommon.VerboseLog(ctx, callCfg, log, "cluster topology retrieved", "brokers", len(topology.Brokers), "clusterSize", topology.ClusterSize)
	return topology, nil
}
