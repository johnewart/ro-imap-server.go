package imap

// Handles a CAPABILITY command
func cmdCapability(args commandArgs, c *Conn) {
	c.writeResponse("", "CAPABILITY IMAP4rev1 AUTH=PLAIN")
	c.writeResponse(args.Id(), "OK CAPABILITY completed")
}