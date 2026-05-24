package main

import (
	"context"

	"github.com/optiqor/optiqor-cli/internal/config"
)

// Unexported context-key types so cobra-tree values don't collide.
type colorPolicyKey struct{}

func withColorPolicy(ctx context.Context, useColor bool) context.Context {
	return context.WithValue(ctx, colorPolicyKey{}, useColor)
}

// colorPolicyFrom defaults to false (plain) so subcommands bypassing
// the persistent pre-run stay safe-by-default.
func colorPolicyFrom(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, ok := ctx.Value(colorPolicyKey{}).(bool)
	if !ok {
		return false
	}
	return v
}

type configKey struct{}

func withConfig(ctx context.Context, c config.Config) context.Context {
	return context.WithValue(ctx, configKey{}, c)
}

// configFrom returns the zero Config when none is set so callers
// skip the nil-check.
func configFrom(ctx context.Context) config.Config {
	if ctx == nil {
		return config.Config{}
	}
	v, ok := ctx.Value(configKey{}).(config.Config)
	if !ok {
		return config.Config{}
	}
	return v
}
