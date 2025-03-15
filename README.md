# deck

`deck` is a tool for creating deck using Google Slides.

## Usage

### Setup

#### Get and set your OAuth client ID credentials

- Create (or reuse) a developer project at https://console.cloud.google.com/apis/dashboard
- Enable `Google Slides API` and `Google Drive API` at `API & Services` page.
- Go to `Credentials` page and click `+ CREATE CREDENTIALS` at the top.
- Create `OAuth client ID` type of credentials.
- Choose type `Desktop app"`.
- Download credentials file to `~/.local/share/deck/credentials.json` ( or `${XDG_DATA_HOME}/deck/credentials.json` ).

### Get presentation ID

Get the presentation ID you want to operate. You can get a list with `deck ls`.

For example, presentation ID is `xxxxxXXXXxxxxxXXXXxxxxxxxxxx` of https://docs.google.com/presentation/d/xxxxxXXXXxxxxxXXXXxxxxxxxxxx/edit .

### Write desk in markdown

The slide pages are represented by dividing them with horizontal lines `---`.

The `---` at the beginning of the markdown is ignored.

### Apply desk written in markdown to Google Slides presentation

```console
$ deck apply xxxxxXXXXxxxxxXXXXxxxxxxxxxx deck.md
```

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

## References

- [googleworkspace/md2googleslides](https://github.com/googleworkspace/md2googleslides): Generate Google Slides from markdown

