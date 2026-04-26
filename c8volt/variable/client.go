// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

type client struct {
	pdApi pdsvc.API
	piApi pisvc.API
}

func New(pdApi pdsvc.API, piApi pisvc.API) API {
	return &client{
		pdApi: pdApi,
		piApi: piApi,
	}
}
