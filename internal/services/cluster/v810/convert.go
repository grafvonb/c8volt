// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// fromTopologyResponse maps a V810 topology payload into the version-neutral topology model.
func fromTopologyResponse(r camundav810.TopologyResponse) d.Topology {
	return d.Topology{
		Brokers:               toolx.MapSlice(r.Brokers, fromBrokerInfo),
		ClusterSize:           r.ClusterSize,
		GatewayVersion:        r.GatewayVersion,
		LastCompletedChangeId: r.LastCompletedChangeId,
		PartitionsCount:       r.PartitionsCount,
		ReplicationFactor:     r.ReplicationFactor,
	}
}

// fromLicenseResponse maps a V810 license payload into the version-neutral license model.
func fromLicenseResponse(r camundav810.LicenseResponse) d.License {
	return d.License{
		ExpiresAt:    r.ExpiresAt,
		IsCommercial: new(r.IsCommercial),
		LicenseType:  r.LicenseType,
		ValidLicense: r.ValidLicense,
	}
}

// fromBrokerInfo maps a V810 broker payload into the shared domain broker shape.
func fromBrokerInfo(b camundav810.BrokerInfo) d.Broker {
	return d.Broker{
		Host:       b.Host,
		NodeId:     b.NodeId,
		Partitions: toolx.MapSlice(b.Partitions, fromPartition),
		Port:       b.Port,
		Version:    b.Version,
	}
}

// fromPartition maps a V810 partition payload into the shared domain partition shape.
func fromPartition(p camundav810.Partition) d.Partition {
	return d.Partition{
		Health:      d.PartitionHealth(p.Health),
		PartitionId: p.PartitionId,
		Role:        d.PartitionRole(p.Role),
	}
}
