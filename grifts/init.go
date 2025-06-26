package grifts

import (
	"github.com/akingundogdu/production-ready-go-backend-architecture/actions"

	"github.com/gobuffalo/buffalo"
)

func init() {
	buffalo.Grifts(actions.App())
}
