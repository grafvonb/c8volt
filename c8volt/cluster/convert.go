package cluster

import (
	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromDomainTopology(t domain.Topology) Topology {
	return Topology{
		Brokers:               toolx.MapSlice(t.Brokers, fromDomainBroker),
		ClusterSize:           t.ClusterSize,
		GatewayVersion:        t.GatewayVersion,
		PartitionsCount:       t.PartitionsCount,
		ReplicationFactor:     t.ReplicationFactor,
		LastCompletedChangeId: t.LastCompletedChangeId,
	}
}

func fromDomainBroker(b domain.Broker) Broker {
	return Broker{
		Host:       b.Host,
		NodeId:     b.NodeId,
		Partitions: toolx.MapSlice(b.Partitions, fromDomainPartition),
		Port:       b.Port,
		Version:    b.Version,
	}
}

func fromDomainPartition(p domain.Partition) Partition {
	return Partition{
		Health:      PartitionHealth(p.Health),
		PartitionId: p.PartitionId,
		Role:        PartitionRole(p.Role),
	}
}
