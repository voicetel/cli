package commands

import (
	"fmt"
	"os"
	"strings"

	voicetel "github.com/voicetel/go-sdk/v2"
)

func registerFax(r *Registry) {
	g := &Group{Name: "fax", Description: "Send faxes from fax-capable numbers; history and documents."}
	g.Commands = []*Command{
		{
			Names:    []string{"numbers"},
			Synopsis: "List fax-capable numbers and their delivery emails.",
			Usage:    "fax numbers",
			Run: func(c *Context, _ []string, _ string) error {
				out, err := c.Client.Fax().ListNumbers(c.Ctx)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"send"},
			Synopsis: "Send a fax (multipart; idempotent per submission id).",
			Usage:    "fax send <fromNumber> <toNumber> <file.pdf> [file2.pdf ...] [id <uuid>]",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("fax send", args, 3, "<fromNumber> <toNumber> <file...>"); err != nil {
					return err
				}
				rest := strings.Fields(tail)
				idempotencyKey := ""
				files := append([]string{}, args[2:]...)
				if len(rest) >= 2 && rest[0] == "id" {
					idempotencyKey = rest[1]
					files = files[:len(files)-2]
				}
				docs := make([]voicetel.FaxDocumentFile, 0, len(files))
				for _, path := range files {
					content, err := os.ReadFile(path)
					if err != nil {
						return fmt.Errorf("fax send: read %s: %w", path, err)
					}
					docs = append(docs, voicetel.FaxDocumentFile{
						Name:        path,
						ContentType: "application/pdf",
						Content:     content,
					})
				}
				out, err := c.Client.Fax().Send(c.Ctx, args[0], args[1], docs, idempotencyKey)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"history"},
			Synopsis: "Fax history for one number (newest first).",
			Usage:    "fax history <number> [limit N] [status all|active|queued|sending|retry_pending|sent|failed]",
			Run: func(c *Context, args []string, tail string) error {
				if err := requireArgs("fax history", args, 1, "<number> [limit N] [status S]"); err != nil {
					return err
				}
				rest := strings.Fields(tail)
				opts := &voicetel.FaxHistoryOptions{}
				for i := 0; i+1 < len(rest); i += 2 {
					switch rest[i] {
					case "limit":
						fmt.Sscanf(rest[i+1], "%d", &opts.Limit)
					case "status":
						opts.Status = rest[i+1]
					}
				}
				out, err := c.Client.Fax().History(c.Ctx, args[0], opts)
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"get"},
			Synopsis: "Fetch one fax job.",
			Usage:    "fax get <id>",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("fax get", args, 1, "<id>"); err != nil {
					return err
				}
				out, err := c.Client.Fax().Get(c.Ctx, args[0])
				return printResultG(c, out, err)
			},
		},
		{
			Names:    []string{"document"},
			Synopsis: "Download a fax document as PDF.",
			Usage:    "fax document <id> [out.pdf]",
			Run: func(c *Context, args []string, _ string) error {
				if err := requireArgs("fax document", args, 1, "<id> [out.pdf]"); err != nil {
					return err
				}
				pdf, err := c.Client.Fax().Document(c.Ctx, args[0])
				if err != nil {
					return err
				}
				out := args[0] + ".pdf"
				if len(args) > 1 {
					out = args[1]
				}
				if err := os.WriteFile(out, pdf, 0o600); err != nil {
					return fmt.Errorf("fax document: write %s: %w", out, err)
				}
				c.Printer.Printf("Wrote %s (%d bytes).", out, len(pdf))
				return nil
			},
		},
	}
	r.AddGroup(g)
}
