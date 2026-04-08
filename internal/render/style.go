package render

import (
	glamouransi "github.com/charmbracelet/glamour/ansi"
	glamourstyles "github.com/charmbracelet/glamour/styles"
)

func readerStyleConfig() glamouransi.StyleConfig {
	style := glamourstyles.DarkStyleConfig

	style.Document.Margin = uintPtr(0)
	style.Document.BlockPrefix = ""
	style.Document.BlockSuffix = ""

	style.BlockQuote.IndentToken = stringPtr("▍ ")
	style.BlockQuote.Color = stringPtr("#94a3b8")

	style.Heading.BlockPrefix = "\n"
	style.Heading.BlockSuffix = "\n"
	style.Heading.Color = stringPtr("#7dd3fc")
	style.Heading.Bold = boolPtr(true)

	style.H1.Prefix = " "
	style.H1.Suffix = " "
	style.H1.Color = stringPtr("#0f172a")
	style.H1.BackgroundColor = stringPtr("#facc15")
	style.H1.Bold = boolPtr(true)

	style.H2.Prefix = "## "
	style.H2.Color = stringPtr("#a7f3d0")
	style.H2.Bold = boolPtr(true)

	style.H3.Prefix = "### "
	style.H3.Color = stringPtr("#bfdbfe")
	style.H3.Bold = boolPtr(true)

	style.H4.Prefix = "#### "
	style.H4.Color = stringPtr("#cbd5e1")

	style.H5.Prefix = "##### "
	style.H5.Color = stringPtr("#cbd5e1")

	style.H6.Prefix = "###### "
	style.H6.Color = stringPtr("#94a3b8")
	style.H6.Bold = boolPtr(false)

	style.HorizontalRule.Format = "\n────────────────────\n"
	style.HorizontalRule.Color = stringPtr("#475569")

	style.Link.Color = stringPtr("#93c5fd")
	style.Link.Underline = boolPtr(true)
	style.LinkText.Color = stringPtr("#a7f3d0")
	style.LinkText.Bold = boolPtr(true)

	style.Code.Prefix = " "
	style.Code.Suffix = " "
	style.Code.Color = stringPtr("#fda4af")
	style.Code.BackgroundColor = stringPtr("#1f2937")

	style.CodeBlock.Margin = uintPtr(1)
	style.CodeBlock.Color = stringPtr("#cbd5e1")
	if style.CodeBlock.Chroma != nil {
		style.CodeBlock.Chroma.Background.BackgroundColor = stringPtr("#111827")
		style.CodeBlock.Chroma.Comment.Color = stringPtr("#6b7280")
		style.CodeBlock.Chroma.Keyword.Color = stringPtr("#7dd3fc")
		style.CodeBlock.Chroma.NameFunction.Color = stringPtr("#86efac")
		style.CodeBlock.Chroma.LiteralString.Color = stringPtr("#f9a8d4")
		style.CodeBlock.Chroma.LiteralNumber.Color = stringPtr("#fcd34d")
	}

	style.Table.Margin = uintPtr(1)
	style.Table.Color = stringPtr("#cbd5e1")
	style.Table.CenterSeparator = stringPtr("┼")
	style.Table.ColumnSeparator = stringPtr("│")
	style.Table.RowSeparator = stringPtr("─")

	return style
}

func stringPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func uintPtr(value uint) *uint {
	return &value
}
