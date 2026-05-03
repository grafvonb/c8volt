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

// sortedClusterBrokers gives renderers stable output without mutating service payloads.
func sortedClusterBrokers(topology cluster.Topology) []cluster.Broker {
	brokers := append([]cluster.Broker(nil), topology.Brokers...)
	sort.SliceStable(brokers, func(i, j int) bool {
		return brokers[i].NodeId < brokers[j].NodeId
	})
	return brokers
}

// sortedBrokerPartitions keeps partition rows deterministic while preserving the source broker.
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

// renderClusterTopologyTree mirrors the existing tree-style CLI output while making
// broker and partition order independent of the API response order.
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

// renderClusterVersion intentionally reads from topology because the version command
// shares the cluster endpoint and only expands broker details on request.
func renderClusterVersion(cmd *cobra.Command, topology cluster.Topology, withBrokers bool) error {
	if !withBrokers {
		renderOutputLine(cmd, "%s", topology.GatewayVersion)
		return nil
	}

	renderOutputLine(cmd, "GatewayVersion: %s", topology.GatewayVersion)
	renderOutputLine(cmd, "")
	renderOutputLine(cmd, "Brokers:")
	for _, broker := range sortedClusterBrokers(topology) {
		renderOutputLine(cmd, "%s", formatClusterBrokerVersionLine(broker))
	}
	return nil
}

type clusterVersionView struct {
	GatewayVersion string                     `json:"GatewayVersion"`
	Brokers        []clusterBrokerVersionView `json:"Brokers,omitempty"`
}

type clusterBrokerVersionView struct {
	NodeId  int32  `json:"NodeId"`
	Version string `json:"Version"`
	Host    string `json:"Host,omitempty"`
}

func newClusterVersionView(topology cluster.Topology, withBrokers bool) clusterVersionView {
	view := clusterVersionView{
		GatewayVersion: topology.GatewayVersion,
	}
	if !withBrokers {
		return view
	}

	brokers := sortedClusterBrokers(topology)
	view.Brokers = make([]clusterBrokerVersionView, 0, len(brokers))
	for _, broker := range brokers {
		view.Brokers = append(view.Brokers, clusterBrokerVersionView{
			NodeId:  broker.NodeId,
			Version: broker.Version,
			Host:    broker.Host,
		})
	}
	return view
}

// renderClusterLicenseFlat omits absent optional fields so missing API values stay distinct
// from explicit false or zero values in human output.
func renderClusterLicenseFlat(cmd *cobra.Command, license cluster.License) error {
	renderOutputLine(cmd, "ValidLicense: %t", license.ValidLicense)
	renderOutputLine(cmd, "LicenseType: %s", license.LicenseType)
	if license.ExpiresAt != nil {
		renderOutputLine(cmd, "ExpiresAt: %s", license.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
	}
	if license.IsCommercial != nil {
		renderOutputLine(cmd, "IsCommercial: %t", *license.IsCommercial)
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

func formatClusterBrokerVersionLine(broker cluster.Broker) string {
	version := broker.Version
	if version == "" {
		version = "-"
	}
	if broker.Host == "" {
		return fmt.Sprintf("Broker %d: %s", broker.NodeId, version)
	}
	return fmt.Sprintf("Broker %d: %s (%s)", broker.NodeId, version, broker.Host)
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
