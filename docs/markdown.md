# Markdown Support in deck

`deck` supports a comprehensive set of Markdown features for creating presentations in Google Slides.

## CommonMark Support

`deck` almost fully supports the [CommonMark specification](https://spec.commonmark.org/) with the following clarifications and limitations:

### Supported CommonMark Features

- **Headings** (H1-H6):
  - ATX headings: `#`, `##`, `###`, etc.
  - Setext headings: underlined with `===` (H1) or `---` (H2)
- **Paragraphs**: Regular text blocks
- **Emphasis**: `*em*` or `_em_`
- **Strong emphasis**: `**strong**` or `__strong__`
- **Lists**: Unordered (`-`, `*`, `+`) and ordered (`1.`, `1)`)
- **Links**: `[text](url)` and reference-style links
- **Images**: `![alt text](url)`
- **Inline code**: `` `code` ``
- **Code blocks**:
  - Fenced code blocks with ` ``` ` or `~~~`
  - Indented code blocks (4 spaces or 1 tab)
- **Block quotes**: `> quoted text`
- **Line breaks**: Two spaces at end of line or `<br>` tag
- **Autolinks**: `<https://example.com>` and `<user@example.com>`
- **Escape sequences**: Backslash escaping of special characters

### Raw HTML Support

#### Inline HTML Elements (Supported)
`deck` supports raw inline HTML elements for text-level semantics and edits:

- **[Text-level semantics elements](https://html.spec.whatwg.org/multipage/text-level-semantics.html):**
    - `<a>`, `<abbr>`, `<b>`, `<cite>`, `<code>`, `<data>`, `<dfn>`, `<em>`, `<i>`, `<kbd>`, `<mark>`, `<q>`, `<rp>`, `<rt>`, `<ruby>`, `<s>`, `<samp>`, `<small>`, `<span>`, `<strong>`, `<sub>`, `<sup>`, `<time>`, `<u>`, `<var>`
- **[Edits elements](https://html.spec.whatwg.org/multipage/edits.html):**
    - `<ins>`, `<del>`
- **Not supported text-level semantics:**
    - `<wbr>`, `<bdi>`, `<bdo>` - These text-direction and line-breaking hints are not supported

#### Block HTML Elements (Not Supported)
Raw HTML block elements are **not supported** and there are no plans to support them:
- Examples: `<div>`, `<section>`, `<article>`, `<header>`, `<footer>`, `<nav>`, `<aside>`, `<table>`, etc.
- **Behavior**: When block HTML elements are encountered, they are treated as plain text and will appear as literal tag strings in the output (no escaping is performed)

### Character References (Not Supported)
HTML character references (`&#60;`, `&lt;`, etc.) are not supported (and won't be in the future) and appear as literal text. Use Unicode characters directly or backslash escaping instead.

## GitHub Flavored Markdown (GFM) Extensions
`deck` selectively supports some [GFM (GitHub Flavored Markdown)](https://github.github.com/gfm/) extensions that are useful for presentations:

### Supported GFM Features

#### Tables
```markdown
| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
```
- Table headers are automatically styled with bold text and a gray background
- Cell content supports inline formatting (bold, italic, code, links, etc.)

#### Strikethrough
```markdown
~~strikethrough text~~
```
- Renders with strikethrough formatting
- Maps to the `<del>` HTML element internally (as specified in the [GFM specification](https://github.github.com/gfm/#strikethrough-extension-))

### Unsupported GFM Features

The following GFM extensions are **not supported** as they are not relevant for presentations:

- **Task lists**: Checkbox lists are not needed in presentations
- **Autolinks without brackets**: Use Markdown autolink syntax by wrapping URLs in angle brackets (`<URL>`) instead

## Horizontal Rules and Page Breaks
Among all Markdown horizontal rule (thematic break) syntaxes, `deck` treats them differently:

### Page Separators (Slide Breaks)

**Only** three or more consecutive hyphens from the beginning of the line create slide breaks:
- `---`, `----`, `-----` etc. (no spaces, no indentation)

```markdown
# Slide 1
---
# Slide 2
```

**Exceptions that do NOT create slide breaks:**
- YAML frontmatter delimiter
- Files starting with `---` without frontmatter (simply ignored/removed)
- Setext H2 headings (`text` underlined with `---`)
- Hyphens inside code blocks

### Content Separators Within Slides

All other horizontal rule syntaxes remain as visual separators within slides:
- `- - -`, `***`, `___` (with or without spaces)
- Indented horizontal rules (`   ---`)

These separate multiple body placeholders within a single slide.

## Line Break Handling

### Default Behavior (CommonMark/GFM Compliant)

By default, `deck` follows the CommonMark and GitHub Flavored Markdown specifications:
- **Soft line breaks** (single line breaks) are rendered as **spaces**
- **Hard line breaks** require either:
  - Two or more spaces at the end of a line
  - Using the HTML `<br>` tag

```markdown
This is line one
This is line two (rendered as: "This is line one This is line two")

This is line one  
This is line two (rendered with actual line break)

This is line one<br>
This is line two (rendered with actual line break)
```

### Alternative: `breaks: true` Setting

You can enable GitHub-style rendering where all line breaks are preserved:

```yaml
---
breaks: true
---

This will
render with
actual line breaks
```

When `breaks: true` is set, soft line breaks in the markdown source are preserved as line breaks in the rendered slides, similar to how GitHub renders markdown on their website.

## Special Considerations for Presentations

### Heading Hierarchy in Slides

Within each slide:
- The **shallowest heading level** becomes the slide title
- The **next heading level** (minimum + 1) becomes the subtitle
- All other content goes into the body placeholders

Example with ATX headings:
```markdown
# Title (→ title placeholder)
## Subtitle (→ subtitle placeholder)

Body content (→ body placeholder)

### Sub-heading (→ body placeholder)
```

Example with Setext headings:
```markdown
Title
=====
(→ title placeholder)

Subtitle
--------
(→ subtitle placeholder)

Body content (→ body placeholder)
```

### Placeholder Insertion Order

Content is inserted into placeholders in the order it appears in the markdown, filling placeholders from top to bottom (or left to right for same-height placeholders). If there are insufficient placeholders, remaining content will not be rendered.
