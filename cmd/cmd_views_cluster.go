// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"sort"

	"github.com/grafvonb/c8volt/internal/domain"
)

func sortedClusterBrokers(topology domain.Topology) []domain.Broker {
	brokers := append([]domain.Broker(nil), topology.Brokers...)
	sort.SliceStable(brokers, func(i, j int) bool {
		return brokers[i].NodeId < brokers[j].NodeId
	})
	return brokers
}

func sortedBrokerPartitions(broker domain.Broker) []domain.Partition {
	partitions := append([]domain.Partition(nil), broker.Partitions...)
	sort.SliceStable(partitions, func(i, j int) bool {
		return partitions[i].PartitionId < partitions[j].PartitionId
	})
	return partitions
}

func formatClusterSummary(topology domain.Topology) string {
	lastCompletedChangeID := topology.LastCompletedChangeId
	if lastCompletedChangeID == "" {
		lastCompletedChangeID = "-"
	}
	return fmt.Sprintf(
		"Cluster: GatewayVersion=%s Brokers=%d Partitions=%d ReplicationFactor=%d LastCompletedChangeId=%s",
		topology.GatewayVersion,
		topology.ClusterSize,
		topology.PartitionsCount,
		topology.ReplicationFactor,
		lastCompletedChangeID,
	)
}
