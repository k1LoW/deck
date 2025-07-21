---
defaults:
  - if: page == 1
    layout: title
  - if: titles.size() == 1 && headings[2].size() == 1
    layout: section-purple
  - if: speakerNote.contains("TODO")
    ignore: true
  - if: true
    layout: title-and-body
---

# Title

---

## Section

---

# Title

Hello World

<!--

TODO: more contents

-->
