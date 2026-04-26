// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"fmt"
	"log/slog"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	servicecommon "github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

func GetClusterLicense[T any](
	ctx context.Context,
	log *slog.Logger,
	baseURL string,
	opts []services.CallOption,
	fetch func(context.Context) (PayloadResponse[T], error),
	convert func(T) d.License,
) (d.License, error) {
	callCfg := services.ApplyCallOptions(opts)
	servicecommon.VerboseLog(ctx, callCfg, log, "requesting cluster license", "baseURL", baseURL)

	resp, err := fetch(ctx)
	if err != nil {
		return d.License{}, err
	}
	if !resp.Received {
		return d.License{}, fmt.Errorf("%w: license response is nil", d.ErrMalformedResponse)
	}
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return d.License{}, err
	}
	if resp.Payload == nil {
		return d.License{}, fmt.Errorf("%w: 200 OK but empty payload; body=%s",
			d.ErrMalformedResponse, string(resp.Body))
	}

	license := convert(*resp.Payload)
	servicecommon.VerboseLog(ctx, callCfg, log, "cluster license retrieved", "licenseType", license.LicenseType, "validLicense", license.ValidLicense)
	return license, nil
}
