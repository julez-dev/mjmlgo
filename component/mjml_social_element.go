package component

import (
	"fmt"
	"io"
	"strings"

	"github.com/julez-dev/mjmlgo/node"
)

const socialNetworkBaseURL = "https://www.mailjet.com/images/theme/v1/icons/ico-social/"

type socialNetwork struct {
	ShareURL        string `json:"share-url,omitempty"`
	BackgroundColor string `json:"background-color"`
	Src             string `json:"src"`
}

func getSocialNetworks() map[string]socialNetwork {
	defaultSocialNetworks := map[string]socialNetwork{
		"facebook": {
			ShareURL:        "https://www.facebook.com/sharer/sharer.php?u=[[URL]]",
			BackgroundColor: "#3b5998",
			Src:             socialNetworkBaseURL + "facebook.png",
		},
		"twitter": {
			ShareURL:        "https://twitter.com/intent/tweet?url=[[URL]]",
			BackgroundColor: "#55acee",
			Src:             socialNetworkBaseURL + "twitter.png",
		},
		"x": {
			ShareURL:        "https://twitter.com/intent/tweet?url=[[URL]]",
			BackgroundColor: "#000000",
			Src:             socialNetworkBaseURL + "twitter-x.png",
		},
		"google": {
			ShareURL:        "https://plus.google.com/share?url=[[URL]]",
			BackgroundColor: "#dc4e41",
			Src:             socialNetworkBaseURL + "google-plus.png",
		},
		"pinterest": {
			ShareURL:        "https://pinterest.com/pin/create/button/?url=[[URL]]&media=&description=",
			BackgroundColor: "#bd081c",
			Src:             socialNetworkBaseURL + "pinterest.png",
		},
		"linkedin": {
			ShareURL:        "https://www.linkedin.com/shareArticle?mini=true&url=[[URL]]&title=&summary=&source=",
			BackgroundColor: "#0077b5",
			Src:             socialNetworkBaseURL + "linkedin.png",
		},
		"instagram": {
			BackgroundColor: "#3f729b",
			Src:             socialNetworkBaseURL + "instagram.png",
		},
		"web": {
			Src:             socialNetworkBaseURL + "web.png",
			BackgroundColor: "#4BADE9",
		},
		"snapchat": {
			Src:             socialNetworkBaseURL + "snapchat.png",
			BackgroundColor: "#FFFA54",
		},
		"youtube": {
			Src:             socialNetworkBaseURL + "youtube.png",
			BackgroundColor: "#EB3323",
		},
		"tumblr": {
			ShareURL:        "https://www.tumblr.com/widgets/share/tool?canonicalUrl=[[URL]]",
			BackgroundColor: "#344356",
			Src:             socialNetworkBaseURL + "tumblr.png",
		},
		"github": {
			Src:             socialNetworkBaseURL + "github.png",
			BackgroundColor: "#000000",
		},
		"xing": {
			ShareURL:        "https://www.xing.com/app/user?op=share&url=[[URL]]",
			BackgroundColor: "#296366",
			Src:             socialNetworkBaseURL + "xing.png",
		},
		"vimeo": {
			Src:             socialNetworkBaseURL + "vimeo.png",
			BackgroundColor: "#53B4E7",
		},
		"medium": {
			Src:             socialNetworkBaseURL + "medium.png",
			BackgroundColor: "#000000",
		},
		"soundcloud": {
			Src:             socialNetworkBaseURL + "soundcloud.png",
			BackgroundColor: "#EF7F31",
		},
		"dribbble": {
			Src:             socialNetworkBaseURL + "dribbble.png",
			BackgroundColor: "#D95988",
		},
	}

	// This loop replicates the JavaScript logic that adds the "-noshare" variants.
	// We iterate over a copy of the original map's keys to avoid modifying the map while iterating.

	// Create a slice to hold the keys to iterate over.
	// This is important because you cannot safely modify a map while ranging over it.
	keys := make([]string, 0, len(defaultSocialNetworks))
	for k := range defaultSocialNetworks {
		keys = append(keys, k)
	}

	for _, key := range keys {
		val := defaultSocialNetworks[key] // Get the original struct

		// Create a copy and modify the ShareURL
		noShareVal := val
		noShareVal.ShareURL = "[[URL]]"

		// Add the new entry to the map
		defaultSocialNetworks[key+"-noshare"] = noShareVal
	}

	return defaultSocialNetworks
}

type MJMLSocialElement struct{}

func (s MJMLSocialElement) applyDefaults(n *node.Node) {
	defaults := map[string]string{
		"alt":             "",
		"align":           "left",
		"icon-position":   "left",
		"color":           "#000",
		"border-radius":   "3px",
		"font-family":     "Ubuntu, Helvetica, Arial, sans-serif",
		"font-size":       "13px",
		"line-height":     "1",
		"padding":         "4px",
		"text-padding":    "4px 4px 4px 0",
		"target":          "_blank",
		"text-decoration": "none",
		"vertical-align":  "middle",
	}

	for key, value := range defaults {
		if _, ok := n.GetAttributeValue(key); !ok {
			n.SetAttribute(key, value)
		}
	}
}

func (s MJMLSocialElement) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	s.applyDefaults(n)

	attrs := s.getSocialAttributes(n)
	styles := s.getStyles(n)
	var (
		src        = attrs["src"]
		srcset     = attrs["srcset"]
		sizes      = attrs["sizes"]
		href       = attrs["href"]
		iconSize   = attrs["icon-size"]
		iconHeight = attrs["icon-height"]
	)

	_, hasLink := n.GetAttributeValue("href")
	iconPosition := n.GetAttributeValueDefault("icon-position")

	makeIcon := func() string {
		b := strings.Builder{}

		_, _ = b.WriteString("<td " + inlineAttributes{"style": styles["td"].InlineString()}.InlineString() + ">")
		_, _ = b.WriteString("<table " + inlineAttributes{
			"border":      "0",
			"cellpadding": "0",
			"cellspacing": "0",
			"role":        "presentation",
			"style":       styles["table"].InlineString(),
		}.InlineString() + ">")
		_, _ = b.WriteString("<tbody><tr>\n")
		_, _ = b.WriteString("<td " + inlineAttributes{"style": styles["icon"].InlineString()}.InlineString() + ">")
		if hasLink {
			_, _ = b.WriteString("<a " + inlineAttributes{"href": href, "rel": n.GetAttributeValueDefault("rel"), "target": n.GetAttributeValueDefault("target")}.InlineString() + ">")
		}

		imgAttr := inlineAttributes{
			"alt":    n.GetAttributeValueDefault("alt"),
			"title":  n.GetAttributeValueDefault("title"),
			"src":    src,
			"srcset": srcset,
			"sizes":  sizes,
			"style":  styles["img"].InlineString(),
			"width":  removeNonNumeric(iconSize),
		}
		if iconHeight != "" {
			imgAttr["height"] = removeNonNumeric(iconHeight)
		} else {
			imgAttr["height"] = removeNonNumeric(iconHeight)
		}

		_, _ = b.WriteString("<img " + imgAttr.InlineString() + " />")

		if hasLink {
			_, _ = b.WriteString("</a>")
		}

		_, _ = b.WriteString("</tr></tbody></table></td>\n")
		return b.String()
	}

	makeContent := func() string {
		if n.Content == "" {
			return ""
		}

		b := strings.Builder{}
		_, _ = b.WriteString("<td " + inlineAttributes{"style": styles["tdText"].InlineString()}.InlineString() + ">")
		if hasLink {
			_, _ = b.WriteString("<a " + inlineAttributes{
				"href":   href,
				"style":  styles["text"].InlineString(),
				"rel":    n.GetAttributeValueDefault("rel"),
				"target": n.GetAttributeValueDefault("target"),
			}.InlineString() + ">")
		} else {
			_, _ = b.WriteString("<span " + inlineAttributes{
				"style": styles["text"].InlineString(),
			}.InlineString() + ">")
		}

		_, _ = b.WriteString(n.Content)
		if hasLink {
			_, _ = b.WriteString("</a>")
		} else {
			_, _ = b.WriteString("</span>")
		}
		_, _ = b.WriteString("</td>\n")

		return b.String()
	}

	renderLeft := func() string {
		return fmt.Sprintf("%s %s", makeIcon(), makeContent())
	}

	renderRight := func() string {
		return fmt.Sprintf("%s %s", makeContent(), makeIcon())
	}

	_, _ = io.WriteString(w, "<tr "+inlineAttributes{
		"class": n.GetAttributeValueDefault("css-class"),
	}.InlineString()+">\n")

	if iconPosition == "left" {
		_, _ = io.WriteString(w, renderLeft())
	} else {
		_, _ = io.WriteString(w, renderRight())
	}

	_, _ = io.WriteString(w, "</tr>\n")

	return nil
}

func (s MJMLSocialElement) getSocialAttributes(n *node.Node) inlineAttributes {
	networks := getSocialNetworks()
	network, hasNetwork := networks[n.GetAttributeValueDefault("name")]

	href := n.GetAttributeValueDefault("href")

	if hasNetwork && network.ShareURL != "" && href != "" {
		href = strings.ReplaceAll(network.ShareURL, "[[URL]]", href)
	}

	finalAttr := make(inlineAttributes)
	finalAttr["href"] = href

	attrKeys := [...]string{
		"icon-size",
		"icon-height",
		"srcset",
		"sizes",
		"src",
		"background-color",
	}

	for _, key := range attrKeys {
		if val, ok := n.GetAttributeValue(key); ok && val != "" {
			finalAttr[key] = val
			continue
		}

		switch key {
		case "src":
			finalAttr[key] = network.Src
		case "background-color":
			finalAttr[key] = network.BackgroundColor
		}
	}

	return finalAttr
}

func (s MJMLSocialElement) getStyles(n *node.Node) map[string]inlineStyle {
	socialAttr := s.getSocialAttributes(n)

	var (
		iconSize        = socialAttr["icon-size"]
		iconHeight      = socialAttr["icon-height"]
		backgroundColor = socialAttr["background-color"]
	)

	tdStyle := inlineStyle{
		{Property: "padding", Value: n.GetAttributeValueDefault("padding")},
		{Property: "padding-top", Value: n.GetAttributeValueDefault("padding-top")},
		{Property: "padding-right", Value: n.GetAttributeValueDefault("padding-right")},
		{Property: "padding-bottom", Value: n.GetAttributeValueDefault("padding-bottom")},
		{Property: "padding-left", Value: n.GetAttributeValueDefault("padding-left")},
		{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")},
	}

	tableStyle := inlineStyle{
		{Property: "background", Value: backgroundColor},
		{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
		{Property: "width", Value: iconSize},
	}

	iconStyle := inlineStyle{
		{Property: "padding", Value: n.GetAttributeValueDefault("icon-padding")},
		{Property: "font-size", Value: "0"},
		{Property: "vertical-align", Value: n.GetAttributeValueDefault("vertical-align")},
		{Property: "width", Value: iconSize},
	}

	if iconHeight != "" {
		iconStyle = append(iconStyle, inlineStyle{{Property: "height", Value: iconHeight}}...)
	} else {
		iconStyle = append(iconStyle, inlineStyle{{Property: "height", Value: iconSize}}...)
	}

	imgStyle := inlineStyle{
		{Property: "border-radius", Value: n.GetAttributeValueDefault("border-radius")},
		{Property: "display", Value: "block"},
	}

	tdText := inlineStyle{
		{Property: "vertical-align", Value: "middle"},
		{Property: "padding", Value: n.GetAttributeValueDefault("text-padding")},
	}

	textStyle := inlineStyle{
		{Property: "color", Value: n.GetAttributeValueDefault("color")},
		{Property: "font-size", Value: n.GetAttributeValueDefault("font-size")},
		{Property: "font-weight", Value: n.GetAttributeValueDefault("font-weight")},
		{Property: "font-style", Value: n.GetAttributeValueDefault("font-style")},
		{Property: "font-family", Value: n.GetAttributeValueDefault("font-family")},
		{Property: "line-height", Value: n.GetAttributeValueDefault("line-height")},
		{Property: "text-decoration", Value: n.GetAttributeValueDefault("text-decoration")},
	}

	return map[string]inlineStyle{
		"td":     tdStyle,
		"table":  tableStyle,
		"icon":   iconStyle,
		"img":    imgStyle,
		"tdText": tdText,
		"text":   textStyle,
	}
}
