// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/cluster"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

// configConnectionFailureMessage renders only typed, bounded failure evidence.
// Detailed HTTP records remain in the existing redacted DEBUG diagnostics path.
func configConnectionFailureMessage(err error) string {
	prefix := "config test-connection: "
	if commandHasRequestRateWaitFailure(err) {
		return prefix + boundedCommandFailureReason(err)
	}
	failure, reason, status, authentication := httpc.FailureDetails(err)
	target := "gateway"
	if authentication {
		target = "authentication service"
	}
	switch failure {
	case "CANCELED":
		return prefix + "connection test canceled"
	case "TIMEOUT":
		return prefix + target + " timed out; check service availability, network access and configured timeout"
	case "DNS_ERROR":
		return prefix + "cannot resolve " + target + " host; check the configured hostname and DNS"
	case "TLS_ERROR":
		return prefix + target + " TLS certificate validation failed; check the certificate trust, hostname and expiry"
	case "INVALID_RESPONSE":
		return prefix + "invalid " + target + " response; check the configured URL and proxy routing"
	}
	switch reason {
	case "connection-refused":
		return prefix + target + " connection refused; check the configured host and port and that the service is running"
	case "connection-reset", "unexpected-eof":
		return prefix + target + " connection interrupted; check the service and proxy"
	}
	if status != 0 {
		var hint string
		switch status {
		case 401:
			hint = "authentication rejected; check the selected profile and credentials"
		case 403:
			hint = "access denied; check credentials, roles and API permissions"
		case 404:
			hint = "topology endpoint not found; check the Camunda API base URL and proxy routing"
		case 429:
			hint = "rate limit reached; retry later or reduce request rate"
		case 500, 502, 503, 504:
			hint = "service error; check service health and proxy logs"
		default:
			hint = "request rejected; check the selected profile and service configuration"
		}
		if authentication && status == 400 {
			hint = "authentication request rejected; check client credentials and scope configuration"
		}
		if authentication && status == 404 {
			hint = "authentication endpoint not found; check the authentication URL and proxy routing"
		}
		return fmt.Sprintf("%s%s HTTP %d: %s", prefix, target, status, hint)
	}
	if authentication {
		return prefix + "authentication failed; check the authentication configuration; use --debug for diagnostic details"
	}
	switch ferrors.Classify(err) {
	case ferrors.ClassTimeout:
		return prefix + "gateway timed out; check service availability, network access and configured timeout"
	case ferrors.ClassMalformedResponse:
		return prefix + "invalid gateway response; check the Camunda API base URL and proxy routing"
	case ferrors.ClassLocalPrecondition, ferrors.ClassInvalidInput:
		return prefix + "connection setup failed; check the selected profile and configuration"
	default:
		return prefix + "unexpected connection failure; use --debug for diagnostic details"
	}
}

type configTestConnectionView struct {
	OK            bool                            `json:"ok"`
	ConfigFile    string                          `json:"config_file,omitempty"`
	Profile       string                          `json:"profile,omitempty"`
	BaseURL       string                          `json:"base_url"`
	TenantContext *tenant.Context                 `json:"tenantContext,omitempty"`
	Cluster       configTestConnectionClusterView `json:"cluster"`
	Warnings      []string                        `json:"warnings"`
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

func newConfigTestConnectionView(cfg *config.Config, source configSourceDescription, topology cluster.Topology, warnings []string, tenantCtx tenant.Context) configTestConnectionView {
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
	if !tenantContextIsZero(tenantCtx) {
		cloned := cloneTenantContext(tenantCtx)
		view.TenantContext = &cloned
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
