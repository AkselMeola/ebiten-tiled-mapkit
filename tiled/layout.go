package tiled

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
)

// go-tiled keeps each layer kind in its own slice, so the interleaved order is read from the XML.
type layout struct {
	infinite bool
	layers   []layoutNode
}

type layoutNode struct {
	kind     string
	children []layoutNode
}

func isLayerElement(name string) bool {
	switch name {
	case "layer", "objectgroup", "imagelayer", "group":
		return true
	}
	return false
}

func scanLayout(data []byte) (layout, error) {
	var lay layout
	dec := xml.NewDecoder(bytes.NewReader(data))

	root := &layoutNode{}
	stack := []*layoutNode{root}
	var names []string

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return lay, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local
			parent := ""
			if len(names) > 0 {
				parent = names[len(names)-1]
			}
			names = append(names, name)

			if name == "map" && parent == "" {
				for _, a := range t.Attr {
					if a.Name.Local == "infinite" && a.Value == "1" {
						lay.infinite = true
					}
				}
			}

			node := stack[len(stack)-1]
			if isLayerElement(name) && (parent == "map" || parent == "group") {
				node.children = append(node.children, layoutNode{kind: name})
				node = &node.children[len(node.children)-1]
			}
			stack = append(stack, node)
		case xml.EndElement:
			names = names[:len(names)-1]
			stack = stack[:len(stack)-1]
		}
	}

	lay.layers = root.children
	return lay, nil
}
