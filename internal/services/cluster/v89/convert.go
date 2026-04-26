// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromTopologyResponse(r camundav89.TopologyResponse) d.Topology {
	return d.Topology{
		Brokers:           toolx.MapSlice(r.Brokers, fromBrokerInfo),
		ClusterSize:       r.ClusterSize,
		GatewayVersion:    r.GatewayVersion,
		PartitionsCount:   r.PartitionsCount,
		ReplicationFactor: r.ReplicationFactor,
	}
}

func fromLicenseResponse(r camundav89.LicenseResponse) d.License {
	return d.License{
		ExpiresAt:    r.ExpiresAt,
		IsCommercial: new(r.IsCommercial),
		LicenseType:  r.LicenseType,
		ValidLicense: r.ValidLicense,
	}
}

func fromBrokerInfo(b camundav89.BrokerInfo) d.Broker {
	return d.Broker{
		Host:       b.Host,
		NodeId:     b.NodeId,
		Partitions: toolx.MapSlice(b.Partitions, fromPartition),
		Port:       b.Port,
		Version:    b.Version,
	}
}

func fromPartition(p camundav89.Partition) d.Partition {
	return d.Partition{
		Health:      d.PartitionHealth(p.Health),
		PartitionId: p.PartitionId,
		Role:        d.PartitionRole(p.Role),
	}
}
