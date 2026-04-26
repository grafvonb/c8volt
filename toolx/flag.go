// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import "time"

type DurationStringValue struct {
	value *string
}

func NewDurationStringValue(defaultValue string, target *string) *DurationStringValue {
	if target == nil {
		target = new(string)
	}
	*target = defaultValue
	return &DurationStringValue{value: target}
}

func (d *DurationStringValue) Set(value string) error {
	if _, err := time.ParseDuration(value); err != nil {
		return err
	}
	*d.value = value
	return nil
}

func (d *DurationStringValue) String() string {
	if d == nil || d.value == nil {
		return ""
	}
	return *d.value
}

func (d *DurationStringValue) Type() string { return "duration" }
