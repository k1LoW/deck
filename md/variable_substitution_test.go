package md

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/deck"
)

func TestApplyVariableSubstitution(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		wantErr  bool
		check    func(t *testing.T, md *MD)
	}{
		{
			name: "Simple variable substitution in title",
			markdown: `---
variables:
  company: "Acme Corp"
  year: "2024"
---

# Welcome to {{company}}

This presentation is for {{year}}.`,
			wantErr: false,
			check: func(t *testing.T, md *MD) {
				if len(md.Contents) != 1 {
					t.Fatalf("expected 1 content, got %d", len(md.Contents))
				}
				content := md.Contents[0]
				if len(content.Titles) != 1 || content.Titles[0] != "Welcome to Acme Corp" {
					t.Errorf("expected title 'Welcome to Acme Corp', got %v", content.Titles)
				}
				// Check body
				if len(content.Bodies) < 1 || len(content.Bodies[0].Paragraphs) < 1 {
					t.Fatalf("expected at least one body paragraph")
				}
				bodyText := content.Bodies[0].Paragraphs[0].Fragments[0].Value
				if !strings.Contains(bodyText, "This presentation is for 2024") {
					t.Errorf("expected body to contain 'This presentation is for 2024', got %s", bodyText)
				}
			},
		},
		{
			name: "Variable substitution in subtitle and multiple bodies",
			markdown: `---
variables:
  product: "CloudSync"
  version: "3.0"
---

# {{product}} Overview

## Version {{version}} Features

- Improved {{product}} performance
- New {{product}} dashboard`,
			wantErr: false,
			check: func(t *testing.T, md *MD) {
				content := md.Contents[0]
				if content.Titles[0] != "CloudSync Overview" {
					t.Errorf("expected title 'CloudSync Overview', got %s", content.Titles[0])
				}
				if len(content.Subtitles) != 1 || content.Subtitles[0] != "Version 3.0 Features" {
					t.Errorf("expected subtitle 'Version 3.0 Features', got %v", content.Subtitles)
				}
				// Check list items
				foundPerformance := false
				foundDashboard := false
				for _, body := range content.Bodies {
					for _, para := range body.Paragraphs {
						for _, frag := range para.Fragments {
							if strings.Contains(frag.Value, "Improved CloudSync performance") {
								foundPerformance = true
							}
							if strings.Contains(frag.Value, "New CloudSync dashboard") {
								foundDashboard = true
							}
						}
					}
				}
				if !foundPerformance {
					t.Error("expected to find 'Improved CloudSync performance' in body")
				}
				if !foundDashboard {
					t.Error("expected to find 'New CloudSync dashboard' in body")
				}
			},
		},
		{
			name: "Variable substitution in speaker notes",
			markdown: `---
variables:
  presenter: "John Doe"
  date: "January 2024"
---

# Introduction

<!--
Presenter: {{presenter}}
Date: {{date}}
-->`,
			wantErr: false,
			check: func(t *testing.T, md *MD) {
				content := md.Contents[0]
				if len(content.Comments) != 1 {
					t.Fatalf("expected 1 comment, got %d", len(content.Comments))
				}
				expectedComment := "Presenter: John Doe\nDate: January 2024"
				if content.Comments[0] != expectedComment {
					t.Errorf("expected comment '%s', got '%s'", expectedComment, content.Comments[0])
				}
			},
		},
		{
			name: "No variables defined",
			markdown: `---
title: "Test Presentation"
---

# No substitution here

This is plain text.`,
			wantErr: false,
			check: func(t *testing.T, md *MD) {
				content := md.Contents[0]
				if content.Titles[0] != "No substitution here" {
					t.Errorf("expected title 'No substitution here', got %s", content.Titles[0])
				}
			},
		},
		{
			name: "Empty variables map",
			markdown: `---
variables:
---

# Title without variables`,
			wantErr: false,
			check: func(t *testing.T, md *MD) {
				content := md.Contents[0]
				if content.Titles[0] != "Title without variables" {
					t.Errorf("expected title 'Title without variables', got %s", content.Titles[0])
				}
			},
		},
		{
			name: "Variables in nested headings",
			markdown: `---
variables:
  section: "Configuration"
  subsection: "Advanced Settings"
---

# Main Title

### {{section}} - {{subsection}}

Content here`,
			wantErr: false,
			check: func(t *testing.T, md *MD) {
				content := md.Contents[0]
				// Check heading level 3
				if headings, ok := content.Headings[3]; ok && len(headings) > 0 {
					expected := "Configuration - Advanced Settings"
					if headings[0] != expected {
						t.Errorf("expected heading '%s', got '%s'", expected, headings[0])
					}
				} else {
					t.Error("expected heading at level 3")
				}
			},
		},
		{
			name: "Complex variable expressions",
			markdown: `---
variables:
  env: "production"
  debug: "false"
---

# {{env == "production" ? "Production" : "Development"}} Environment

Debug mode: {{debug == "true" ? "Enabled" : "Disabled"}}`,
			wantErr: false,
			check: func(t *testing.T, md *MD) {
				content := md.Contents[0]
				if content.Titles[0] != "Production Environment" {
					t.Errorf("expected title 'Production Environment', got %s", content.Titles[0])
				}
				bodyText := content.Bodies[0].Paragraphs[0].Fragments[0].Value
				if !strings.Contains(bodyText, "Debug mode: Disabled") {
					t.Errorf("expected body to contain 'Debug mode: Disabled', got %s", bodyText)
				}
			},
		},
		{
			name: "Variable in formatted text",
			markdown: `---
variables:
  important: "CRITICAL"
---

# Title

**{{important}}**: This is important.
*Version {{important}}*`,
			wantErr: false,
			check: func(t *testing.T, md *MD) {
				content := md.Contents[0]
				foundBold := false
				foundItalic := false
				for _, body := range content.Bodies {
					for _, para := range body.Paragraphs {
						for _, frag := range para.Fragments {
							if frag.Bold && strings.Contains(frag.Value, "CRITICAL") {
								foundBold = true
							}
							if frag.Italic && strings.Contains(frag.Value, "Version CRITICAL") {
								foundItalic = true
							}
						}
					}
				}
				if !foundBold {
					t.Error("expected to find bold text with 'CRITICAL'")
				}
				if !foundItalic {
					t.Error("expected to find italic text with 'Version CRITICAL'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, err := Parse(".", []byte(tt.markdown))
			if err != nil {
				t.Fatalf("failed to parse markdown: %v", err)
			}

			// Apply ToSlides which includes variable substitution
			ctx := context.Background()
			_, err = md.ToSlides(ctx, "")
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ToSlides() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, md)
			}
		})
	}
}

func TestApplyVariableSubstitutionToBody(t *testing.T) {
	tests := []struct {
		name    string
		body    *deck.Body
		store   map[string]any
		want    *deck.Body
		wantErr bool
	}{
		{
			name: "Simple substitution",
			body: &deck.Body{
				Paragraphs: []*deck.Paragraph{
					{
						Fragments: []*deck.Fragment{
							{Value: "Hello {{name}}!"},
						},
					},
				},
			},
			store: map[string]any{"name": "World"},
			want: &deck.Body{
				Paragraphs: []*deck.Paragraph{
					{
						Fragments: []*deck.Fragment{
							{Value: "Hello World!"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Multiple fragments",
			body: &deck.Body{
				Paragraphs: []*deck.Paragraph{
					{
						Fragments: []*deck.Fragment{
							{Value: "Welcome to "},
							{Value: "{{company}}", Bold: true},
							{Value: " in {{year}}"},
						},
					},
				},
			},
			store: map[string]any{"company": "Acme Corp", "year": "2024"},
			want: &deck.Body{
				Paragraphs: []*deck.Paragraph{
					{
						Fragments: []*deck.Fragment{
							{Value: "Welcome to "},
							{Value: "Acme Corp", Bold: true},
							{Value: " in 2024"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "Nil body",
			body:    nil,
			store:   map[string]any{"test": "value"},
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applyVariableSubstitutionToBody(tt.body, tt.store)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("applyVariableSubstitutionToBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.want != nil {
				if diff := cmp.Diff(tt.want, tt.body); diff != "" {
					t.Errorf("Body mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestVariableSubstitutionIntegration(t *testing.T) {
	// Test with multiple pages
	markdown := `---
variables:
  conference: "TechCon 2024"
  speaker: "Jane Smith"
  topic: "Cloud Architecture"
---

# {{conference}}

Welcome to {{topic}} by {{speaker}}

---

# {{topic}} Overview

Presented by {{speaker}} at {{conference}}

---

# Thank You

{{speaker}} - {{conference}}`

	md, err := Parse(".", []byte(markdown))
	if err != nil {
		t.Fatalf("failed to parse markdown: %v", err)
	}

	ctx := context.Background()
	slides, err := md.ToSlides(ctx, "")
	if err != nil {
		t.Fatalf("ToSlides() error: %v", err)
	}

	if len(slides) != 3 {
		t.Fatalf("expected 3 slides, got %d", len(slides))
	}

	// Check first slide
	if slides[0].Titles[0] != "TechCon 2024" {
		t.Errorf("slide 1: expected title 'TechCon 2024', got %s", slides[0].Titles[0])
	}

	// Check second slide
	if slides[1].Titles[0] != "Cloud Architecture Overview" {
		t.Errorf("slide 2: expected title 'Cloud Architecture Overview', got %s", slides[1].Titles[0])
	}

	// Check third slide
	if slides[2].Titles[0] != "Thank You" {
		t.Errorf("slide 3: expected title 'Thank You', got %s", slides[2].Titles[0])
	}
	bodyText := slides[2].Bodies[0].String()
	if bodyText != "Jane Smith - TechCon 2024" {
		t.Errorf("slide 3: expected body 'Jane Smith - TechCon 2024', got %s", bodyText)
	}
}