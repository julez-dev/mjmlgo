// Package customcomponent is the in-repo example of the custom-component
// contract in package component. It is deliberately small: a leaf component
// with two typed attributes, defaults, and no children.
//
// A host registers Badge on a renderer and gets a new MJML element, with no
// change to this repository:
//
//	renderer := mjmlgo.NewRenderer(mjmlgo.WithComponents(customcomponent.Badge{}))
//	html, err := renderer.RenderString(ctx, fsys, "newsletter.mjml")
//
//	<mjml>
//	  <mj-body>
//	    <mj-section>
//	      <mj-column>
//	        <mj-badge text="New" background-color="#1f2937" />
//	      </mj-column>
//	    </mj-section>
//	  </mj-body>
//	</mjml>
//
// The tag behaves like a built-in one: an undeclared attribute is rejected, the
// declared validators run, an element under it is rejected because Badge is a
// leaf, and defaults from mj-attributes and mj-class apply before Render runs.
//
// Badge is the shape a real component takes:
//
//   - Name returns the tag, which must not shadow a built-in element and must be
//     registered only once per renderer.
//   - AllowedAttributes declares every accepted attribute with its validator.
//     Only the universal css-class and mj-class need no entry.
//   - DefaultAttributes returns the values applied when the author wrote none.
//   - Render writes HTML from the node it is handed and the render context; it
//     reads attributes through node.Node, never through package internals.
//
// A component that accepts children implements component.ChildComponent and
// declares them in AllowedChildren. Container rendering is ticket T4 (issue #5);
// a ChildComponent's Render is responsible for its children.
package customcomponent
