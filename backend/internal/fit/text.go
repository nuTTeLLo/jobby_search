package fit

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Elements whose text is never part of a posting.
var skippedElements = map[string]bool{
	"script": true, "style": true, "noscript": true, "template": true, "svg": true,
	"head": true, "nav": true, "footer": true, "form": true, "button": true, "iframe": true,
}

// Elements that start a new line in the extracted text.
var blockElements = map[string]bool{
	"p": true, "div": true, "br": true, "li": true, "ul": true, "ol": true, "tr": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"section": true, "article": true, "header": true, "main": true, "table": true,
}

var (
	spaceRun     = regexp.MustCompile(`[ \t\x{00a0}]+`)
	blankLineRun = regexp.MustCompile(`\n{3,}`)
)

// textOf returns the readable text under a node, one block per line, with list
// items marked. Good enough for a model to read; not meant to round-trip.
func textOf(node *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && skippedElements[n.Data] {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		block := n.Type == html.ElementNode && blockElements[n.Data]
		if block {
			b.WriteString("\n")
			if n.Data == "li" {
				b.WriteString("- ")
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if block {
			b.WriteString("\n")
		}
	}
	walk(node)

	lines := strings.Split(b.String(), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(spaceRun.ReplaceAllString(line, " "))
	}
	text := strings.Join(lines, "\n")
	return strings.TrimSpace(blankLineRun.ReplaceAllString(text, "\n\n"))
}

// htmlToText converts an HTML fragment, such as a board's description markup.
func htmlToText(fragment string) string {
	doc, err := html.Parse(strings.NewReader(fragment))
	if err != nil {
		return ""
	}
	return textOf(doc)
}

// findByClass returns the first element under n whose class list contains class.
func findByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		for _, attr := range n.Attr {
			if attr.Key == "class" && hasClass(attr.Val, class) {
				return n
			}
		}
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findByClass(child, class); found != nil {
			return found
		}
	}
	return nil
}

func findElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func hasClass(classes, class string) bool {
	for _, c := range strings.Fields(classes) {
		if c == class {
			return true
		}
	}
	return false
}

// textByClass is the flattened text of the first element with the class, on one
// line, or "" if there is none.
func textByClass(doc *html.Node, class string) string {
	node := findByClass(doc, class)
	if node == nil {
		return ""
	}
	return strings.Join(strings.Fields(textOf(node)), " ")
}
