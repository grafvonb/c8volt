// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"fmt"
	"strings"
)

// ClusterTopology contains the readiness facts required before live mutations.
type ClusterTopology struct {
	GatewayVersion string           `json:"gateway_version"`
	Brokers        []BrokerTopology `json:"broker_details"`
}

// BrokerTopology retains each broker's independently reported version/health.
type BrokerTopology struct {
	Version    string              `json:"version"`
	Partitions []PartitionTopology `json:"partitions"`
}

// PartitionTopology retains the health value without assuming its letter case.
type PartitionTopology struct {
	Health string `json:"health"`
}

// ValidateTopology rejects mixed or unknown broker versions and unhealthy or
// unknown partition states, even when the connection request itself succeeds.
func ValidateTopology(cluster ClusterTopology) error {
	minor, err := ObservedMinor(cluster.GatewayVersion)
	if err != nil {
		return err
	}
	if len(cluster.Brokers) == 0 {
		return fmt.Errorf("cluster topology has no brokers")
	}
	partitions := 0
	for i, broker := range cluster.Brokers {
		brokerMinor, err := ObservedMinor(broker.Version)
		if err != nil {
			return fmt.Errorf("broker %d version: %w", i, err)
		}
		if brokerMinor != minor {
			return fmt.Errorf("broker %d version %q differs from gateway minor %q", i, broker.Version, minor)
		}
		for j, partition := range broker.Partitions {
			partitions++
			if !strings.EqualFold(partition.Health, "healthy") {
				return fmt.Errorf("broker %d partition %d has unhealthy or unknown health %q", i, j, partition.Health)
			}
		}
	}
	if partitions == 0 {
		return fmt.Errorf("cluster topology has no reported partitions")
	}
	return nil
}
