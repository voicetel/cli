package commands

import (
	"fmt"
	"strings"

	"github.com/voicetel/cli/internal/repl"
	voicetel "github.com/voicetel/go-sdk/v2"
)

func registerMessaging(r *Registry) {
	g := &Group{Name: "messaging", Description: "SMS/MMS, message history, 10DLC brands & campaigns, messaging health."}
	g.Commands = []*Command{
		{
			Names:    []string{"history"},
			Synopsis: "Fetch message history for one number.",
			Usage:    "messaging history --number TN [--from unix] [--to unix] [--type sms|mms|dlr]",
			Run: func(c *Context, args []string, _ string) error {
				fs := repl.NewFlagSet()
				number := fs.RegisterString("number")
				from := fs.RegisterString("from")
				to := fs.RegisterString("to")
				typ := fs.RegisterString("type")
				if err := fs.Parse(args); err != nil {
					return err
				}
				if *number == "" {
					return fmt.Errorf("messaging history: --number is required")
				}
				opts := voicetel.HistoryOptions{Number: *number, Type: *typ}
				var err error
				if *from != "" {
					if opts.From, err = argInt("--from", *from); err != nil {
						return err
					}
				}
				if *to != "" {
					if opts.To, err = argInt("--to", *to); err != nil {
						return err
					}
				}
				out, err := c.Client.Messaging().History(c.Ctx, opts)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"send"},
			Synopsis: "Send an SMS or MMS.",
			Usage:    `messaging send <json>  e.g. {"fromNumber":"2015550123","toNumber":"2015555678","text":"hi"}`,
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.MessageSendRequest
				if err := parseJSON("messaging send", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.Messaging().Send(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"create-brand"},
			Synopsis: "Register a 10DLC brand.",
			Usage:    "messaging create-brand <json>",
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.MessagingBrandCreateRequest
				if err := parseJSON("messaging create-brand", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.Messaging().CreateBrand(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"campaign-status"},
			Synopsis: "Read the status of the given campaigns.",
			Usage:    "messaging campaign-status <id[,id...]>\n  Comma-separated alphanumeric campaign ids; max 100.",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("messaging campaign-status", args, 1, "<id[,id...]>"); err != nil {
					return err
				}
				out, err := c.Client.Messaging().CampaignStatus(c.Ctx, strings.Split(args[0], ","))
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"create-campaign"},
			Synopsis: "Register a 10DLC campaign.",
			Usage:    "messaging create-campaign <json>",
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.MessagingCampaignCreateRequest
				if err := parseJSON("messaging create-campaign", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.Messaging().CreateCampaign(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"numbers-state"},
			Synopsis: "Messaging state for many TNs.",
			Usage:    "messaging numbers-state --numbers=2015550123,2015551235\n  Comma-separated 10-digit numbers (required); max 1000.",
			Run: func(c *Context, args []string, _ string) error {
				fs := repl.NewFlagSet()
				numbers := fs.RegisterString("numbers")
				if err := fs.Parse(args); err != nil {
					return err
				}
				if *numbers == "" {
					return fmt.Errorf("messaging numbers-state: --numbers is required")
				}
				ns := strings.Split(*numbers, ",")
				out, err := c.Client.Messaging().NumbersState(c.Ctx, ns)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"health"},
			Synopsis: "Account-wide messaging health overview (portal Messaging tab).",
			Usage:    "messaging health",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Messaging().Health(c.Ctx)
				return printResultG(c, &out, err)
			},
		},
		{
			Names:    []string{"health-alerts"},
			Synopsis: "Read the alert-email setting.",
			Usage:    "messaging health-alerts",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Messaging().HealthAlerts(c.Ctx)
				return printResultG(c, &out, err)
			},
		},
		{
			Names:    []string{"set-health-alerts"},
			Synopsis: "Turn alert emails on or off.",
			Usage:    "messaging set-health-alerts <true|false>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("messaging set-health-alerts", args, 1, "<true|false>"); err != nil {
					return err
				}
				enabled, err := argBool("messaging set-health-alerts", args[0])
				if err != nil {
					return err
				}
				out, err := c.Client.Messaging().SetHealthAlerts(c.Ctx, enabled)
				return printResultG(c, &out, err)
			},
		},
		{
			Names:    []string{"health-check"},
			Synopsis: "Live health check of one number (30/hour/account).",
			Usage:    "messaging health-check <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("messaging health-check", args, 1, "<number>"); err != nil {
					return err
				}
				out, err := c.Client.Messaging().HealthCheck(c.Ctx, args[0])
				return printResultG(c, &out, err)
			},
		},
		{
			Names:    []string{"health-samples"},
			Synopsis: "Recent outbound messages from a number blocked as spam.",
			Usage:    "messaging health-samples <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("messaging health-samples", args, 1, "<number>"); err != nil {
					return err
				}
				out, err := c.Client.Messaging().HealthSamples(c.Ctx, args[0])
				return printResultG(c, &out, err)
			},
		},
	}
	r.AddGroup(g)
}
