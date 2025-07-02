<p align="center">
<img src="https://github.com/k1LoW/deck/raw/main/img/logo.svg" width="200" alt="deck">
</p>

# deck

[![build](https://github.com/k1LoW/deck/actions/workflows/ci.yml/badge.svg)](https://github.com/k1LoW/deck/actions/workflows/ci.yml) ![Coverage](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/deck/coverage.svg) ![Code to Test Ratio](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/deck/ratio.svg) [![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/k1LoW/deck)

`deck` is a tool for creating deck using Markdown and Google Slides.

## Key concept

- **Continuous deck building**
    - Generate and modify deck iteratively.
- **Separate content and design**
    - Markdown for content, Google Slides for design.

## Usage

### Setup

#### Get and set your OAuth client credentials

- Create (or reuse) a developer project at https://console.cloud.google.com.
- Enable `Google Slides API` and `Google Drive API` at [`API & Services` page](https://console.cloud.google.com/apis/dashboard).
- Go to `Credentials` page and click `+ CREATE CREDENTIALS` at the top.
- Create `OAuth client ID` type of credentials.
- Choose type `Desktop app`.
- Download credentials file to `~/.local/share/deck/credentials.json` ( or `${XDG_DATA_HOME}/deck/credentials.json` ).

### Get presentation ID

#### When using an existing presentation

Get the presentation ID you want to operate. You can get a list with `deck ls`.

For example, presentation ID is `xxxxxXXXXxxxxxXXXXxxxxxxxxxx` of https://docs.google.com/presentation/d/xxxxxXXXXxxxxxXXXXxxxxxxxxxx/edit .

#### When creating a new presentation

You can create a new presentation with the deck new command and obtain the presentation ID.

If you want to apply a theme, specify the presentation ID of a presentation that is already using that theme with the `--from` option.

```console
$ deck new --from yyyyyyyYYYYyYYYYYYYyyyyyyyyy --title "Talk about deck"
xxxxxXXXXxxxxxXXXXxxxxxxxxxx
```

### Write desk in markdown

The slide pages are represented by dividing them with horizontal lines `---`.

> [!NOTE]
> The `---` at the beginning of the markdown is ignored.

### Apply desk written in markdown to Google Slides presentation

```console
$ deck apply deck.md --presentation-id xxxxxXXXXxxxxxXXXXxxxxxxxxxx
```

If your markdown file includes `presentationID` in the frontmatter, you can use the simplified syntax:

```console
$ deck apply deck.md
```

#### Watch mode

You can use the `--watch` flag to continuously monitor changes to your markdown file and automatically apply them to the presentation:

```console
$ deck apply --watch deck.md --presentation-id xxxxxXXXXxxxxxXXXXxxxxxxxxxx
```

Or with frontmatter:

```console
$ deck apply --watch deck.md
```

This is useful during the content creation process as it allows you to see your changes reflected in the presentation in real-time as you edit the markdown file.

> [!NOTE]
> The `--watch` flag cannot be used together with the `--page` flag.

## Support markdown rules

### YAML Frontmatter

`deck` accepts YAML frontmatter at the beginning of your markdown file.

```markdown
---
presentationID: xxxxxXXXXxxxxxXXXXxxxxxxxxxx
title: Talk about deck
---

# First Slide

Content...
```

The frontmatter must be:
- At the very beginning of the file
- Enclosed between `---` delimiters
- Valid YAML syntax
- Use `camelCase` for fields used in `deck` settings

#### Available fields

- `presentationID`: Google Slides presentation ID. When specified, you can use the simplified command syntax.
- `title`: title of the presentation. When specified, you can use the simplified command syntax.
- `layout`: Automatic layout assignment based on heading levels.

#### Automatic layout assignment

You can specify layouts for different heading levels in the frontmatter. This automatically applies the appropriate layout to each slide based on its title heading level:

```markdown
---
presentationID: xxxxxXXXXxxxxxXXXXxxxxxxxxxx
title: My Presentation
layout:
  title: title-slide    # Layout for the first slide
  h1: part              # Layout for H1 headings
  h2: chapter           # Layout for H2 headings
  h3: section           # Layout for H3 headings
---

# Part 1: Introduction
Content for part slide...

---

## Chapter 1: Overview
Content for chapter slide...

---

### Section 1.1: Details
Content for section slide...
```

- The first slide uses the layout specified by `title`
- Subsequent slides use layouts based on their heading level (`h1`, `h2`, `h3`, `h4`, `h5`, `h6`)
- Explicit layout specifications via JSON comments (`<!-- {"layout": "custom"} -->`) take precedence over frontmatter settings

### Insertion rule

`deck` inserts values according to the following rules regardless of the slide layout.

- The minimum heading level within each slide content is treated as the title and inserted into the title placeholder ( `CENTERED_TITLE` or `TITLE` ) in order.
  - In most cases, this will be H1 (`#`), which is the standard for slide titles
- The next heading level (minimum level + 1) is treated as the subtitle and inserted into the subtitle placeholder ( `SUBTITLE` ) in order.
  - When H1 is used for titles, H2 (`##`) becomes the subtitle
- All other items are inserted into the body placeholder ( `BODY` ) in order.

For example:
- **Standard case**: If a slide contains `#` (H1), then `#` becomes title and `##` becomes subtitle
- **Alternative case**: If a slide only contains `##` (H2) or higher, then `##` becomes title and `###` becomes subtitle
- This allows flexibility in creating slides from various markdown content structures while maintaining familiar heading hierarchies

> [!NOTE]
> They are inserted in the order they appear in the markdown document, **from the placeholder at the top of the slide** (or from the placeholder on the left if the slides are the same height).

#### Input markdown document

```markdown
# CAP theorem

## In Database theory

## Consistency

Every read receives the most recent write or an error.

## Availability

Every request received by a non-failing node in the system must result in a response.

## Partition tolerance

The system continues to operate despite an arbitrary number of messages being dropped (or delayed) by the network between nodes.
```

#### Layout and placeholders

![img](img/layout.png)

#### Result of applying

![img](img/result.png)

### Support syntax in body

- Bold ( `**bold**` )
- Italic ( `*italic*` `__italic__` )
- List ( `-` `*` )
- Ordered list ( `1.` `1)` )
- Link ( `[Link](https://example.com)` )
- Angle bracket autolinks ( `<https://example.com>` )
- Code ( <code>\`code\`</code> )
- `<br>` (for newline)
- Image (`![Image](path/to/image.png)` )

#### Style for syntax

Create a layout named `style` and add a `Text box` to enter specific word. The styles (`bold`, `italic`, `underline`, `backgroundColor`, `foregroundColor`, `fontFamily`) will be applied as the style for each Markdown syntax.

![img](img/style.png)

| Word | |
| --- | --- |
| `bold` | style for **bold**. |
| `italic` | style for *italic*. |
| `link` | style for [link](#). |
| `code` | style for `code`. |
| (other word) | style for content of inline HTML elements with matching class name ( e.g. `<span class="notice">THIS IS NOTICE</span>` ) |

#### Code blocks to images

By using the `--code-block-to-image-command (-c)` option, you can convert [Markdown code blocks](testdata/codeblock.md) to images. The specified command is executed for each code block, and its standard output is treated as an image.

```console
$ deck apply --code-block-to-image-command "some-command" -i xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
```

The command is executed with `bash -c`.
The command must output image data (PNG, JPEG, GIF) to standard output.

##### How to receive values

From code blocks like the following, you can obtain the optional language identifier `go` and the content within the code block.

    ```go
    package main

    import "fmt"

    func main() {
    	fmt.Println("Hello, 世界")
    }
    ```

There are three ways to receive code block information within the command:

1. **Receive from standard input**
   - The content of the code block is passed as standard input
   - The optional language identifier cannot be obtained, so use it in combination with other methods

2. **Receive as environment variables**
   - `CODEBLOCK_LANG`: Optional language identifier of the code block (e.g., `go`, `python`)
   - `CODEBLOCK_CONTENT`: Content of the code block

3. **Receive with template syntax ( with [expr-lang](https://expr-lang.org/) )**
   - `{{lang}}`: Optional language identifier of the code block
   - `{{content}}`: Content of the code block
   - `{{env.XXX}}`: Value of environment variable XXX

These methods can be used in combination, and you can choose the appropriate method according to the command requirements.

##### Examples

```console
# Convert Mermaid diagrams to images
$ deck apply -c 'mmdc -i - -o output.png --quiet; cat output.png' -i xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
```

```console
# Generate code images with syntax highlighting (e.g., silicon)
$ deck apply -c 'silicon -l {{lang == "" ? "md" : lang}} -o output.png; cat output.png' -i xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
```

```console
# Use different tools depending on the language
$ deck apply -c 'if [ {{lang}} = "mermaid" ]; then mmdc -i - -o output.png --quet; else silicon -l {{lang == "" ? "md" : lang}} --output output.png; fi; cat output.png' -i xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
```

### Comment

The comments `<!--` `-->` are used as a speaker notes or page config.

## Page config

If the comment `<!--` `-->` can be JSON-encoded, it will be processed as page config.

```markdown
<!-- {"layout": "title-and-body"} -->
```

### `"layout":`

It is possible to specify the page layout.

The layout name (e.g. `title-and-body`) is specified.

```markdown
<!-- {"layout": "title-and-body"} -->
```

![img](img/layout_name.png)

> [!TIP]
> With `deck ls-layouts` it is possible to obtain a list of the layout names for a specific presentation.
>
> ```console
> $ deck ls-layouts -i xxxxxXXXXxxxxxXXXXxxxxxxxxxx
> title
> section
> section-dark
> title-and-body
> title-and-body-half
> title-and-body-2col
> title-and-body-3col
> ```

### `"freeze":`

It is possible to skip the operation of the target page.

> [!TIP]
> If you set it to a page that has been completed with layout and design, the page will not be modified unnecessarily by deck.

```markdown
<!-- {"freeze": true} -->
```

## Integration

- [zonuexe/deck-slides.el](https://github.com/zonuexe/deck-slides.el) ... Creating deck using Markdown and Google Slides.

## With AI agent

By collaborating with AI agents to create Markdown-formatted slides, you may be able to create effective presentations.

For example, it is a good idea to provide the following rules for creating deck slides in the prompt.

    Create a presentation in Markdown according to the following rules.

    # Rules for describing presentations using Markdown

    Unless otherwise specified, please follow the rules below.

    - Use `---` to indicate page breaks in slides.
    - Within each slide, the minimum heading level will be treated as the title, and the next level as the subtitle. Higher level headings will be treated as body content. It is recommended to use only one title heading per slide.
    - The following syntax can be used in the page body. Note that it cannot be used in headings.
        - Bold ( `**bold**` )
        - Italic ( `*italic*` `__italic__` )
        - List ( `-` `*` ) - Ordered list ( `1.` `1)` )
        - Link ( `[Link](https://example.com)` )
        - Code ( <code>\`code\`</code> )
        - `<br>` (for newline)
        - Code Block
    - Speaker notes for each page should be written in comments ( `<!--` `-->` ). However, if the comment is JSON, it is a page setting, so do not overwrite it. Instead, write the speaker notes in a new comment.
    - If the comment ( `<!--` `-->` ) is JSON, it is a page setting. If the value `“freeze”:true` is present, do not modify the page content at all.


## Install

**homebrew tap:**

```console
$ brew install k1LoW/tap/deck
```

**go install:**

```console
$ go install github.com/k1LoW/deck/cmd/deck@latest
```

**manually:**

Download binary from [releases page](https://github.com/k1LoW/deck/releases)

## Alternatives

- [googleworkspace/md2googleslides](https://github.com/googleworkspace/md2googleslides): Generate Google Slides from markdown

## License

- [MIT License](LICENSE)
    - Include logo as well as source code.
    - Only logo license can be selected [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
    - Also, if there is no alteration to the logo and it is used for technical information about deck, I would not say anything if the copyright notice is omitted.
