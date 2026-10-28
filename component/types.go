package component

type MJMLComponentName = string

const (
	MJMLTagName MJMLComponentName = "mjml"
	HeadTagName MJMLComponentName = "mj-head"
	BodyTagName MJMLComponentName = "mj-body"
	RawTagName  MJMLComponentName = "mj-raw"

	// IncludeTagName is the include element. It is resolved before validation
	// and never reaches a component, so no component renders it - but it is
	// part of the language, and a custom component must not shadow it.
	IncludeTagName MJMLComponentName = "mj-include"

	// head specific
	AttributesTagName MJMLComponentName = "mj-attributes"
	BreakpointTagName MJMLComponentName = "mj-breakpoint"
	TextTagName       MJMLComponentName = "mj-text"
	ClassTagName      MJMLComponentName = "mj-class"
	AllTagName        MJMLComponentName = "mj-all"
	FontTagName       MJMLComponentName = "mj-font"
	TitleTagName      MJMLComponentName = "mj-title"
	PreviewTagName    MJMLComponentName = "mj-preview"
	StyleTagName      MJMLComponentName = "mj-style"

	HTMLAttributesTagName MJMLComponentName = "mj-html-attributes"
	SelectorTagName       MJMLComponentName = "mj-selector"
	HTMLAttributeTagName  MJMLComponentName = "mj-html-attribute"

	SectionTagName          MJMLComponentName = "mj-section"
	ColumnTagName           MJMLComponentName = "mj-column"
	WrapperTagName          MJMLComponentName = "mj-wrapper"
	GroupTagName            MJMLComponentName = "mj-group"
	SpacerTagName           MJMLComponentName = "mj-spacer"
	ImageTagName            MJMLComponentName = "mj-image"
	SocialTagName           MJMLComponentName = "mj-social"
	SocialElementTagName    MJMLComponentName = "mj-social-element"
	DividerTagName          MJMLComponentName = "mj-divider"
	TableTagName            MJMLComponentName = "mj-table"
	ButtonTagName           MJMLComponentName = "mj-button"
	HeroTagName             MJMLComponentName = "mj-hero"
	NavbarTagName           MJMLComponentName = "mj-navbar"
	NavbarLinkTagName       MJMLComponentName = "mj-navbar-link"
	AccordionTagName        MJMLComponentName = "mj-accordion"
	AccordionElementTagName MJMLComponentName = "mj-accordion-element"
	AccordionTitleTagName   MJMLComponentName = "mj-accordion-title"
	AccordionTextTagName    MJMLComponentName = "mj-accordion-text"
	CarouselTagName         MJMLComponentName = "mj-carousel"
	CarouselImageTagName    MJMLComponentName = "mj-carousel-image"
)
