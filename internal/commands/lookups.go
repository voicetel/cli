package commands

func registerLookups(r *Registry) {
	g := &Group{Name: "lookups", Description: "CNAM and LRN dips."}
	g.Commands = []*Command{
		{
			Names:    []string{"cnam"},
			Synopsis: "Perform a CNAM dip.",
			Usage:    "lookups cnam <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("lookups cnam", args, 1, "<number>"); err != nil {
					return err
				}
				out, err := c.Client.Lookups().CNAM(c.Ctx, args[0])
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"lrn"},
			Synopsis: "Perform an LRN dip.",
			Usage:    "lookups lrn <number> <ani>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("lookups lrn", args, 2, "<number> <ani>"); err != nil {
					return err
				}
				out, err := c.Client.Lookups().LRN(c.Ctx, args[0], args[1])
				return printResultG(c, out, err)
			},
		},
	}
	r.AddGroup(g)
}
