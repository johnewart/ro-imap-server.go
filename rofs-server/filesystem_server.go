package main

import (
	"fmt"
	"os"

	imap "ro-imap-server.go"
	"ro-imap-server.go/mailstore"
)

func main() {
	store := mailstore.NewFilesystemMailstore(".")
	s := imap.NewServer(store)
	s.Transcript = os.Stdout
	s.Addr = ":10143"

	err := s.ListenAndServe()
	if err != nil {
		fmt.Printf("Error creating test connection: %s\n", err)
	}
}