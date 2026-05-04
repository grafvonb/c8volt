// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/cluster"
	"github.com/grafvonb/c8volt/config"
)

type configTestConnectionView struct {
	OK         bool                            `json:"ok"`
	ConfigFile string                          `json:"config_file,omitempty"`
	Profile    string                          `json:"profile,omitempty"`
	BaseURL    string                          `json:"base_url"`
	Cluster    configTestConnectionClusterView `json:"cluster"`
	Warnings   []string                        `json:"warnings"`
}

type configTestConnectionClusterView struct {
	GatewayVersion        string                           `json:"gateway_version"`
	Brokers               int32                            `json:"brokers"`
	Partitions            int32                            `json:"partitions"`
	ReplicationFactor     int32                            `json:"replication_factor"`
	LastCompletedChangeID string                           `json:"last_completed_change_id"`
	BrokerDetails         []configTestConnectionBrokerView `json:"broker_details"`
}

type configTestConnectionBrokerView struct {
	ID         int32                               `json:"id"`
	Host       string                              `json:"host,omitempty"`
	Port       int32                               `json:"port,omitempty"`
	Address    string                              `json:"address,omitempty"`
	Version    string                              `json:"version,omitempty"`
	Partitions []configTestConnectionPartitionView `json:"partitions"`
}

type configTestConnectionPartitionView struct {
	ID     int32  `json:"id"`
	Role   string `json:"role,omitempty"`
	Health string `json:"health,omitempty"`
}

func newConfigTestConnectionView(cfg *config.Config, source configSourceDescription, topology cluster.Topology, warnings []string) configTestConnectionView {
	view := configTestConnectionView{
		OK:         true,
		ConfigFile: source.loadedPath,
		BaseURL:    cfg.APIs.Camunda.BaseURL,
		Cluster: configTestConnectionClusterView{
			GatewayVersion:        topology.GatewayVersion,
			Brokers:               topology.ClusterSize,
			Partitions:            topology.PartitionsCount,
			ReplicationFactor:     topology.ReplicationFactor,
			LastCompletedChangeID: topology.LastCompletedChangeId,
		},
		Warnings: warnings,
	}
	if cfg.ActiveProfile != "" {
		view.Profile = cfg.ActiveProfile
	}

	brokers := sortedClusterBrokers(topology)
	view.Cluster.BrokerDetails = make([]configTestConnectionBrokerView, 0, len(brokers))
	for _, broker := range brokers {
		brokerView := configTestConnectionBrokerView{
			ID:      broker.NodeId,
			Host:    broker.Host,
			Port:    broker.Port,
			Address: brokerAddress(broker),
			Version: broker.Version,
		}

		partitions := sortedBrokerPartitions(broker)
		brokerView.Partitions = make([]configTestConnectionPartitionView, 0, len(partitions))
		for _, partition := range partitions {
			brokerView.Partitions = append(brokerView.Partitions, configTestConnectionPartitionView{
				ID:     partition.PartitionId,
				Role:   string(partition.Role),
				Health: string(partition.Health),
			})
		}
		view.Cluster.BrokerDetails = append(view.Cluster.BrokerDetails, brokerView)
	}

	return view
}

func brokerAddress(broker cluster.Broker) string {
	if broker.Host == "" {
		return ""
	}
	if broker.Port == 0 {
		return broker.Host
	}
	return fmt.Sprintf("%s:%d", broker.Host, broker.Port)
}
