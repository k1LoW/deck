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

## Installation

**Homebrew:**

```console
$ brew install deck
```

**go install:**

```console
$ go install github.com/k1LoW/deck/cmd/deck@latest
```

**Manual installation:**

Download the binary from the [releases page](https://github.com/k1LoW/deck/releases)

## Usage

### Setup

#### Get and set your OAuth client credentials

- Create (or reuse) a developer project at https://console.cloud.google.com.
- Enable [`Google Slides API`](https://console.cloud.google.com/apis/library/slides.googleapis.com) and [`Google Drive API`](https://console.cloud.google.com/apis/library/drive.googleapis.com) at the [`API & Services` page](https://console.cloud.google.com/apis/dashboard).
- Go to the [`Credentials` page](https://console.cloud.google.com/apis/credentials) and click [`+ CREATE CREDENTIALS`](https://console.cloud.google.com/auth/clients/create) at the top.
- Create an `OAuth client ID` type of credentials.
- Choose the type `Desktop app`.
    - Since there is no need to publish the app, add your email address as a test user from [Google Auth Platform / Audience](https://console.cloud.google.com/auth/audience).
- Download the credentials file to `${XDG_DATA_HOME:-~/.local/share}/deck/credentials.json`.

#### For CI/CD automation (Service Account)

If you're setting up `deck` for automated workflows (GitHub Actions, CI/CD pipelines), see [Service Account Setup Guide](docs/setup-service-account.md).

#### Check your setup with `deck doctor`

You can verify if `deck` is ready to use and diagnose any configuration issues with the `deck doctor` command.

### Prepare presentation ID and markdown file with `deck new`

`deck` requires two main components:
- **Presentation ID**: A unique identifier for your Google Slides presentation (e.g., `xxxxxXXXXxxxxxXXXXxxxxxxxxxx` from the URL `https://docs.google.com/presentation/d/xxxxxXXXXxxxxxXXXXxxxxxxxxxx/edit`)
- **Markdown file**: Your slide content written in markdown format

#### When creating a new presentation

You can create a new presentation with the `deck new` command:

```console
$ deck new deck.md --title "Talk about deck"
Applied frontmatter to deck.md
xxxxxXXXXxxxxxXXXXxxxxxxxxxx
```

This will create (or update) the given markdown file with frontmatter containing the presentation ID and title.

##### Reusing theme from an existing presentation

To reuse the theme from an existing presentation, you have two options:

**Option 1: Use the `--base` flag**
```console
$ deck new deck.md --base yyyyyyyYYYYyYYYYYYYyyyyyyyyy --title "Talk about deck"
xxxxxXXXXxxxxxXXXXxxxxxxxxxx
```

**Option 2: Set a default base presentation in your configuration file**

```yaml
# ~/.config/deck/config.yml
basePresentationID: "yyyyyyyYYYYyYYYYYYYyyyyyyyyy"
```

With this configuration, you can reuse the theme from the base presentation without using the `--base` flag. If both the configuration and `--base` flag are present, the `--base` flag takes precedence.

#### When using an existing presentation

Get the presentation ID you want to work with. You can list all presentations with `deck ls`.

```console
$ deck ls
xxxxxXXXXxxxxxXXXXxxxxxxxxxx    My Presentation
yyyyyYYYYyyyyyYYYYyyyyyyyyyy    Team Project Slides
```

> [!NOTE]
> `deck` fully supports Google Shared Drives (Team Drives). Presentations stored in shared drives are automatically included in listings and can be operated on just like personal drive presentations.

To use this presentation, specify it with the `--presentation-id` flag or add it to your markdown file's frontmatter as `presentationID`.

### Write your slides in markdown

Edit your markdown file with your favorite editor. Among horizontal rule syntaxes, three or more consecutive hyphens at the beginning of a line (e.g. `---`) are treated as slide page separators. See [Markdown file format for `deck`](#markdown-file-format-for-deck) for details.

### Apply markdown content to Google Slides with `deck apply`

```console
$ deck apply deck.md
```

#### Watch mode

You can use the `--watch` flag to continuously monitor changes to your markdown file and automatically apply them to the presentation:

```console
$ deck apply --watch deck.md
```

This is useful during the content creation process as it allows you to see your changes reflected in the presentation in real-time as you edit the markdown file.

> [!NOTE]
> The `--watch` flag cannot be used together with the `--page` flag.

### Open presentation in your browser with `deck open`

You can open your Google Slides presentation in your default web browser:

```console
$ deck open deck.md
```

## Markdown file format for `deck`

The Markdown used by `deck` consists of YAML frontmatter and a body section.

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

- `presentationID` (string): Google Slides presentation ID. When specified, you can use the simplified command syntax.
- `title` (string): The title of the presentation. When specified, you can use the simplified command syntax.
- `breaks` (boolean): Control how line breaks are rendered. Default (`false` or omitted) renders line breaks as spaces. When `true`, line breaks in markdown are rendered as actual line breaks in slides. Can also be configured globally in `config.yml`.
- `codeBlockToImageCommand` (string): Command to convert code blocks to images. When specified, code blocks in the presentation will be converted to images using this command. Can also be configured globally in `config.yml`.
- `defaults` (array): Define conditional actions using CEL (Common Expression Language) expressions. Actions are automatically applied to pages based on page structure and content. Only applies to pages without explicit page configuration. Can also be configured globally in `config.yml`.


### Supported Markdown syntax

`deck` supports CommonMark and selected GitHub Flavored Markdown extensions. For comprehensive documentation, see [Markdown Support Documentation](docs/markdown.md).

**Key supported features:**
- Bold ( `**bold**` )
- Italic ( `*italic*` `__italic__` )
- Strikethrough ( `~~strikethrough~~` )
- List ( `-` `*` )
- Ordered list ( `1.` `1)` )
- Link ( `[Link](https://example.com)` )
- Angle bracket autolinks ( `<https://example.com>` )
- Code ( <code>\`code\`</code> )
- `<br>` (for newline)
- Image (`![Image](path/to/image.png)` )
- Block quote ( `> block quote` )
- Table (GitHub Flavored Markdown tables)
- RAW inline HTML (e.g., `<mark>`, `<small>`, `<kbd>`, `<cite>`, `<q>`, `<span>`, `<u>`, `<s>`, `<del>`, `<ins>`, `<sub>`, `<sup>`, `<var>`, `<samp>`, `<data>`, `<dfn>`, `<time>`, `<abbr>`)

#### Line break handling

`deck` provides configurable line break behavior through the `breaks` setting:

- **Default (`breaks: false`)**: Single line breaks become spaces (per CommonMark/GFM specs)
- **With `breaks: true`**: Line breaks are preserved (GitHub-style rendering)
- **For explicit breaks**: Use hard line break syntax (two spaces at line end per CommonMark standard) or `<br>` tag

Example with `breaks: true`:
```markdown
---
breaks: true
---
Text with
line breaks
preserved
```

#### Comments

HTML comments `<!--` `-->` are used for speaker notes or [page configuration](#page-configuration).

## How markdown maps to slide placeholders

`deck` inserts values according to the following rules regardless of the slide layout.

- The shallowest heading level within each slide content is treated as the title and inserted into the title placeholder ( `CENTERED_TITLE` or `TITLE` ) in order.
  - In most cases, this will be H1 (`#`), which is the standard for slide titles
- The next heading level (minimum level + 1) is treated as the subtitle and inserted into the subtitle placeholder ( `SUBTITLE` ) in order.
  - When H1 is used as the title, H2 (`##`) becomes the subtitle
- All other items are inserted into the body placeholder ( `BODY` ) in order.
    - The remaining contents are divided into one or more bodies by headings corresponding to the title or subtitle in the slide.

For example:
- **Standard case**: If a slide contains `#` (H1), then `#` becomes the title and `##` becomes the subtitle
- **Alternative case**: If a slide only contains `##` (H2) or deeper, then `##` becomes the title and `###` becomes the subtitle

> [!NOTE]
> They are inserted in the order they appear in the markdown document, **from the placeholder at the top of the slide** (or from the placeholder on the left if placeholders are at the same height).
>
> Also, if there are not enough placeholders, the remaining contents will not be rendered.

### Example

**Input markdown document:**

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

**Layout and placeholders:**

![img](img/layout.png)

**Result of applying:**

![img](img/result.png)

## Configuration File
`deck` supports global configuration files that provide default settings for all presentations. Configuration files are loaded in the following order:

1. `${XDG_CONFIG_HOME:-~/.config}/deck/config-{profile}.yml` (when using `--profile` option)
2. `${XDG_CONFIG_HOME:-~/.config}/deck/config.yml` (default config file)

The configuration file uses YAML format and supports the same fields as frontmatter. Settings in frontmatter take precedence over configuration file settings, which in turn take precedence over built-in defaults.

### Configuration file example

```yaml
# Global configuration for deck
basePresentationID: "1wIik04tlp1U4SBHTLrSu20dPFlAGTbRHxnqdRFF9nPo"
breaks: true
codeBlockToImageCommand: "go run testdata/txt2img/main.go"
folderID: "1aBcDeFgHiJkLmNoPqRsTuVwXyZ"

defaults:
  # First page should always use title layout
  - if: page == 1
    layout: title
  # Pages with only one title and one H2 heading use section layout
  - if: titles.size() == 1 && headings[2].size() == 1
    layout: section-purple
  # Skip pages with TODO in speaker notes
  - if: speakerNote.contains("TODO")
    skip: true
  # Default layout for all other pages
  - if: true
    layout: title-and-body
```

### Available configuration fields
- **`basePresentationID`** (string): Base presentation ID to use as a template when creating new presentations
- **`breaks`** (boolean): Global line break rendering behavior
- **`codeBlockToImageCommand`** (string): Global command to convert code blocks to images
- **`folderID`** (string): Default folder ID to create presentations and upload temporary images to
- **`defaults`** (array): A series of conditions and actions written in CEL expressions for default page configs

### Configuration precedence
Settings are applied in the following order (highest to lowest priority):

1. **Command-line options** - Takes highest precedence when available
2. **Frontmatter settings** - Takes precedence over config file
3. **Configuration file settings** - Applied when neither command-line nor frontmatter specify the setting
4. **Built-in defaults** - Used when no other source specifies a setting

This allows you to set organization-wide or project-wide defaults while still maintaining the flexibility to override them on a per-file basis using frontmatter or command-line options.

## Advanced features

### Style for syntax

Create a layout named `style` and add a `Text box` to enter specific words. The styles (`bold`, `italic`, `underline`, `backgroundColor`, `foregroundColor`, `fontFamily`) will be applied as the style for each Markdown syntax.

![img](img/style.png)

| Word | |
| --- | --- |
| `bold` | style for **bold**. |
| `italic` | style for *italic*. |
| `link` | style for [link](#). |
| `code` | style for `code`. |
| `del` | style for ~~strikethrough~~ (also applies to `<del>` tag). |
| `blockquote` | style for block quote. |
| HTML element names | style for content of inline HTML elements ( e.g. `<cite>`, `<q>`, `<s>`, `<ins>`, etc. ) |
| (other word) | style for content of inline HTML elements with matching class name ( e.g. `<span class="notice">THIS IS NOTICE</span>` ) |

### Code blocks to images

You can convert [Markdown code blocks](testdata/codeblock.md) to images by specifying a command that outputs image data (PNG, JPEG, GIF) to standard output or to a file by using the `{{output}}` placeholder for the output file path.

```console
$ deck apply --code-block-to-image-command "some-command" -i xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
```

Alternatively, you can configure it in `config.yml` or frontmatter:
```yaml
codeBlockToImageCommand: "some-command"
```


#### How to receive values

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
   - `CODEBLOCK_OUTPUT`: Path to a temporary output file

3. **Receive with CEL template syntax**
   - `{{lang}}`: Optional language identifier of the code block
   - `{{content}}`: Content of the code block
   - `{{output}}`: Path to a temporary output file
   - `{{env.XXX}}`: Value of environment variable XXX

   The template expansion uses CEL (Common Expression Language) for evaluating expressions within `{{ }}` delimiters. This supports:
   - Ternary operators: `{{ lang == "" ? "md" : lang }}`
   - String concatenation: `{{ "prefix_" + lang }}`
   - Boolean logic: `{{ lang != "" && content.contains("main") }}`
   - Arithmetic operations: `{{ count + 1 }}`

These methods can be used in combination, and you can choose the appropriate method according to the command requirements.

> [!NOTE]
> When `{{output}}` is not specified, `deck` reads the image data from the command's stdout. When `{{output}}` is specified, the command should write the image to that file path, and `deck` will read the image data from that file.

#### Examples

```console
# Convert Mermaid diagrams to images
$ deck apply -c 'mmdc -i - -o {{output}} --quiet' deck.md
```

```console
# Generate code images using the built-in text-to-image tool
$ deck apply -c 'go run testdata/txt2img/main.go' deck.md
```

```console
# Use different tools depending on the language
$ deck apply -c 'if [ {{lang}} = "mermaid" ]; then mmdc -i - -o {{output}} --quiet; else go run testdata/txt2img/main.go; fi' deck.md

# Alternatively, you can use Songmu/laminate to use the appropriate tool for each language.
$ deck apply -c 'laminate' deck.md
```

## Page configuration

You can configure individual pages using JSON comments. Available settings:

- **`"layout"`**: Specifies which slide layout to use from your presentation template. Different layouts have different placeholder arrangements (title, subtitle, body, etc.)
- **`"freeze"`**: Prevents `deck` from modifying the page (useful for slides with completed designs)
- **`"ignore"`**: Excludes the page from slide generation (for drafts, notes, or unused content)
- **`"skip"`**: Creates the slide but skips it during presentation playback (automatically advances to next slide)

```markdown
<!-- {"layout": "title-and-body"} -->
# Your slide content

---

<!-- {"freeze": true} -->
# This slide won't be modified

---

<!-- {"ignore": true} -->
# This content won't appear in slides

---

<!-- {"skip": true} -->
# This slide will be skipped during presentation
```

> [!TIP]
> Use `deck ls-layouts` to see all available layout names for your presentation:
> ```console
> $ deck ls-layouts deck.md
> title
> section
> section-dark
> title-and-body
> title-and-body-half
> title-and-body-2col
> title-and-body-3col
> ```

![img](img/layout_name.png)

## Default page configs with CEL expressions

The `defaults` field in Frontmatter or configuration file allows you to define default page configs using CEL (Common Expression Language) expressions. This feature automatically sets layouts and controls page behavior based on their structure and content, eliminating the need for manual configuration on each page.

### Available actions

The following actions can be applied to pages through the `defaults` configuration:

- **`layout`**: Set layout automatically
- **`freeze`**: Freeze page from modifications
- **`ignore`**: Exclude page from generation
- **`skip`**: Skip page during presentation

```yaml
---
defaults:
  # First page should always use title layout
  - if: page == 1
    layout: title
  # Pages with only one title and one H2 heading use section layout
  - if: titles.size() == 1 && headings[2].size() == 1
    layout: section-purple
  # Skip pages with TODO in speaker notes
  - if: speakerNote.contains("TODO")
    skip: true
  # Default layout for all other pages
  - if: true
    layout: title-and-body
---
```

### Available CEL variables

| Variable | Type | Description |
|----------|------|-------------|
| `page` | `int` | Current page number (1-based) |
| `pageTotal` | `int` | Total number of pages |
| `titles` | `[]string` | List of titles in the page |
| `subtitles` | `[]string` | List of subtitles in the page |
| `bodies` | `[]string` | List of body texts in the page |
| `blockQuotes` | `[]string` | List of block quotes in the page |
| `codeBlocks` | `[]CodeBlock` | List of code blocks in the page |
| `images` | `[]Image` | List of images in the page |
| `comments` | `[]string` | List of comments in the page |
| `headings` | `map[int][]string` | Headings grouped by level |
| `speakerNote` | `string` | Speaker note |
| `topHeadingLevel` | `int` | The highest heading level in the content |

### CEL condition examples

- `page == 1` - First page only
- `titles.size() == 0` - Pages without titles
- `codeBlocks.size() > 0` - Pages containing code blocks
- `headings[3].size() >= 2` - Pages with 2 or more H3 headings
- `bodies[0].contains("TODO")` - Pages with TODO in first body text
- `page > pageTotal - 3` - Last 3 pages
- `images.size() >= 2` - Pages with 2 or more images

### Important notes

- **Evaluation order**: Conditions are evaluated in order, and the first matching condition's action is applied
- **Priority**: Default actions only apply to pages without explicit page configuration (via JSON comments like `<!-- {"layout": "title"} -->`)
- **Performance**: Using `ignore` for unnecessary content improves processing speed
- **Workflow**: This feature enables automatic page management based on content patterns, reducing manual configuration overhead

## Profile support

`deck` supports multiple profiles through the `--profile` option. This feature allows you to manage separate profiles (authentication Google accounts or environments).

```console
$ deck apply deck.md --profile work
$ deck ls --profile personal
$ deck new presentation.md --profile project-a
```

When using profiles, authentication files are managed as follows:
- **Credentials file**: `credentials-{profile}.json` - Create this file manually to use profile-specific credentials. If this file exists, it will be automatically used for the specified profile.
- **Token file**: `token-{profile}.json` - This file is automatically generated when you use the `--profile` option and complete the OAuth authentication process.

## FAQ

### A setting permission error occurs during image upload, as shown below

```
Error: failed to apply page: failed to upload image: failed to set permission for image: googleapi: Error 403: The user does not have sufficient permissions for this file., insufficientFilePermissions
```

Please verify that you can grant public read access to files in Google Drive. Organizational policies may prevent this. If restricted, create a folder with appropriate permissions and specify its ID using the `--folder-id` flag during `deck apply`.

To insert images into slides, `deck` temporarily uploads image files to Google Drive, obtains a publicly accessible URL from there, and passes it to the API. Therefore, you must be able to grant reader permissions to anyone for image files on Google Drive.

## Integration

- [zonuexe/deck-slides.el](https://github.com/zonuexe/deck-slides.el) ... Emacs integration for creating presentations using Markdown and Google Slides
- [Songmu/laminate](https://github.com/Songmu/laminate) ... Selects image generation commands based on code block language. Useful for converting code blocks to images

## With AI agent

By collaborating with AI agents to create Markdown-formatted slides, you may be able to create effective presentations.

<details>
<summary>It is a good idea to provide the following rules for creating presentations with `deck` in the prompt. (Click to expand)</summary>

    Create a presentation in Markdown according to the following rules.

    # Rules for describing presentations using Markdown

    Unless otherwise specified, please follow the rules below.

    ## Basic Structure
    - Use a line containing only three or more consecutive hyphens (`---`, `----`, etc.) from the beginning to the end of the line to indicate page breaks between slides.
    - Other horizontal rule elements (like `- - -`, `***`, `___`) remain in the content as visual separators and can be used to separate multiple body placeholders.
    - Within each slide, the minimum heading level will be treated as the title, and the next level as the subtitle. Higher level headings will be treated as body content. It is recommended to use only one title heading per slide.

    ## YAML Frontmatter
    You can include YAML frontmatter at the beginning of the file:
    ```yaml
    ---
    title: "Presentation Title"
    presentationID: "presentation_id"
    breaks: true
    author: "Author Name"
    date: "2024-01-01"
    tags: ["tag1", "tag2"]
    custom:
      nested: "value"
      number: 42
    ---
    ```

    ## Supported Markdown Syntax
    The following syntax can be used in the slide content:

    ### Text Formatting
    - **Bold** (`**bold**`)
    - *Italic* (`*italic*` or `__italic__`)
    - `Inline code` (<code>\`code\`</code>)
    - Combined formatting (e.g., ***bold and italic***)

    ### Lists
    - Bullet lists (`-` or `*`)
    - Numbered lists (`1.` or `1)`)
    - Nested lists (with proper indentation)
    - Alphabetical lists (a. b. c.)

    ### Links and Images
    - Links (`[Link text](https://example.com)`)
    - Angle bracket autolinks (`<https://example.com>`)
    - Images (`![alt text](image.jpg)`)
    - Supports PNG, JPEG, GIF formats
    - Supports both local files and URLs (HTTP/HTTPS)

    ### Block Elements
    - Block quotes (`> quoted text`)
    - Nested block quotes
    - Code blocks with language specification:
      ```language
      code content
      ```
    - Mermaid diagrams (in code blocks with `mermaid` language)

    ### Tables
    - GitHub Flavored Markdown (GFM) tables
    - Supports table headers with automatic bold formatting
    - Cell content can include inline formatting (bold, italic, code)
    - Example:
      ```markdown
      | Header 1 | Header 2 | Header 3 |
      |----------|----------|----------|
      | Cell 1   | **Bold** | `code`   |
      | Cell 2   | *Italic* | Normal   |
      ```
    - Header rows are automatically styled with bold text and gray background
    - Tables created by users in Google Slides are preserved

    ### HTML Elements
    You can use the following HTML inline elements:
    - `<strong>`, `<em>`, `<b>`, `<i>`, `<mark>`, `<small>`
    - `<code>`, `<kbd>`, `<cite>`, `<q>`, `<ruby>`, `<rt>`
    - `<span>`, `<u>`, `<s>`, `<sub>`, `<sup>`, `<var>`
    - `<samp>`, `<data>`, `<dfn>`, `<time>`, `<abbr>`, `<rp>`
    - `<br>` (for line breaks)
    - Use `class` attribute for custom styling

    ### Line Break Handling
    - Default (`breaks: false`): Soft line breaks become spaces
    - With `breaks: true`: Soft line breaks become actual line breaks
    - Use `<br>` tags for explicit line breaks

    ## Page Configuration
    Use HTML comments for page settings and speaker notes:
    - Page settings: `<!-- {"layout": "title-and-body"} -->`
    - Available settings: `"freeze": true`, `"ignore": true`, `"skip": true`
    - Speaker notes: `<!-- This is a speaker note -->` (use separate comments for notes)

    ## Important Notes
    - If a comment (`<!-- -->`) contains JSON, it's a page setting - do not overwrite it
    - If `"freeze": true` is present in page settings, do not modify that page content at all
    - Write speaker notes in separate comments, not in JSON configuration comments
    - Code blocks can be converted to images using the `--code-block-to-image-command` option

</details>

## Alternatives

- [googleworkspace/md2googleslides](https://github.com/googleworkspace/md2googleslides): Generates Google Slides from markdown

## License

- [MIT License](LICENSE)
    - Include logo as well as source code.
    - Only logo license can be selected [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
    - Also, if there is no alteration to the logo and it is used for technical information about `deck`, I would not say anything if the copyright notice is omitted.
