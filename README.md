# deck

`deck` is a tool for creating deck using Google Slides.

## Usage

### Get and set your OAuth client ID credentials

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
