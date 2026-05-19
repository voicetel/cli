package commands

import (
	"strings"

	voicetel "github.com/voicetel/go-sdk"
)

func registerNumbers(r *Registry) {
	g := &Group{Name: "numbers", Description: "TN provisioning, routing, features (CNAM, LIDB, fax, forward, SMS, ...)."}
	g.Commands = []*Command{
		{
			Names:    []string{"list"},
			Synopsis: "Every TN on the account.",
			Usage:    "numbers list",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Numbers().List(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"add"},
			Synopsis: "Attach a TN.",
			Usage:    `numbers add <json>  e.g. {"number":"2015551234"}`,
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.NumberAddRequest
				if err := parseJSON("numbers add", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().Add(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"get"},
			Synopsis: "Fetch one TN.",
			Usage:    "numbers get <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("numbers get", args, 1, "<number>"); err != nil {
					return err
				}
				out, err := c.Client.Numbers().Get(c.Ctx, args[0])
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"remove"},
			Synopsis: "Detach a TN.",
			Usage:    "numbers remove <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("numbers remove", args, 1, "<number>"); err != nil {
					return err
				}
				if err := c.Client.Numbers().Remove(c.Ctx, args[0]); err != nil {
					return err
				}
				c.Printer.Println("Removed.")
				return nil
			},
		},
		{
			Names:    []string{"move"},
			Synopsis: "Move a TN to another account.",
			Usage:    "numbers move <number> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("numbers move", args, 1, "<number> <json>"); err != nil {
					return err
				}
				var body voicetel.NumberMoveRequest
				if err := parseJSON("numbers move", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().Move(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"release"},
			Synopsis: "Release a TN back to the network.",
			Usage:    "numbers release <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("numbers release", args, 1, "<number>"); err != nil {
					return err
				}
				if err := c.Client.Numbers().Release(c.Ctx, args[0]); err != nil {
					return err
				}
				c.Printer.Println("Released.")
				return nil
			},
		},
		{
			Names:    []string{"set-route"},
			Synopsis: "Update a TN's outbound route.",
			Usage:    "numbers set-route <number> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("numbers set-route", args, 1, "<number> <json>"); err != nil {
					return err
				}
				var body voicetel.NumberRouteRequest
				if err := parseJSON("numbers set-route", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().SetRoute(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"set-translation"},
			Synopsis: "Update a TN's DNIS translation.",
			Usage:    "numbers set-translation <number> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("numbers set-translation", args, 1, "<number> <json>"); err != nil {
					return err
				}
				var body voicetel.NumberTranslationRequest
				if err := parseJSON("numbers set-translation", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().SetTranslation(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"set-cnam"},
			Synopsis: "Toggle inbound CNAM lookup.",
			Usage:    "numbers set-cnam <number> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("numbers set-cnam", args, 1, "<number> <json>"); err != nil {
					return err
				}
				var body voicetel.NumberCnamRequest
				if err := parseJSON("numbers set-cnam", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().SetCNAM(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"set-lidb"},
			Synopsis: "Update outbound caller name (LIDB).",
			Usage:    "numbers set-lidb <number> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("numbers set-lidb", args, 1, "<number> <json>"); err != nil {
					return err
				}
				var body voicetel.NumberLidbRequest
				if err := parseJSON("numbers set-lidb", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().SetLidb(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"get-fax"},
			Synopsis: "Read fax-to-email routing.",
			Usage:    "numbers get-fax <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("numbers get-fax", args, 1, "<number>"); err != nil {
					return err
				}
				out, err := c.Client.Numbers().GetFax(c.Ctx, args[0])
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"set-fax"},
			Synopsis: "Enable fax-to-email.",
			Usage:    "numbers set-fax <number> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("numbers set-fax", args, 1, "<number> <json>"); err != nil {
					return err
				}
				var body voicetel.NumberFaxRequest
				if err := parseJSON("numbers set-fax", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().SetFax(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"remove-fax"},
			Synopsis: "Disable fax-to-email.",
			Usage:    "numbers remove-fax <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("numbers remove-fax", args, 1, "<number>"); err != nil {
					return err
				}
				if err := c.Client.Numbers().RemoveFax(c.Ctx, args[0]); err != nil {
					return err
				}
				c.Printer.Println("Removed.")
				return nil
			},
		},
		{
			Names:    []string{"set-forward"},
			Synopsis: "Enable call forwarding.",
			Usage:    "numbers set-forward <number> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("numbers set-forward", args, 1, "<number> <json>"); err != nil {
					return err
				}
				var body voicetel.NumberForwardRequest
				if err := parseJSON("numbers set-forward", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().SetForward(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"remove-forward"},
			Synopsis: "Disable call forwarding.",
			Usage:    "numbers remove-forward <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("numbers remove-forward", args, 1, "<number>"); err != nil {
					return err
				}
				if err := c.Client.Numbers().RemoveForward(c.Ctx, args[0]); err != nil {
					return err
				}
				c.Printer.Println("Removed.")
				return nil
			},
		},
		{
			Names:    []string{"get-sms"},
			Synopsis: "Read SMS routing.",
			Usage:    "numbers get-sms <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("numbers get-sms", args, 1, "<number>"); err != nil {
					return err
				}
				out, err := c.Client.Numbers().GetSMS(c.Ctx, args[0])
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"set-sms"},
			Synopsis: "Configure SMS routing.",
			Usage:    "numbers set-sms <number> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("numbers set-sms", args, 1, "<number> <json>"); err != nil {
					return err
				}
				var body voicetel.NumberSmsRequest
				if err := parseJSON("numbers set-sms", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().SetSMS(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"remove-sms"},
			Synopsis: "Clear SMS routing.",
			Usage:    "numbers remove-sms <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("numbers remove-sms", args, 1, "<number>"); err != nil {
					return err
				}
				if err := c.Client.Numbers().RemoveSMS(c.Ctx, args[0]); err != nil {
					return err
				}
				c.Printer.Println("Removed.")
				return nil
			},
		},
		{
			Names:    []string{"get-messaging"},
			Synopsis: "Read messaging state for one TN.",
			Usage:    "numbers get-messaging <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("numbers get-messaging", args, 1, "<number>"); err != nil {
					return err
				}
				out, err := c.Client.Numbers().GetMessaging(c.Ctx, args[0])
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"patch-messaging"},
			Synopsis: "Update inbound/outbound messaging routing.",
			Usage:    "numbers patch-messaging <number> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("numbers patch-messaging", args, 1, "<number> <json>"); err != nil {
					return err
				}
				var body voicetel.NumberMessagingPatchRequest
				if err := parseJSON("numbers patch-messaging", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().PatchMessaging(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"assign-campaign"},
			Synopsis: "Bind a 10DLC campaign to a TN.",
			Usage:    "numbers assign-campaign <number> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("numbers assign-campaign", args, 1, "<number> <json>"); err != nil {
					return err
				}
				var body voicetel.NumberCampaignAssignRequest
				if err := parseJSON("numbers assign-campaign", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().AssignCampaign(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"unassign-campaign"},
			Synopsis: "Remove the campaign binding from a TN.",
			Usage:    "numbers unassign-campaign <number>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("numbers unassign-campaign", args, 1, "<number>"); err != nil {
					return err
				}
				out, err := c.Client.Numbers().UnassignCampaign(c.Ctx, args[0])
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"bulk-unassign-campaign"},
			Synopsis: "Remove campaign bindings from many TNs at once.",
			Usage:    "numbers bulk-unassign-campaign <2015551234,2015551235>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("numbers bulk-unassign-campaign", args, 1, "<csv of numbers>"); err != nil {
					return err
				}
				ns := strings.Split(args[0], ",")
				out, err := c.Client.Numbers().BulkUnassignCampaign(c.Ctx, ns)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"set-port-out-pin"},
			Synopsis: "Set the port-out PIN.",
			Usage:    "numbers set-port-out-pin <number> <json>",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("numbers set-port-out-pin", args, 1, "<number> <json>"); err != nil {
					return err
				}
				var body voicetel.PortOutPinUpdateRequest
				if err := parseJSON("numbers set-port-out-pin", skipArgs(tail, 1), &body); err != nil {
					return err
				}
				out, err := c.Client.Numbers().SetPortOutPin(c.Ctx, args[0], body)
				return printResultG(c, out, err)
			},
		},
	}
	r.AddGroup(g)
}
