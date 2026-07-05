package markdown

import (
	"bytes"
	"html"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldhtml "github.com/yuin/goldmark/renderer/html"
)

var (
	mentionRe      = regexp.MustCompile(`@([a-zA-Z0-9_]{3,30})`)
	urlRe          = regexp.MustCompile(`https?://[^\s<>"]+|www\.[^\s<>"]+`)
	httpPrefixRe   = regexp.MustCompile(`^https?://`)
	multipleNewline = regexp.MustCompile(`\n{3,}`)

	md   goldmark.Markdown
	policy *bluemonday.Policy
)

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.Linkify,
			extension.Strikethrough,
			extension.Table,
			extension.TaskList,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			goldhtml.WithHardWraps(),
			goldhtml.WithUnsafe(),
		),
	)

	policy = bluemonday.NewPolicy()

	policy.AllowStandardURLs()

	policy.AllowElements("p", "br")

	policy.AllowElements("strong", "b")
	policy.AllowElements("em", "i")
	policy.AllowElements("s", "del", "strike")
	policy.AllowElements("code", "pre")
	policy.AllowElements("ul", "ol", "li")
	policy.AllowElements("blockquote")
	policy.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")
	policy.AllowElements("hr")
	policy.AllowElements("table", "thead", "tbody", "tr", "th", "td")

	policy.AllowAttrs("href").OnElements("a")
	policy.AllowAttrs("src", "alt", "title").OnElements("img")
	policy.AllowAttrs("class").OnElements("span", "code", "pre")

	policy.AllowURLSchemes("mailto", "http", "https")
	policy.AllowRelativeURLs(true)

	policy.AddTargetBlankToFullyQualifiedLinks(true)
	policy.RequireNoFollowOnFullyQualifiedLinks(true)
}

type MentionUser struct {
	UserID   string
	Username string
}

func SanitizeInput(raw string) string {
	s := strings.TrimSpace(raw)
	s = multipleNewline.ReplaceAllString(s, "\n\n")

	s = html.EscapeString(s)

	return s
}

func RenderToHTML(raw string, mentions []MentionUser) string {
	mentionMap := make(map[string]string)
	for _, m := range mentions {
		mentionMap[m.Username] = m.UserID
	}

	processed := mentionRe.ReplaceAllStringFunc(raw, func(match string) string {
		username := match[1:]
		if _, ok := mentionMap[username]; ok {
			return match
		}
		return match
	})

	var buf bytes.Buffer
	if err := md.Convert([]byte(processed), &buf); err != nil {
		return "<p>" + html.EscapeString(raw) + "</p>"
	}

	htmlRaw := buf.String()

	htmlRaw = mentionRe.ReplaceAllStringFunc(htmlRaw, func(match string) string {
		username := match[1:]
		if uid, ok := mentionMap[username]; ok {
			return `<span class="mention" data-user-id="` + uid + `">` + match + `</span>`
		}
		return match
	})

	htmlRaw = urlRe.ReplaceAllStringFunc(htmlRaw, func(match string) string {
		href := match
		if !httpPrefixRe.MatchString(match) {
			href = "https://" + match
		}
		return `<a href="` + href + `">` + match + `</a>`
	})

	safe := policy.Sanitize(htmlRaw)

	return safe
}
