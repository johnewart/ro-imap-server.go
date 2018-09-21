package mailstore

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/mail"
	"os"
	"path/filepath"
	"time"

	"github.com/ramoncasares/ro-imap-server.go/types"
)

// FilesystemMailstore is a filesystem mail storage
// It points to a dir in a filesystem
type FilesystemMailstore struct {
	dirname   string
	users     []*FilesystemUser
	mailboxes []*FilesystemMailbox
}

// FilesystemUser is a representation of a user
type FilesystemUser struct {
	username      string
	password      string
	authenticated bool
	mailstore     *FilesystemMailstore
}

// FilesystemMailbox is a filesystem implementation of a Mailstore Mailbox
// It points to a subdir of the Mailstore
type FilesystemMailbox struct {
	path       string
	info       *os.FileInfo
	ID         uint32
	messages   []*FilesystemMessage
}

// FilesystemMessage is a representation of a single message in a FilesystemMailbox.
// It points to a file
type FilesystemMessage struct {
	path     string
	info     *os.FileInfo
	ID       uint32
	flags    types.Flags
}

// NewFilesystemMailstore performs some initialisation and should always be
// used to create a new FilesystemMailstore
func NewFilesystemMailstore(dirname string) *FilesystemMailstore {

	err := os.Chdir(dirname)
	if err != nil {
		return nil
	}

	fmt.Printf("Creating mailstore in %v\n", dirname)

	users := make([]*FilesystemUser, 1)
	user := &FilesystemUser{
		username:      "username",
		password:      "password",
		authenticated: false,
		mailstore:     nil,
	}
	users[0] = user
	mailboxes := make([]*FilesystemMailbox, 0)

	ms := &FilesystemMailstore{
		dirname:   dirname,
		users:     users,
		mailboxes: mailboxes,
	}

	var mbID, emID, temID uint32

	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Walking has failed: %s\n", err)
			return err
		}
		if info.IsDir() {
			mb := &FilesystemMailbox{
				path:       path,
				info:       &info,
				ID:         mbID,
				messages:   make([]*FilesystemMessage, 0),
			}
			ms.mailboxes = append(ms.mailboxes, mb)
			mbID++
			emID = 0
		} else {
			mb := ms.mailboxes[mbID-1]
			ma := &FilesystemMessage{
				path:     path,
				info:     &info,
				ID:       emID,
				flags:    types.Flags(1),
			}
			mb.messages = append(mb.messages, ma)
			emID++
			temID++
		}
		return nil
	})
	if err != nil {
		return nil
	}
	fmt.Printf("Mailboxes = %v, Emails = %v\n", mbID, temID)
	return ms
}

// Authenticate implements the Authenticate method on the Mailstore interface
func (d *FilesystemMailstore) Authenticate(username string, password string) (User, error) {
	if username != (*d.users[0]).username {
		return &FilesystemUser{}, errors.New("Invalid username. Use 'username'")
	}
	if password != (*d.users[0]).password {
		return &FilesystemUser{}, errors.New("Invalid password. Use 'password'")
	}
	(*d.users[0]).authenticated = true
	(*d.users[0]).mailstore = d
	return *d.users[0], nil
}

// Mailboxes implements the Mailboxes method on the User interface
func (u FilesystemUser) Mailboxes() []Mailbox {
	original := u.mailstore.mailboxes
	mailboxes := make([]Mailbox, len(original))
	for i, _ := range original {
		mailboxes[i] = (original[i])
	}
	return mailboxes
}

// MailboxByName implements the MailboxByName method on the User interface
func (u FilesystemUser) MailboxByName(name string) (Mailbox, error) {
	for _, mailbox := range u.mailstore.mailboxes {
		if mailbox.Name() == name {
			return mailbox, nil
		}
	}
	return nil, errors.New("Invalid mailbox")
}

// DebugPrintMailbox prints out all messages in the mailbox to the command line
// for debugging purposes
func (m *FilesystemMailbox) DebugPrintMailbox() {
	seqset, _ := types.InterpretSequenceSet("*")
	debugPrintMessages(m.MessageSetByUID(seqset))
}

// Name returns the Mailbox's name
//func (m *FilesystemMailbox) Name() string { return m.subdirname }
func (m *FilesystemMailbox) Name() string { return m.path }

// NextUID returns the UID that is likely to be assigned to the next
// new message in the Mailbox
func (m *FilesystemMailbox) NextUID() uint32 { return uint32(len(m.messages) + 1) }

// LastUID returns the UID of the last message in the mailbox or if the
// mailbox is empty, the next expected UID
func (m *FilesystemMailbox) LastUID() uint32 { return uint32(len(m.messages)) }

// Messages returns the total number of messages in the Mailbox
func (m *FilesystemMailbox) Messages() uint32 { return uint32(len(m.messages)) }

// Recent returns the number of messages in the mailbox which are currently
// marked with the 'Recent' flag
func (m *FilesystemMailbox) Recent() uint32 {
	var count uint32
	for _, message := range m.messages {
		if message.Flags().HasFlags(types.FlagRecent) {
			count++
		}
	}
	return count
}

// Unseen returns the number of messages in the mailbox which are currently
// marked with the 'Unseen' flag
func (m *FilesystemMailbox) Unseen() uint32 {
	count := uint32(0)
	for _, message := range m.messages {
		if !message.Flags().HasFlags(types.FlagSeen) {
			count++
		}
	}
	return count
}

// MessageBySequenceNumber returns a single message given the message's sequence number
func (m *FilesystemMailbox) MessageBySequenceNumber(seqno uint32) Message {
	if seqno > uint32(len(m.messages)) {
		return nil
	}
	return m.messages[seqno-1]
}

// MessageByUID returns a single message given the message's sequence number
func (m *FilesystemMailbox) MessageByUID(uidno uint32) Message {
	if uidno > uint32(len(m.messages)) {
		return nil
	}
	return m.messages[uidno-1]
}

// MessageSetByUID returns a slice of messages given a set of UID ranges.
// eg 1,5,9,28:140,190:*
func (m *FilesystemMailbox) MessageSetByUID(set types.SequenceSet) []Message {
	var msgs []Message

	// If the mailbox is empty, return empty array
	if m.Messages() == 0 {
		return msgs
	}

	for _, msgRange := range set {
		// If Min is "*", meaning the last UID in the mailbox, Max should
		// always be Nil
		if msgRange.Min.Last() {
			// Return the last message in the mailbox
			msgs = append(msgs, m.MessageByUID(m.LastUID()))
			continue
		}

		start, err := msgRange.Min.Value()
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			return msgs
		}

		// If no Max is specified, the sequence number must be either a fixed
		// sequence number or
		if msgRange.Max.Nil() {
			var uid uint32
			// Fetch specific message by sequence number
			uid, err = msgRange.Min.Value()
			msg := m.MessageByUID(uid)
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				return msgs
			}
			if msg != nil {
				msgs = append(msgs, msg)
			}
			continue
		}

		var end uint32
		if msgRange.Max.Last() {
			end = m.LastUID()
		} else {
			end, err = msgRange.Max.Value()
		}

		// Note this is very inefficient when
		// the message array is large. A proper
		// storage system using eg SQL might
		// instead perform a query here using
		// the range values instead.
		for _, msg := range m.messages {
			uid := msg.UID()
			if uid >= start && uid <= end {
				msgs = append(msgs, msg)
			}
		}
		for index := uint32(start); index <= end; index++ {
		}
	}

	return msgs
}

// MessageSetBySequenceNumber returns a slice of messages given a set of
// sequence number ranges
func (m *FilesystemMailbox) MessageSetBySequenceNumber(set types.SequenceSet) []Message {
	var msgs []Message

	// If the mailbox is empty, return empty array
	if m.Messages() == 0 {
		return msgs
	}

	// For each sequence range in the sequence set
	for _, msgRange := range set {
		// If Min is "*", meaning the last message in the mailbox, Max should
		// always be Nil
		if msgRange.Min.Last() {
			// Return the last message in the mailbox
			msgs = append(msgs, m.MessageBySequenceNumber(m.Messages()))
			continue
		}

		start, err := msgRange.Min.Value()
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			return msgs
		}

		// If no Max is specified, the sequence number must be either a fixed
		// sequence number or
		if msgRange.Max.Nil() {
			var sequenceNo uint32
			// Fetch specific message by sequence number
			sequenceNo, err = msgRange.Min.Value()
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				return msgs
			}
			msg := m.MessageBySequenceNumber(sequenceNo)
			if msg != nil {
				msgs = append(msgs, msg)
			}
			continue
		}

		var end uint32
		if msgRange.Max.Last() {
			end = uint32(len(m.messages))
		} else {
			end, err = msgRange.Max.Value()
		}

		// Note this is very inefficient when
		// the message array is large. A proper
		// storage system using eg SQL might
		// instead perform a query here using
		// the range values instead.
		for seqNo := start; seqNo <= end; seqNo++ {
			msgs = append(msgs, m.MessageBySequenceNumber(seqNo))
		}
	}
	return msgs

}

// NewMessage creates a new message in the mailbox.
// NOTE: This should not make any changes to the mailbox until the
// message's `Save` method is called.
func (m *FilesystemMailbox) NewMessage() Message {
	return &FilesystemMessage{}
}

// DeleteFlaggedMessages deletes messages marked with the Delete flag and
// returns them. None in this read only imap server.
func (m *FilesystemMailbox) DeleteFlaggedMessages() ([]Message, error) {
	var delMsgs []Message
	return delMsgs, nil
}

// Header returns the message's MIME Header.
func (m *FilesystemMessage) Header() (hdr mail.Header) {
	file, err := os.Open(m.path)
	if err != nil {
		return nil
	}
	defer file.Close()
	email, err := mail.ReadMessage(file)
	if err != nil {
		return nil
	}
	return email.Header
}

// Body returns the full body of the message
func (m *FilesystemMessage) Body() string {
	file, err := os.Open(m.path)
	if err != nil {
		return ""
	}
	defer file.Close()
	email, err := mail.ReadMessage(file)
	if err != nil {
		return ""
	}
	body, err := ioutil.ReadAll(email.Body)
	if err != nil {
		return ""
	}
	return string(body)
}

// UID returns the message's unique identifier (UID).
func (m *FilesystemMessage) UID() uint32 { return m.ID + 1 }

// SequenceNumber returns the message's sequence number.
func (m *FilesystemMessage) SequenceNumber() uint32 { return m.ID + 1 }

// Size returns the message's full RFC822 size,
// including full message header and body.
func (m *FilesystemMessage) Size() uint32 {
	return uint32((*m.info).Size())
}

// InternalDate returns the internally stored date of the message
func (m *FilesystemMessage) InternalDate() time.Time {
	return (*m.info).ModTime()
}

// Keywords returns any keywords associated with the message
func (m *FilesystemMessage) Keywords() []string {
	var f []string
	return f
}

// Flags returns any flags on the message.
func (m *FilesystemMessage) Flags() types.Flags {
	return m.flags
}

// OverwriteFlags replaces any flags on the message with those specified.
func (m *FilesystemMessage) OverwriteFlags(newFlags types.Flags) Message {
	//m.flags = newFlags
	return m
}

// AddFlags adds the given flag to the message.
func (m *FilesystemMessage) AddFlags(newFlags types.Flags) Message {
	//m.flags = m.flags.SetFlags(newFlags)
	return m
}

// RemoveFlags removes the given flag from the message.
func (m *FilesystemMessage) RemoveFlags(newFlags types.Flags) Message {
	//m.flags = m.flags.ResetFlags(newFlags)
	return m
}

// SetHeaders sets the e-mail headers of the message.
func (m *FilesystemMessage) SetHeaders(newHeader mail.Header) Message {
	return m
}

// SetBody sets the body of the message.
func (m *FilesystemMessage) SetBody(newBody string) Message {
	return m
}

// Save saves the message to the mailbox it belongs to.
func (m *FilesystemMessage) Save() (Message, error) {
	return m, errors.New("This mail store is read-only")
}
