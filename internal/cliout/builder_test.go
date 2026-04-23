package cliout

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_Messages_accumulatesInOrder(t *testing.T) {
	b := New().
		Header(Active, "Spotnik", "authenticated").
		KV(Pair("Client ID", "present")).
		Hint("Run", "spotnik auth login", "to reconnect")

	got := b.Messages()
	require.Len(t, got, 3)
	assert.IsType(t, Header{}, got[0])
	assert.IsType(t, KV{}, got[1])
	assert.IsType(t, Hint{}, got[2])

	hdr := got[0].(Header)
	assert.Equal(t, Active, hdr.Status)
	assert.Equal(t, "Spotnik", hdr.Subject)
	assert.Equal(t, "authenticated", hdr.State)
}

func TestBuilder_WriteTo_rendersAndFlushes(t *testing.T) {
	var buf bytes.Buffer
	n, err := New().
		Header(Active, "Spotnik", "authenticated").
		WriteTo(&buf)
	require.NoError(t, err)
	assert.Greater(t, n, int64(0), "WriteTo must return byte count > 0")
	assert.Contains(t, buf.String(), "Spotnik")
	assert.Contains(t, buf.String(), "authenticated")
}

func TestPair_helper(t *testing.T) {
	p := Pair("Label", "Value")
	assert.Equal(t, KVPair{Label: "Label", Value: "Value"}, p)
}

func TestPairWithCaption_helper(t *testing.T) {
	p := PairWithCaption("Label", "Value", "caption")
	assert.Equal(t, KVPair{Label: "Label", Value: "Value", Caption: "caption"}, p)
}

func TestBuilder_allChainMethods(t *testing.T) {
	b := New().
		Header(Active, "H", "s").
		Step(StatusSuccess, "step text").
		KV(Pair("k", "v")).
		Steps("one", "two").
		Hint("Run", "cmd", "tail").
		URL("label", "https://example.com").
		Paragraph("plain").
		Dim("muted text")

	msgs := b.Messages()
	require.Len(t, msgs, 8)
	assert.IsType(t, Header{}, msgs[0])
	assert.IsType(t, Step{}, msgs[1])
	assert.IsType(t, KV{}, msgs[2])
	assert.IsType(t, Steps{}, msgs[3])
	assert.IsType(t, Hint{}, msgs[4])
	assert.IsType(t, URL{}, msgs[5])
	assert.IsType(t, Paragraph{}, msgs[6])
	dimMsg := msgs[7].(Paragraph)
	assert.True(t, dimMsg.Dim)
	assert.Equal(t, "muted text", dimMsg.Text)
}
