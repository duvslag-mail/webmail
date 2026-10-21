package mail

import "time"

type Address struct {
	Name  string
	Email string
}

type MessageSummary struct {
	UID     uint32 // stable id that we can use in links
	From    []Address
	Subject string
	Date    time.Time
}

type Message struct {
	MessageSummary
	To, Cc      []Address
	TextBody    string
	HTMLBody    string
	Attachments []Attachment
}

type Attachment struct {
	Filename string
	MIMEType string
	Size     uint32
	PartPath []int
}
