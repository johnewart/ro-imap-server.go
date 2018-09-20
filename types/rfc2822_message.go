package types

import (
  "bytes"
  "errors"
  "io/ioutil"
	"net/mail"
)

// RFC2822Message is a message compliant with RFC 2822.
type RFC2822Message struct {
	Headers mail.Header
	Body    string
}

// MessageFromBytes creates a RFC2822Message from its byte representation.
func MessageFromBytes(msgBytes []byte) (msg RFC2822Message, err error) {
  r := bytes.NewReader(msgBytes)
  email, err := mail.ReadMessage(r)
  if err != nil { return RFC2822Message{}, errors.New("Bad RFC2822 message") }
  msg.Headers = email.Header
  body, err := ioutil.ReadAll(email.Body)
  if err != nil { return RFC2822Message{}, errors.New("Bad RFC2822 body") }
	msg.Body = string(body)
  return msg, nil
}
