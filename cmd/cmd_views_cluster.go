// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/cluster"
	"github.com/spf13/cobra"
)

func sortedClusterBrokers(topology cluster.Topology) []cluster.Broker {
	brokers := append([]cluster.Broker(nil), topology.Brokers...)
	sort.SliceStable(brokers, func(i, j int) bool {
		return brokers[i].NodeId < brokers[j].NodeId
	})
	return brokers
}

func sortedBrokerPartitions(broker cluster.Broker) []cluster.Partition {
	partitions := append([]cluster.Partition(nil), broker.Partitions...)
	sort.SliceStable(partitions, func(i, j int) bool {
		return partitions[i].PartitionId < partitions[j].PartitionId
	})
	return partitions
}

func formatClusterSummary(topology cluster.Topology) string {
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

func renderClusterTopologyTree(cmd *cobra.Command, topology cluster.Topology) error {
	renderOutputLine(cmd, "%s", formatClusterSummary(topology))

	brokers := sortedClusterBrokers(topology)
	for i, broker := range brokers {
		lastBroker := i == len(brokers)-1
		brokerBranch := "├─ "
		partitionPrefix := "│  "
		if lastBroker {
			brokerBranch = "└─ "
			partitionPrefix = "   "
		}

		renderOutputLine(cmd, "%s%s", brokerBranch, formatClusterBrokerLine(broker))

		partitions := sortedBrokerPartitions(broker)
		for j, partition := range partitions {
			partitionBranch := "├─ "
			if j == len(partitions)-1 {
				partitionBranch = "└─ "
			}
			renderOutputLine(cmd, "%s%s%s", partitionPrefix, partitionBranch, formatClusterPartitionLine(partition))
		}
	}

	return nil
}

func formatClusterBrokerLine(broker cluster.Broker) string {
	parts := []string{fmt.Sprintf("Broker %d", broker.NodeId)}

	details := make([]string, 0, 2)
	if broker.Host != "" {
		address := broker.Host
		if broker.Port != 0 {
			address = fmt.Sprintf("%s:%d", broker.Host, broker.Port)
		}
		details = append(details, address)
	} else if broker.Port != 0 {
		details = append(details, fmt.Sprintf("port=%d", broker.Port))
	}
	if broker.Version != "" {
		details = append(details, fmt.Sprintf("version=%s", broker.Version))
	}

	if len(details) > 0 {
		parts = append(parts, strings.Join(details, " "))
	}
	return strings.Join(parts, ": ")
}

func formatClusterPartitionLine(partition cluster.Partition) string {
	parts := []string{fmt.Sprintf("Partition %d", partition.PartitionId)}

	details := make([]string, 0, 2)
	if partition.Role != "" {
		details = append(details, fmt.Sprintf("role=%s", partition.Role))
	}
	if partition.Health != "" {
		details = append(details, fmt.Sprintf("health=%s", partition.Health))
	}

	if len(details) > 0 {
		parts = append(parts, strings.Join(details, " "))
	}
	return strings.Join(parts, ": ")
}
