package svc

import "bilibili-web/server/common/health"

type ServiceContext struct {
	Domain string
	Deps   func() map[string]bool
}

func NewServiceContext(domain string) *ServiceContext {
	if domain == "" {
		domain = "video"
	}
	return &ServiceContext{Domain: domain, Deps: health.Dependencies}
}
