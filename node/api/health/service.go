// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package health

import (
	"net/http"

	"github.com/luxfi/log"
)

type Service struct {
	log    log.Logger
	health Reporter
}

// Readiness returns if the node has finished initialization
func (s *Service) Readiness(_ *http.Request, args *APIArgs, reply *APIReply) error {
	s.log.Debug("API called",
		log.UserString("service", "health"),
		log.UserString("method", "readiness"),
		log.Reflect("tags", args.Tags),
	)
	reply.Checks, reply.Healthy = s.health.Readiness(args.Tags...)
	return nil
}

// Health returns a summation of the health of the node
func (s *Service) Health(_ *http.Request, args *APIArgs, reply *APIReply) error {
	s.log.Debug("API called",
		log.UserString("service", "health"),
		log.UserString("method", "health"),
		log.Reflect("tags", args.Tags),
	)

	reply.Checks, reply.Healthy = s.health.Health(args.Tags...)
	return nil
}

// Liveness returns if the node is in need of a restart
func (s *Service) Liveness(_ *http.Request, args *APIArgs, reply *APIReply) error {
	s.log.Debug("API called",
		log.UserString("service", "health"),
		log.UserString("method", "liveness"),
		log.Reflect("tags", args.Tags),
	)
	reply.Checks, reply.Healthy = s.health.Liveness(args.Tags...)
	return nil
}
