package commands

import (
	voicetel "github.com/voicetel/go-sdk"
)

func registerAccount(r *Registry) {
	g := &Group{Name: "account", Description: "Profile, sub-accounts, CDRs, credits, payments, MRC, registration."}
	g.Commands = []*Command{
		{
			Names:    []string{"get"},
			Synopsis: "Fetch the authenticated account profile.",
			Usage:    "account get",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Account().Get(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"update"},
			Synopsis: "Patch account settings (timezone, callerId, notify, ...).",
			Usage:    "account update <json>",
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.AccountPutRequest
				if err := parseJSON("account update", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.Account().Update(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"add"},
			Synopsis: "Create a sub-account (admin-only).",
			Usage:    "account add <json>",
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.AccountAddRequest
				if err := parseJSON("account add", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.Account().Add(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"signup"},
			Synopsis: "Public sign-up flow.",
			Usage:    "account signup <json>",
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.AccountSignupRequest
				if err := parseJSON("account signup", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.Account().Signup(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"cdr"},
			Synopsis: "Fetch call detail records (rate-limited 6/hr/IP).",
			Usage:    "account cdr [start-unix-seconds] [end-unix-seconds]",
			Run: func(c *Context, args []string, _ string) error {
				var start, end int
				var err error
				if len(args) >= 1 {
					if start, err = argInt("account cdr start", args[0]); err != nil {
						return err
					}
				}
				if len(args) >= 2 {
					if end, err = argInt("account cdr end", args[1]); err != nil {
						return err
					}
				}
				out, err := c.Client.Account().CDR(c.Ctx, start, end)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"credits"},
			Synopsis: "Full credit history.",
			Usage:    "account credits",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Account().Credits(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"recurring-charges", "mrc"},
			Synopsis: "Active monthly recurring charges (rate-limited).",
			Usage:    "account recurring-charges  (alias: account mrc)",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Account().RecurringCharges(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"payments"},
			Synopsis: "Full payment history (rate-limited).",
			Usage:    "account payments",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Account().Payments(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"registration"},
			Synopsis: "Current SIP registration (rate-limited).",
			Usage:    "account registration",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Account().Registration(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"recover"},
			Synopsis: "Password recovery flow (no auth required).",
			Usage:    "account recover <json>",
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.AccountRecoverRequest
				if err := parseJSON("account recover", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.Account().Recover(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
	}
	r.AddGroup(g)
}
