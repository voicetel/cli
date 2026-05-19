package commands

import (
	"fmt"
	"strconv"
)

// loginCommand swaps username + password for an API key and installs it
// on the active client. Persisted only if onConfigChanged is wired.
func loginCommand() *Command {
	return &Command{
		Names:    []string{"login"},
		Synopsis: "Exchange username + password for a 32-hex API key.",
		Usage: "login <username> <password>\n" +
			"  Username is your numeric account id. Password is never persisted.\n" +
			"  Counts against the 6 req/hr/IP rate limit on account/* endpoints.",
		Run: func(c *Context, args []string, _ string) error {
			if len(args) != 2 {
				return fmt.Errorf("login: expected 2 arguments (username, password), got %d", len(args))
			}
			username, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("login: username must be an integer: %w", err)
			}
			key, err := c.Client.Login(c.Ctx, username, args[1])
			if err != nil {
				return err
			}
			c.Client.SetAPIKey(key)
			if c.OnConfigChanged != nil {
				c.OnConfigChanged()
			}
			c.Printer.Println("Logged in. API key installed and saved.")
			return nil
		},
	}
}

func setCommand() *Command {
	return &Command{
		Names:    []string{"set"},
		Synopsis: "Install an existing API key or override the base URL.",
		Usage: "set api-key <key>     Install an existing 32-hex bearer.\n" +
			"set base-url <url>    Override the API endpoint (rare).",
		Run: func(c *Context, args []string, _ string) error {
			if len(args) < 2 {
				return fmt.Errorf("set: expected `api-key <value>` or `base-url <value>`")
			}
			switch args[0] {
			case "api-key", "apikey":
				c.Client.SetAPIKey(args[1])
				if c.OnConfigChanged != nil {
					c.OnConfigChanged()
				}
				c.Printer.Println("API key installed.")
			case "base-url", "baseurl", "url":
				return fmt.Errorf("set base-url: not mutable after start; restart with --base-url")
			default:
				return fmt.Errorf("set: unknown key %q (use api-key or base-url)", args[0])
			}
			return nil
		},
	}
}

func whoamiCommand() *Command {
	return &Command{
		Names:    []string{"whoami"},
		Synopsis: "Show current API key, base URL, and caveats.",
		Usage: "whoami\n" +
			"  Prints the redacted API key, the configured base URL, and a note\n" +
			"  about the 6 req/hr/IP rate limit on account/* endpoints.",
		Run: func(c *Context, _ []string, _ string) error {
			key := c.Client.APIKey()
			masked := "<not set>"
			if len(key) >= 8 {
				masked = key[:4] + "..." + key[len(key)-4:]
			} else if key != "" {
				masked = "<set>"
			}
			return c.Printer.JSON(map[string]any{
				"apiKey":     masked,
				"baseURL":    c.Client.BaseURL(),
				"rateLimits": "account/cdr, account/mrc, account/payments, account/registration, account/api-key (login): 6 req/hr/IP",
			})
		},
	}
}
