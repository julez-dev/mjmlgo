package component

type MJMLComponentName = string

const (
	MJMLTagName MJMLComponentName = "mjml"
	HeadTagName MJMLComponentName = "mj-head"
	BodyTagName MJMLComponentName = "mj-body"

	// head specific
	AttibutesTagName MJMLComponentName = "mj-attributes"
	TextTagName      MJMLComponentName = "mj-text"
	ClassTagName     MJMLComponentName = "mj-class"
	AllTagName       MJMLComponentName = "mj-all"
	TitleTagName     MJMLComponentName = "mj-title"
	PreviewTagName   MJMLComponentName = "mj-preview"
)
