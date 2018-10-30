package conn

import (
	"regexp"
	"strings"
)

const listArgSelector int = 1

func cmdList(args commandArgs, c *Conn) {
	if !c.assertAuthenticated(args.ID()) {
		return
	}
	reference := args.Arg(0)
	mailboxname := args.Arg(1)

	if mailboxname == "" {
	// Blank mailboxname means request directory separator
		c.writeResponse("", "LIST (\\Noselect) \"/\" \"\"")
	} else {
		re, error := regexp.Compile(strings.Replace(strings.Replace("^" + reference + mailboxname + "$", "*", ".*", -1), "%", "[^/]*", -1))
		if error != nil { return }
		for _, mailbox := range c.User.Mailboxes() {
			if re.MatchString(mailbox.Name()) {
				c.writeResponse("", "LIST () \"/\" \""+mailbox.Name()+"\"")
			}
		}
	}

	//~ if args.Arg(listArgSelector) == "" {
		//~ // Blank selector means request directory separator
		//~ c.writeResponse("", "LIST (\\Noselect) \"/\" \"\"")
	//~ } else if args.Arg(listArgSelector) == "*" {
		//~ // List all mailboxes requested
		//~ for _, mailbox := range c.User.Mailboxes() {
			//~ c.writeResponse("", "LIST () \"/\" \""+mailbox.Name()+"\"")
		//~ }
	//~ } else if args.Arg(listArgSelector) == "%" {
		//~ // List all mailboxes requested
		//~ for _, mailbox := range c.User.Mailboxes() {
			//~ c.writeResponse("", "LIST () \"/\" \""+mailbox.Name()+"\"")
		//~ }
  //~ }

	c.writeResponse(args.ID(), "OK LIST completed")
}
