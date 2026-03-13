package exporter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// ExportPDF converts markdown content to a PDF file at the given output path.
func ExportPDF(ctx context.Context, markdownContent, outputPath string) error {
	// Convert markdown to basic HTML
	html := markdownToHTML(markdownContent)

	// Write HTML to a temp file
	tmpFile, err := os.CreateTemp("", "brag-report-*.html")
	if err != nil {
		return fmt.Errorf("creating temp HTML file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(html); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing HTML: %w", err)
	}
	tmpFile.Close()

	absPath, err := filepath.Abs(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("getting absolute path: %w", err)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var pdfBuf []byte
	if err := chromedp.Run(taskCtx,
		chromedp.Navigate("file://"+absPath),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithMarginTop(0.5).
				WithMarginBottom(0.5).
				WithMarginLeft(0.5).
				WithMarginRight(0.5).
				Do(ctx)
			return err
		}),
	); err != nil {
		return fmt.Errorf("generating PDF via chromedp: %w", err)
	}

	if err := os.WriteFile(outputPath, pdfBuf, 0644); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}
	return nil
}

// markdownToHTML does a minimal conversion of markdown to HTML for PDF rendering.
func markdownToHTML(md string) string {
	lines := strings.Split(md, "\n")
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html><head>
<meta charset="UTF-8">
<style>
body { font-family: Georgia, serif; max-width: 800px; margin: 40px auto; color: #222; line-height: 1.6; }
h1 { font-size: 2em; border-bottom: 2px solid #333; padding-bottom: 0.3em; }
h2 { font-size: 1.5em; border-bottom: 1px solid #ccc; margin-top: 2em; }
h3 { font-size: 1.2em; margin-top: 1.5em; color: #444; }
em { color: #666; }
strong { color: #111; }
a { color: #0366d6; }
hr { border: none; border-top: 1px solid #eee; margin: 2em 0; }
p { margin: 0.5em 0; }
</style>
</head><body>`)

	inPara := false
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "# "):
			if inPara { sb.WriteString("</p>"); inPara = false }
			sb.WriteString("<h1>" + escapeHTML(line[2:]) + "</h1>\n")
		case strings.HasPrefix(line, "## "):
			if inPara { sb.WriteString("</p>"); inPara = false }
			sb.WriteString("<h2>" + escapeHTML(line[3:]) + "</h2>\n")
		case strings.HasPrefix(line, "### "):
			if inPara { sb.WriteString("</p>"); inPara = false }
			sb.WriteString("<h3>" + escapeHTML(line[4:]) + "</h3>\n")
		case strings.TrimSpace(line) == "---":
			if inPara { sb.WriteString("</p>"); inPara = false }
			sb.WriteString("<hr>\n")
		case strings.TrimSpace(line) == "":
			if inPara { sb.WriteString("</p>"); inPara = false }
		default:
			rendered := renderInline(line)
			if !inPara {
				sb.WriteString("<p>")
				inPara = true
			} else {
				sb.WriteString("<br>")
			}
			sb.WriteString(rendered)
		}
	}
	if inPara {
		sb.WriteString("</p>")
	}
	sb.WriteString("</body></html>")
	return sb.String()
}

func renderInline(s string) string {
	// Bold: **text**
	result := ""
	for len(s) > 0 {
		if idx := strings.Index(s, "**"); idx >= 0 {
			result += escapeHTML(s[:idx])
			s = s[idx+2:]
			if end := strings.Index(s, "**"); end >= 0 {
				result += "<strong>" + escapeHTML(s[:end]) + "</strong>"
				s = s[end+2:]
			}
		} else if idx := strings.Index(s, "_"); idx >= 0 {
			result += escapeHTML(s[:idx])
			s = s[idx+1:]
			if end := strings.Index(s, "_"); end >= 0 {
				result += "<em>" + escapeHTML(s[:end]) + "</em>"
				s = s[end+1:]
			}
		} else if strings.Contains(s, "[") {
			// simple link [text](url)
			lBracket := strings.Index(s, "[")
			rBracket := strings.Index(s, "](")
			if rBracket > lBracket {
				urlEnd := strings.Index(s[rBracket+2:], ")")
				if urlEnd >= 0 {
					result += escapeHTML(s[:lBracket])
					text := s[lBracket+1 : rBracket]
					href := s[rBracket+2 : rBracket+2+urlEnd]
					result += fmt.Sprintf(`<a href="%s">%s</a>`, href, escapeHTML(text))
					s = s[rBracket+2+urlEnd+1:]
					continue
				}
			}
			result += escapeHTML(s)
			break
		} else {
			result += escapeHTML(s)
			break
		}
	}
	return result
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
