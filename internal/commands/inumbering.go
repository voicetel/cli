package commands

import (
	"github.com/voicetel/cli/internal/repl"
	voicetel "github.com/voicetel/go-sdk"
)

func registerINumbering(r *Registry) {
	g := &Group{Name: "inumbering", Description: "Inventory search, orders, port-ins, port-out availability."}
	g.Commands = []*Command{
		{
			Names:    []string{"search-inventory"},
			Synopsis: "Search available TNs by NPA/NXX/state/etc.",
			Usage: "inumbering search-inventory [--npa N] [--nxx N] [--state XX]\n" +
				"                            [--rate-center NAME] [--contains DIGITS]\n" +
				"                            [--ends-with DIGITS] [--limit N]",
			Run: func(c *Context, args []string, _ string) error {
				fs := repl.NewFlagSet()
				npa := fs.RegisterString("npa")
				nxx := fs.RegisterString("nxx")
				state := fs.RegisterString("state")
				rateCenter := fs.RegisterString("rate-center")
				contains := fs.RegisterString("contains")
				endsWith := fs.RegisterString("ends-with")
				limit := fs.RegisterString("limit")
				if err := fs.Parse(args); err != nil {
					return err
				}
				q := voicetel.InventoryQuery{
					State:      *state,
					RateCenter: *rateCenter,
					Contains:   *contains,
					EndsWith:   *endsWith,
				}
				var err error
				if *npa != "" {
					if q.NPA, err = argInt("--npa", *npa); err != nil {
						return err
					}
				}
				if *nxx != "" {
					if q.NXX, err = argInt("--nxx", *nxx); err != nil {
						return err
					}
				}
				if *limit != "" {
					if q.Limit, err = argInt("--limit", *limit); err != nil {
						return err
					}
				}
				out, err := c.Client.INumbering().SearchInventory(c.Ctx, q)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"coverage"},
			Synopsis: "Aggregated availability buckets.",
			Usage:    "inumbering coverage [--state XX] [--rate-center NAME]",
			Run: func(c *Context, args []string, _ string) error {
				fs := repl.NewFlagSet()
				state := fs.RegisterString("state")
				rateCenter := fs.RegisterString("rate-center")
				if err := fs.Parse(args); err != nil {
					return err
				}
				q := voicetel.CoverageQuery{State: *state, RateCenter: *rateCenter}
				out, err := c.Client.INumbering().Coverage(c.Ctx, q)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"order"},
			Synopsis: "Purchase new TNs.",
			Usage:    `inumbering order <json>  e.g. {"numbers":["2015551234","2015551235"]}`,
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.OrderCreateRequest
				if err := parseJSON("inumbering order", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.INumbering().Order(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"ports"},
			Synopsis: "List every port-in record.",
			Usage:    "inumbering ports",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.INumbering().Ports(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"port"},
			Synopsis: "Fetch one port-in by id.",
			Usage:    "inumbering port <id>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("inumbering port", args, 1, "<id>"); err != nil {
					return err
				}
				id, err := argInt("port id", args[0])
				if err != nil {
					return err
				}
				out, err := c.Client.INumbering().Port(c.Ctx, id)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"submit-port"},
			Synopsis: "Submit a port-in order.",
			Usage:    "inumbering submit-port <json>",
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.PortSubmitRequest
				if err := parseJSON("inumbering submit-port", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.INumbering().SubmitPort(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"port-availability"},
			Synopsis: "Check whether a TN can be ported in.",
			Usage:    "inumbering port-availability <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("inumbering port-availability", args, 1, "<number>"); err != nil {
					return err
				}
				out, err := c.Client.INumbering().PortAvailability(c.Ctx, args[0])
				return printResultG(c, out, err)
			},
		},
	}
	r.AddGroup(g)
}
