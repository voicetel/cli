package commands

import (
	voicetel "github.com/voicetel/go-sdk"
)

func registerSupport(r *Registry) {
	g := &Group{Name: "support", Description: "Support tickets (CRUD, threaded messages, replies)."}
	g.Commands = []*Command{
		{
			Names:    []string{"list"},
			Synopsis: "List every ticket.",
			Usage:    "support list",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Support().List(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"create"},
			Synopsis: "Open a new support ticket.",
			Usage:    "support create <json>",
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.TicketCreateRequest
				if err := parseJSON("support create", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.Support().Create(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"get"},
			Synopsis: "Fetch one ticket by id.",
			Usage:    "support get <id>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("support get", args, 1, "<id>"); err != nil {
					return err
				}
				id, err := argInt("support get id", args[0])
				if err != nil {
					return err
				}
				out, err := c.Client.Support().Get(c.Ctx, id)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"update"},
			Synopsis: "Change a ticket status.",
			Usage:    `support update <id> <json>  e.g. {"status":"closed"}`,
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("support update", args, 1, "<id> <json>"); err != nil {
					return err
				}
				id, err := argInt("support update id", args[0])
				if err != nil {
					return err
				}
				var body voicetel.TicketUpdateRequest
				if err := parseJSON("support update", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Support().Update(c.Ctx, id, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"delete"},
			Synopsis: "Delete a ticket (admin-only).",
			Usage:    "support delete <id>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("support delete", args, 1, "<id>"); err != nil {
					return err
				}
				id, err := argInt("support delete id", args[0])
				if err != nil {
					return err
				}
				if err := c.Client.Support().Delete(c.Ctx, id); err != nil {
					return err
				}
				c.Printer.Println("Deleted.")
				return nil
			},
		},
		{
			Names:    []string{"messages"},
			Synopsis: "List every thread (message) on a ticket.",
			Usage:    "support messages <id>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("support messages", args, 1, "<id>"); err != nil {
					return err
				}
				id, err := argInt("support messages id", args[0])
				if err != nil {
					return err
				}
				out, err := c.Client.Support().Messages(c.Ctx, id)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"reply"},
			Synopsis: "Reply to a ticket.",
			Usage:    "support reply <id> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("support reply", args, 1, "<id> <json>"); err != nil {
					return err
				}
				id, err := argInt("support reply id", args[0])
				if err != nil {
					return err
				}
				var body voicetel.TicketReplyRequest
				if err := parseJSON("support reply", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Support().Reply(c.Ctx, id, body)
				return printResultG(c, out, err)
			},
		},
	}
	r.AddGroup(g)
}
