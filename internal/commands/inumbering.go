package commands

import (
	"fmt"

	"github.com/voicetel/cli/internal/repl"
	voicetel "github.com/voicetel/go-sdk/v2"
)

func registerINumbering(r *Registry) {
	g := &Group{Name: "inumbering", Description: "Inventory search, orders, port-ins, port-out availability."}
	g.Commands = []*Command{
		{
			Names:    []string{"search-inventory"},
			Synopsis: "Search available TNs by prefix/wildcard mask.",
			Usage: "inumbering search-inventory --tn-mask MASK [--quantity N]\n" +
				"                            [--province XX] [--rate-center NAME] [--lata N]\n" +
				"  MASK is a prefix or wildcard pattern: 201, 201555, 201555* (required).",
			Run: func(c *Context, args []string, _ string) error {
				fs := repl.NewFlagSet()
				tnMask := fs.RegisterString("tn-mask")
				quantity := fs.RegisterString("quantity")
				province := fs.RegisterString("province")
				rateCenter := fs.RegisterString("rate-center")
				lata := fs.RegisterString("lata")
				if err := fs.Parse(args); err != nil {
					return err
				}
				if *tnMask == "" {
					return fmt.Errorf("inumbering search-inventory: --tn-mask is required (e.g. --tn-mask=201555*)")
				}
				q := voicetel.InventoryQuery{
					TnMask:     *tnMask,
					Province:   *province,
					RateCenter: *rateCenter,
				}
				var err error
				if *quantity != "" {
					if q.Quantity, err = argInt("--quantity", *quantity); err != nil {
						return err
					}
				}
				q.Lata = *lata
				out, err := c.Client.INumbering().SearchInventory(c.Ctx, q)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"coverage"},
			Synopsis: "Aggregated availability buckets.",
			Usage: "inumbering coverage --count-by state|city|rateCenter|npanxx|block\n" +
				"                     [--loc-state XX] [--city NAME] [--rc-abbre NAME]\n" +
				"                     [--npa NPA] [--lata N]",
			Run: func(c *Context, args []string, _ string) error {
				fs := repl.NewFlagSet()
				countBy := fs.RegisterString("count-by")
				locState := fs.RegisterString("loc-state")
				city := fs.RegisterString("city")
				rcAbbre := fs.RegisterString("rc-abbre")
				npa := fs.RegisterString("npa")
				lata := fs.RegisterString("lata")
				if err := fs.Parse(args); err != nil {
					return err
				}
				if *countBy == "" {
					return fmt.Errorf("inumbering coverage: --count-by is required (state|city|rateCenter|npanxx|block)")
				}
				q := voicetel.CoverageQuery{
					CountBy:  *countBy,
					LocState: *locState,
					City:     *city,
					RcAbbre:  *rcAbbre,
					NPA:      *npa,
				}
				var err error
				q.Lata = *lata
				out, err := c.Client.INumbering().Coverage(c.Ctx, q)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"order"},
			Synopsis: "Purchase new TNs.",
			Usage:    `inumbering order <json>  e.g. {"numbers":["2015550123","2015551235"]}`,
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
			Usage:    "inumbering port <id>\n  <id> is the 5-character pid returned by submit-port, or an id from `inumbering ports`.",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("inumbering port", args, 1, "<id>"); err != nil {
					return err
				}
				out, err := c.Client.INumbering().Port(c.Ctx, args[0])
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
