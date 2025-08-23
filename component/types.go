package component

type MJMLComponentName = string

const (
	MJMLTagName MJMLComponentName = "mjml"
	HeadTagName MJMLComponentName = "mj-head"
	BodyTagName MJMLComponentName = "mj-body"
	RawTagName  MJMLComponentName = "mj-raw"

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

	SectionTagName       MJMLComponentName = "mj-section"
	ColumnTagName        MJMLComponentName = "mj-column"
	WrapperTagName       MJMLComponentName = "mj-wrapper"
	GroupTagName         MJMLComponentName = "mj-group"
	SpacerTagName        MJMLComponentName = "mj-spacer"
	ImageTagName         MJMLComponentName = "mj-image"
	SocialTagName        MJMLComponentName = "mj-social"
	SocialElementTagName MJMLComponentName = "mj-social-element"
	DividerTagName       MJMLComponentName = "mj-divider"
	TableTagName         MJMLComponentName = "mj-table"
)
