package deck

import "fmt"

// ListSlideURLs lists URLs of the slides in the Google Slides presentation.
func (d *Deck) ListSlideURLs() []string {
	var slideURLs []string
	for _, s := range d.presentation.Slides {
		slideURLs = append(slideURLs, fmt.Sprintf("https://docs.google.com/presentation/d/%s/present?slide=id.%s", d.id, s.ObjectId))
	}
	return slideURLs
}
