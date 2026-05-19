package commands

import (
	voicetel "github.com/voicetel/go-sdk"
)

func registerGateways(r *Registry) {
	g := &Group{Name: "gateways", Description: "Outbound termination gateways."}
	g.Commands = []*Command{
		{
			Names:    []string{"list"},
			Synopsis: "List every gateway.",
			Usage:    "gateways list",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Gateways().List(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"add"},
			Synopsis: "Create a new gateway.",
			Usage:    `gateways add <json>  e.g. {"gateway":"203.0.113.10","limit":50}`,
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.GatewayAddRequest
				if err := parseJSON("gateways add", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.Gateways().Add(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"get"},
			Synopsis: "Fetch a single gateway.",
			Usage:    "gateways get <id>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("gateways get", args, 1, "<id>"); err != nil {
					return err
				}
				id, err := argInt("gateways get id", args[0])
				if err != nil {
					return err
				}
				out, err := c.Client.Gateways().Get(c.Ctx, id)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"update"},
			Synopsis: "Patch a gateway.",
			Usage:    "gateways update <id> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("gateways update", args, 1, "<id> <json>"); err != nil {
					return err
				}
				id, err := argInt("gateways update id", args[0])
				if err != nil {
					return err
				}
				var body voicetel.GatewayUpdateRequest
				if err := parseJSON("gateways update", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Gateways().Update(c.Ctx, id, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"remove"},
			Synopsis: "Delete a gateway.",
			Usage:    "gateways remove <id>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("gateways remove", args, 1, "<id>"); err != nil {
					return err
				}
				id, err := argInt("gateways remove id", args[0])
				if err != nil {
					return err
				}
				if err := c.Client.Gateways().Remove(c.Ctx, id); err != nil {
					return err
				}
				c.Printer.Println("Removed.")
				return nil
			},
		},
		{
			Names:    []string{"numbers"},
			Synopsis: "List every TN routed through a gateway.",
			Usage:    "gateways numbers <id>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("gateways numbers", args, 1, "<id>"); err != nil {
					return err
				}
				id, err := argInt("gateways numbers id", args[0])
				if err != nil {
					return err
				}
				out, err := c.Client.Gateways().Numbers(c.Ctx, id)
				return printResultG(c, out, err)
			},
		},
	}
	r.AddGroup(g)
}
