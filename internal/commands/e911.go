package commands

import (
	voicetel "github.com/voicetel/go-sdk"
)

func registerE911(r *Registry) {
	g := &Group{Name: "e911", Description: "e911 record provisioning and address validation."}
	g.Commands = []*Command{
		{
			Names:    []string{"list"},
			Synopsis: "List every e911 record.",
			Usage:    "e911 list",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.E911().List(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"create"},
			Synopsis: "Validate + provision in one call.",
			Usage:    "e911 create <json>",
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.E911CreateRequest
				if err := parseJSON("e911 create", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.E911().Create(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"validate"},
			Synopsis: "Validate a street address (returns addressid).",
			Usage:    "e911 validate <json>",
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.E911AddressRequest
				if err := parseJSON("e911 validate", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.E911().Validate(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"get"},
			Synopsis: "Fetch the e911 record for a TN.",
			Usage:    "e911 get <dn>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("e911 get", args, 1, "<dn>"); err != nil {
					return err
				}
				out, err := c.Client.E911().Get(c.Ctx, args[0])
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"provision"},
			Synopsis: "Provision e911 for a TN using a previously-validated addressid.",
			Usage:    "e911 provision <dn> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("e911 provision", args, 1, "<dn> <json>"); err != nil {
					return err
				}
				var body voicetel.E911ProvisionByIDRequest
				if err := parseJSON("e911 provision", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.E911().Provision(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"remove"},
			Synopsis: "Delete the e911 record for a TN.",
			Usage:    "e911 remove <dn>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("e911 remove", args, 1, "<dn>"); err != nil {
					return err
				}
				if err := c.Client.E911().Remove(c.Ctx, args[0]); err != nil {
					return err
				}
				c.Printer.Println("Removed.")
				return nil
			},
		},
	}
	r.AddGroup(g)
}
