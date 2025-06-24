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
$ deck apply xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
```

#### Watch mode

You can use the `--watch` flag to continuously monitor changes to your markdown file and automatically apply them to the presentation:

```console
$ deck apply --watch xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
```

This is useful during the content creation process as it allows you to see your changes reflected in the presentation in real-time as you edit the markdown file.

> [!NOTE]
> The `--watch` flag cannot be used together with the `--page` flag.

## Support markdown rules

### Insertion rule

`deck` inserts values according to the following rules regardless of the slide layout.

- Heading1 (`#`) is inserted into the title placeholder ( `CENTERED_TITLE` or `TITLE` ) in order.
- Heading2 (`##`) is inserted into the subtitle placeholder ( `SUBTITLE` ) in order.
- All other items are inserted into the body placeholder ( `BODY` ) in order.

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
$ deck apply --code-block-to-image-command "some-command" xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
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
   - `CODEBLOCK_VALUE`: Content of the code block

3. **Receive with template syntax ( with [expr-lang](https://expr-lang.org/) )**
   - `{{lang}}`: Optional language identifier of the code block
   - `{{value}}`: Content of the code block
   - `{{env.XXX}}`: Value of environment variable XXX

These methods can be used in combination, and you can choose the appropriate method according to the command requirements.

##### Examples

```console
# Convert Mermaid diagrams to images
$ deck apply -c 'mmdc -i - -o output.png --quiet; cat output.png' xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
```

```console
# Generate code images with syntax highlighting (e.g., silicon)
$ deck apply -c 'silicon -l {{lang == "" ? "-l md" : "-l " + lang}} -o output.png; cat output.png' xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
```

```console
# Use different tools depending on the language
$ deck apply -c 'if [ {{lang}} = "mermaid" ]; then mmdc -i - -o output.png --quet; else silicon {{lang == "" ? "-l md" : "-l " + lang}} --output output.png; fi; cat output.png' xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
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
> $ deck ls-layouts xxxxxXXXXxxxxxXXXXxxxxxxxxxx
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
