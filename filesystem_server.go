package main

import (
	"fmt"
	"os"

	//imap "github.com/ramoncasares/ro-imap-server.go"
	"github.com/ramoncasares/ro-imap-server.go/mailstore"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Error: %s requires the root of the filesystem\n", os.Args[0])
		fmt.Printf("   Syntax: %s Root_of_Filesystem\n", os.Args[0])
		os.Exit(1)
	}
	store := mailstore.NewFilesystemMailstore(os.Args[1])
	s := NewServer(store)
	s.Transcript = os.Stdout
	s.Addr = ":10143"

	err := s.ListenAndServe()
	if err != nil {
		fmt.Printf("Error creating connection: %s\n", err)
		os.Exit(2)
	}
}
