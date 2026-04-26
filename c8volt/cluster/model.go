// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cluster

import "time"

type Topology struct {
	Brokers               []Broker
	ClusterSize           int32
	GatewayVersion        string
	PartitionsCount       int32
	ReplicationFactor     int32
	LastCompletedChangeId string
}

type Broker struct {
	Host       string
	NodeId     int32
	Partitions []Partition
	Port       int32
	Version    string
}

type Partition struct {
	Health      PartitionHealth
	PartitionId int32
	Role        PartitionRole
}

type PartitionHealth string
type PartitionRole string

type License struct {
	ExpiresAt    *time.Time `json:"ExpiresAt,omitempty"`
	IsCommercial *bool      `json:"IsCommercial,omitempty"`
	LicenseType  string     `json:"LicenseType"`
	ValidLicense bool       `json:"ValidLicense"`
}
