package v87

import (
	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromTopologyResponse(r camundav87.TopologyResponse) d.Topology {
	return d.Topology{
		Brokers:           toolx.DerefSlicePtr(r.Brokers, fromBrokerInfo),
		ClusterSize:       toolx.Deref(r.ClusterSize, int32(0)),
		GatewayVersion:    toolx.Deref(r.GatewayVersion, ""),
		PartitionsCount:   toolx.Deref(r.PartitionsCount, int32(0)),
		ReplicationFactor: toolx.Deref(r.ReplicationFactor, int32(0)),
	}
}

func fromBrokerInfo(b camundav87.BrokerInfo) d.Broker {
	return d.Broker{
		Host:       toolx.Deref(b.Host, ""),
		NodeId:     toolx.Deref(b.NodeId, int32(0)),
		Partitions: toolx.DerefSlicePtr(b.Partitions, fromPartition),
		Port:       toolx.Deref(b.Port, int32(0)),
		Version:    toolx.Deref(b.Version, ""),
	}
}

func fromPartition(p camundav87.Partition) d.Partition {
	return d.Partition{
		Health:      d.PartitionHealth(toolx.Deref(p.Health, "")),
		PartitionId: toolx.Deref(p.PartitionId, int32(0)),
		Role:        d.PartitionRole(toolx.Deref(p.Role, "")),
	}
}
