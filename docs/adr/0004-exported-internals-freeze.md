---
status: accepted
---

# Export the element tree and render context, and freeze their fields

`internal/node` and `internal/component` became the public packages `node` and
`component` with their fields exactly as they were. The exported shape is now
API: custom components (ticket T3, issue #4) must receive the same node and
render context the built-in components receive, and re-exporting a projection of
them would cost a conversion on every render and a second type system to keep in
sync.

The fields below are frozen. Methods may be added; exported fields may not be
removed, renamed, retyped or given a new meaning without an ADR superseding this
one.

## `node.Node`

The parsed element tree: one node per element, with its attributes, its raw
content and its children.

| Field | Type | Meaning |
| --- | --- | --- |
| `Type` | `string` | Tag name of the element, e.g. `mj-section`. |
| `Attributes` | `[]xml.Attr` | The element's attributes in document order; rendering appends component defaults, so it can grow during a render. |
| `Content` | `string` | Unparsed text or markup of a raw-content element (`mj-text`, `mj-raw`, ...); empty for containers. |
| `Children` | `[]*Node` | Child elements in document order; text and raw content live in `Content`, not as children. |
| `Parent` | `*Node` | Parser bookkeeping pointer, nil on the root. The field is frozen, but it is not API: user code must not set it and components must not rely on it. |

## `component.RenderContext`

The state of one render: what the document head configured, what the renderer
passes down the tree, and what components contribute back to the head.

| Field | Type | Meaning |
| --- | --- | --- |
| `InlineStyles` | `[]Stylesheet` | Stylesheets to inject into the finished HTML (`<mj-style inline>`), in document order. |
| `Fonts` | `map[string]string` | Font family to import URL; seeded with the built-in registry unless the caller supplied one, extended by `<mj-font>`. |
| `FontOrder` | `[]string` | Font families in emission order; caller-supplied registries are sorted because a map has no order. |
| `FontsConfigured` | `bool` | Reports that `Fonts` came from the caller and must not be extended with the built-ins. |
| `BodyContent` | `string` | The rendered body HTML, used to detect which registered web fonts the document actually uses. |
| `CustomComponents` | `map[string]Component` | The custom components the renderer was configured with, keyed by tag name; nil when the renderer has none. The map belongs to the renderer and is shared by its renders, so component code must not modify it. |
| `AttributeDefaults` | `map[string]AttributeMap` | Per-tag defaults from `<mj-attributes>`, keyed by tag name; `mj-all` lives under the key `mj-all`. |
| `Classes` | `map[string]AttributeMap` | Attributes of each `<mj-class>`, keyed by class name. |
| `ClassDefaults` | `map[string]map[string]AttributeMap` | Per-tag defaults declared inside an `<mj-class>`, keyed by class name and then tag name. |
| `HTMLAttributes` | `[]HTMLAttributeSelector` | `<mj-html-attributes>` selectors in declaration order; repeated paths are merged in place. |
| `MJMLStylesheet` | `map[string][]string` | Generated component CSS, keyed by the class name it is registered under. |
| `MJMLStyleOrder` | `[]string` | Registered class names in first-registration order, so the emitted CSS is deterministic. |
| `ComponentHeadStyles` | `[]ComponentHeadStyle` | Named component stylesheets (navbar, accordion, ...), deduplicated by component name. |
| `InstanceComponentHeadStyles` | `[]ComponentHeadStyle` | Per-instance component stylesheets (carousel), never deduplicated, emitted before the named ones. |
| `IDGenerator` | `func() string` | Produces the random ids for navbar hamburger labels and carousels; nil falls back to `DefaultIDGenerator`. |
| `IncludeMobileFullWidthStyle` | `bool` | The document contains an `mj-image`, so the mobile-full-width rule is emitted. |
| `ForceOWADesktop` | `bool` | `<mjml owa="desktop">` was set, so the conditional CSS targets desktop Outlook. |
| `PrinterSupport` | `bool` | Repeats the generated media queries inside `@media only print`, like MJML's `printerSupport`. |
| `ContainerWidth` | `string` | Width content may occupy at the current nesting depth, e.g. `600px`; containers narrow it and restore it. |
| `PreviewText` | `string` | Text of `<mj-preview>`, emitted as the hidden preheader of the body. |
| `Title` | `string` | Text of `<mj-title>`, emitted as the `aria-label` of the body div. |
| `Breakpoint` | `string` | Media-query breakpoint, e.g. `480px`; `<mj-breakpoint>` overrides the default. |
| `Language` | `string` | Document language from `<mjml lang>`, emitted on the body. |
| `Direction` | `string` | Document direction from `<mjml dir>`, emitted on the body. |

The two unexported fields (`accordionFontFamily`, `elementFontFamily`) hold the
contextual font inheritance of the accordion currently being rendered. They are
not frozen and may change freely.

## Types the frozen fields expose

A frozen field freezes its type with it, so these types keep their exported
fields too: `AttributeMap` (`map[string]string`), `Stylesheet` (`Rules`,
`MediaRules`), `MediaRule` (`Condition`, `Rules`), `Rule` (`Selectors`,
`Declarations`), `Style` (`Property`, `Value`, `Important`),
`HTMLAttributeSelector` (`Path`, `Attributes`), `HTMLAttribute` (`Name`,
`Value`) and `ComponentHeadStyle` (`Name`, `Style`).

## Why exporting beat a facade

- The renderer already walks this exact tree, and components read exactly these
  fields, including ones a facade would have to hide: `Attributes` carries
  component defaults and the synthetic `mobileWidth` marker, `Content` carries
  raw markup, and the render context is mutated in place while the tree is
  walked. A projection would have to copy it per render - the allocation cost is
  the whole point of the budget table - or wrap it in accessors that return the
  same values with more indirection.
- Two parallel type systems drift. Parity work adjusts component internals every
  week; every adjustment would need a matching change to the public projection,
  and the two would eventually disagree about what a component may touch.
- Custom components have to round-trip nodes: a container passes its children to
  the renderer and expects defaults to have been applied in place. Converting at
  that boundary would either lose identity or force the conversion to be
  lossless, which is exactly what sharing the type gives for free.
- Keeping everything `internal/` and re-exporting from the root package would
  make the root package the union of every type the message pipeline touches,
  and still leak the same fields through the exported signatures.

## What may still change

- Methods may be added to `Node` and `RenderContext`; the existing methods keep
  their signatures.
- Unexported fields may change, be added or be removed.
- New exported fields may be added to `RenderContext` when they are additive and
  documented here; user code constructs it with keyed literals, so adding a field
  does not break compilation.
- The concrete components (`MJML`, `MJMLSection`, ...), the raw-block markers,
  the error sentinels and package `node`'s `Parent` semantics are not frozen by
  this ADR.
- `InitComponent` and `ValidateDocument` are API to call, but their signatures
  are not frozen: validation became custom-component aware in T3 (issue #4) and
  returns the structured failure of T5 (issue #6), and the package documentation
  says so.
- The `Component` interface is not frozen. T3 (issue #4) made it the
  custom-component contract by replacing its unexported validator type with the
  exported `AttributeValidator`, and added `ChildComponent` for declared child
  lists; container support (T4) may extend it further.

## Compatibility promise

These exported fields are API. From the first tagged release onwards, a minor
release will not remove, rename, retype or change the meaning of any field
listed above, and will not change the exported fields of the types those fields
expose. A breaking change requires a new ADR that supersedes this one and a major
version bump. Adding methods or new documented `RenderContext` fields is not a
breaking change. No release has been tagged yet, so this promise starts at the
first tag; until then the freeze still governs reviews.

## Known warts, recorded instead of fixed

This ticket freezes the shape and forbids fixing it, so the warts are written
down here for the next revision rather than corrected now:

- `node.Node.Attributes` uses `[]xml.Attr`, which puts `encoding/xml` in the
  public API. `Attributes` and `Parent` also carry `json:"-"`, so `encoding/json`
  silently skips both.
- `Rule` has unexported fields (the compiled-selector cache) and
  `RenderContext` has unexported fields, so callers outside the package cannot
  use unkeyed literals for them and cannot set the contextual accordion font
  families (a hand-built context simply inherits nothing). A hand-built `Rule`
  has no compiled selector until `CompiledSelectors` compiles it on first use;
  that lazy path exists for exactly this case.
- `Node.Parent` is exported because the parser needs it, not because callers
  should use it; the package documentation says so, but the field cannot be
  unexported without changing the shape this ADR freezes.
- The `Component` interface is implementable outside the package since T3
  (issue #4), but a custom tag is only accepted in the content containers
  (mj-body, mj-column, mj-hero) and children of a custom component are rendered
  by that component itself (see above).
- `node.Node.Content` holds raw markup for raw-content elements and plain text
  elsewhere; there is no flag that tells the two apart, so a consumer has to know
  the element's type to interpret it.
