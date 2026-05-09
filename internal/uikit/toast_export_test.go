package uikit

// RenderToastMessageForTest exposes renderToastMessage to black-box tests.
func RenderToastMessageForTest(glyph, title string, bodyLines ...string) string {
	body := ""
	if len(bodyLines) > 0 {
		body = bodyLines[0]
	}
	return renderToastMessage(glyph, title, body)
}
