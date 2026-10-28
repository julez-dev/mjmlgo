// Package customcomponent is the in-repo example of the custom-component
// contract in package component. It holds the two shapes a component takes:
// Badge, a leaf with typed attributes and defaults, and Notice, a container that
// wraps child elements and contributes CSS to the document head.
//
// A host registers the components on a renderer and gets new MJML elements, with
// no change to this repository:
//
//	renderer := mjmlgo.NewRenderer(mjmlgo.WithComponents(
//		customcomponent.Badge{},
//		customcomponent.Notice{},
//	))
//	html, err := renderer.RenderString(ctx, fsys, "newsletter.mjml")
//
//	<mjml>
//	  <mj-body>
//	    <mj-section>
//	      <mj-column>
//	        <mj-badge text="New" background-color="#1f2937" />
//	        <mj-notice>
//	          <mj-text>Your trial ends on Friday.</mj-text>
//	        </mj-notice>
//	      </mj-column>
//	    </mj-section>
//	  </mj-body>
//	</mjml>
//
// Both tags behave like built-in ones: an undeclared attribute is rejected, the
// declared validators run, an element under a leaf is rejected, and defaults
// from mj-attributes and mj-class apply before Render runs.
//
// Badge is the shape of a leaf component:
//
//   - Name returns the tag, which must not shadow a built-in element and must be
//     registered only once per renderer.
//   - AllowedAttributes declares every accepted attribute with its validator.
//     Only the universal css-class and mj-class need no entry.
//   - DefaultAttributes returns the values applied when the author wrote none.
//   - Render writes HTML from the node it is handed and the render context; it
//     reads attributes through node.Node, never through package internals.
//
// Notice is the shape of a container component:
//
//   - It implements component.ChildComponent and declares its children in
//     AllowedChildren. A child outside the list is a disallowed-child error with
//     the element path; a declared child nothing can render is rejected when the
//     component is registered.
//   - Its Render calls component.RenderChildren to render the children it
//     accepted - each with its own defaults applied and dispatched to its
//     built-in or custom component - into a buffer of the component's own, then
//     writes its wrapper markup around that HTML.
//   - It contributes head CSS with RenderContext.AddComponentHeadStyle (a named
//     block), AddMediaQuery (rules inside the document's breakpoint media query)
//     and AddFluidOnMobileClass (MJML's fluid-on-mobile rule for its own class).
package customcomponent
