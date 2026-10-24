package mail

import "github.com/microcosm-cc/bluemonday"

const (
	emailCSP           = `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data: cid:; style-src 'unsafe-inline'; font-src data:; form-action 'none'; base-uri 'none'">`
	emailCSPWithImages = `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src https: data: cid:; style-src 'unsafe-inline'; font-src data:; form-action 'none'; base-uri 'none'">`

	// don't tell image hosts which page the request came from
	noReferrer = `<meta name="referrer" content="no-referrer">`
)

var policy = emailPolicy()

func emailPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()

	// old email things
	p.AllowAttrs("align", "valign", "width", "height", "bgcolor", "border",
		"cellpadding", "cellspacing").OnElements("table", "tr", "td", "th", "img", "div", "p")

	// inline styles or everything will break
	p.AllowAttrs("style").Globally()
	p.AllowStyles("color", "background-color", "font-family", "font-size", "font-weight",
		"font-style", "text-align", "text-decoration", "line-height", "padding", "margin",
		"border", "border-radius", "width", "max-width", "height", "display",
		"vertical-align").Globally()

	// links open in a new tab, with noopener added automatically
	p.AddTargetBlankToFullyQualifiedLinks(true)

	return p
}

func SanitizeHTML(htmlBody string, allowImages bool) string {
	// bluemonday step
	sanitized := policy.Sanitize(htmlBody)

	// add the csp, there's always one, allowImages only opens img-src to https
	csp := emailCSP
	if allowImages {
		csp = emailCSPWithImages
	}

	return csp + noReferrer + sanitized
}
