package commands

import (
	voicetel "github.com/voicetel/go-sdk"
)

func registerACL(r *Registry) {
	g := &Group{Name: "acl", Description: "IP allowlist (CIDR) management."}
	g.Commands = []*Command{
		{
			Names:    []string{"list"},
			Synopsis: "List the current allowlist.",
			Usage:    "acl list",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.ACL().List(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"add"},
			Synopsis: "Add one or more CIDR entries.",
			Usage:    `acl add <json>  e.g. {"acl":[{"cidr":"203.0.113.0/24"}]}`,
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.AclModifyRequest
				if err := parseJSON("acl add", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.ACL().Add(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"remove"},
			Synopsis: "Remove one or more CIDR entries.",
			Usage:    `acl remove <json>  e.g. {"acl":[{"cidr":"203.0.113.0/24"}]}`,
			Run: func(c *Context, _ []string, tail string) error {
				var body voicetel.AclModifyRequest
				if err := parseJSON("acl remove", tail, &body); err != nil {
					return err
				}
				out, err := c.Client.ACL().Remove(c.Ctx, body)
				return printResultG(c, out, err)
			},
		},
	}
	r.AddGroup(g)
}
