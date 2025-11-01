package v88

import (
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromTopologyResponse(r camundav88.TopologyResponse) d.Topology {
	return d.Topology{
		Brokers:           toolx.MapSlice(r.Brokers, fromBrokerInfo),
		ClusterSize:       r.ClusterSize,
		GatewayVersion:    r.GatewayVersion,
		PartitionsCount:   r.PartitionsCount,
		ReplicationFactor: r.ReplicationFactor,
	}
}

func fromBrokerInfo(b camundav88.BrokerInfo) d.Broker {
	return d.Broker{
		Host:       b.Host,
		NodeId:     b.NodeId,
		Partitions: toolx.MapSlice(b.Partitions, fromPartition),
		Port:       b.Port,
		Version:    b.Version,
	}
}

func fromPartition(p camundav88.Partition) d.Partition {
	return d.Partition{
		Health:      d.PartitionHealth(p.Health),
		PartitionId: p.PartitionId,
		Role:        d.PartitionRole(p.Role),
	}
}
