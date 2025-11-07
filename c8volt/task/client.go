package task

import (
	"log/slog"

	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

type client struct {
	pdApi pdsvc.API
	piApi pisvc.API
	log   *slog.Logger
}

func New(pdApi pdsvc.API, piApi pisvc.API, log *slog.Logger) API {
	return &client{
		pdApi: pdApi,
		piApi: piApi,
		log:   log,
	}
}
