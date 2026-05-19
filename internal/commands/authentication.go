package commands

import (
	voicetel "github.com/voicetel/go-sdk"
)

func registerAuthentication(r *Registry) {
	g := &Group{Name: "authentication", Description: "SIP/HTTP auth mode and password."}
	g.Commands = []*Command{
		{
			Names:    []string{"get"},
			Synopsis: "Read current auth mode + allowlist.",
			Usage:    "authentication get",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Authentication().Get(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"update"},
			Synopsis: "Set the auth mode and/or password.",
			Usage:    `authentication update <json>  e.g. {"authType":1}`,
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.AuthPutRequest
				if err := parseJSON("authentication update", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.Authentication().Update(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
	}
	r.AddGroup(g)
}
