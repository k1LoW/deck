---
presentationID: your-presentation-id
title: Example of Layout by Heading Level
layout:
  title: title-slide
  h1: part
  h2: chapter
  h3: section
  h4: subsection
---

# Presentation Title

This is the first slide. It will use the "title-slide" layout if specified in the frontmatter.
Without the "title" key in layout, it would use the default title layout.

---

# Part 1: Introduction

This slide will automatically use the "part" layout because it has an H1 heading (and it's not the first slide).

---

## Chapter 1: Getting Started

This slide will automatically use the "chapter" layout because it has an H2 heading.

---

### Section 1.1: Prerequisites

This slide will automatically use the "section" layout because it has an H3 heading.

---

<!-- {"layout": "custom-layout"} -->

## Chapter 2: Custom Layout

Even with the frontmatter configuration, you can still override the layout for specific slides using the JSON comment syntax.

---

#### Subsection: Details

This slide has an H4 heading. It will use the "subsection" layout from the frontmatter h4 mapping.

---

Content without a heading

Slides without headings will use the default layout.