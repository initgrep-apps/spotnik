package cliout

import (
	"bytes"
	"io"
)

// Builder provides a fluent façade over []Message. All chain methods append to
// the same slice; WriteTo renders and flushes.
type Builder struct {
	msgs []Message
}

// New returns an empty Builder.
func New() *Builder { return &Builder{} }

// Header appends a Header message.
func (b *Builder) Header(s Status, subject, state string) *Builder {
	b.msgs = append(b.msgs, Header{Status: s, Subject: subject, State: state})
	return b
}

// Step appends a Step message.
func (b *Builder) Step(s Status, text string) *Builder {
	b.msgs = append(b.msgs, Step{Status: s, Text: text})
	return b
}

// KV appends a KV message with the given pairs.
func (b *Builder) KV(pairs ...KVPair) *Builder {
	b.msgs = append(b.msgs, KV{Pairs: pairs})
	return b
}

// Steps appends a Steps message with the given instruction items.
func (b *Builder) Steps(items ...string) *Builder {
	b.msgs = append(b.msgs, Steps{Items: items})
	return b
}

// Hint appends a Hint message.
func (b *Builder) Hint(verb, cmd, tail string) *Builder {
	b.msgs = append(b.msgs, Hint{Verb: verb, Cmd: cmd, Tail: tail})
	return b
}

// URL appends a URL message.
func (b *Builder) URL(label, href string) *Builder {
	b.msgs = append(b.msgs, URL{Label: label, Href: href})
	return b
}

// Paragraph appends a plain Paragraph message.
func (b *Builder) Paragraph(text string) *Builder {
	b.msgs = append(b.msgs, Paragraph{Text: text})
	return b
}

// Dim appends a dimmed Paragraph (Dim: true).
func (b *Builder) Dim(text string) *Builder {
	b.msgs = append(b.msgs, Paragraph{Text: text, Dim: true})
	return b
}

// Messages returns the accumulated message slice for test assertions.
func (b *Builder) Messages() []Message { return b.msgs }

// WriteTo renders and flushes to w using Write. Implements io.WriterTo.
func (b *Builder) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	Write(&buf, b.msgs...)
	n, err := w.Write(buf.Bytes())
	return int64(n), err
}

// Pair is a shorthand constructor for KVPair with no caption.
func Pair(label, value string) KVPair { return KVPair{Label: label, Value: value} }

// PairWithCaption constructs a KVPair with a trailing muted caption.
func PairWithCaption(label, value, caption string) KVPair {
	return KVPair{Label: label, Value: value, Caption: caption}
}
