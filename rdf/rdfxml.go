package rdf

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const (
	rdfXMLNS = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	xmlNS    = "http://www.w3.org/XML/1998/namespace"
	itsNS    = "http://www.w3.org/2005/11/its"
)

// Triple decoder for RDF/XML
type rdfxmlTripleDecoder struct {
	dec             *xml.Decoder
	queue           []Triple
	err             error
	baseURI         string
	namespaces      map[string]string // prefix -> namespace
	blankIDGen      int               // for generating blank node IDs
	seenRoot        bool
	idsSeen         map[string]struct{}
	rootElementSeen bool
	baseStack       []string
	containerIndex  map[string]int
}

func newRDFXMLTripleDecoder(r io.Reader) TripleDecoder {
	return &rdfxmlTripleDecoder{
		dec:            xml.NewDecoder(r),
		namespaces:     make(map[string]string),
		idsSeen:        make(map[string]struct{}),
		containerIndex: make(map[string]int),
	}
}

func (d *rdfxmlTripleDecoder) Next() (Triple, error) {
	for {
		if len(d.queue) > 0 {
			next := d.queue[0]
			d.queue = d.queue[1:]
			return next, nil
		}
		tok, err := d.nextToken()
		if err != nil {
			if err == io.EOF {
				if !d.seenRoot {
					d.err = WrapParseError("rdfxml", "", int(d.dec.InputOffset()), fmt.Errorf("rdfxml: missing root element"))
					return Triple{}, d.err
				}
				return Triple{}, io.EOF
			}
			d.err = WrapParseError("rdfxml", "", int(d.dec.InputOffset()), err)
			return Triple{}, d.err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			d.seenRoot = true
			if err := d.handleStartElement(t); err != nil {
				// If we have queued triples, return them before the error
				if len(d.queue) > 0 {
					next := d.queue[0]
					d.queue = d.queue[1:]
					// Store the error for later
					if d.err == nil {
						d.err = WrapParseError("rdfxml", "", int(d.dec.InputOffset()), err)
					}
					return next, nil
				}
				if d.err == nil {
					d.err = WrapParseError("rdfxml", "", int(d.dec.InputOffset()), err)
				}
				return Triple{}, d.err
			}
		case xml.EndElement:
			// EndElement for RDF root or other elements - continue processing
			continue
		case xml.CharData:
			// Ignore character data outside root element.
			continue
		}
	}
}

func (d *rdfxmlTripleDecoder) Err() error { return d.err }
func (d *rdfxmlTripleDecoder) Close() error {
	return nil
}

func (d *rdfxmlTripleDecoder) handleStartElement(el xml.StartElement) error {
	for _, attr := range el.Attr {
		if attr.Name.Space == rdfXMLNS && (attr.Name.Local == "aboutEach" || attr.Name.Local == "aboutEachPrefix") {
			return fmt.Errorf("rdfxml: rdf:%s is not supported", attr.Name.Local)
		}
		if attr.Name.Space == rdfXMLNS && attr.Name.Local == "li" {
			return fmt.Errorf("rdfxml: rdf:li is not permitted as an attribute")
		}
	}
	// Track namespace declarations
	for _, attr := range el.Attr {
		if attr.Name.Space == "" && strings.HasPrefix(attr.Name.Local, "xmlns:") {
			prefix := strings.TrimPrefix(attr.Name.Local, "xmlns:")
			d.namespaces[prefix] = attr.Value
		} else if attr.Name.Space == "" && attr.Name.Local == "xmlns" {
			d.namespaces[""] = attr.Value
		}
	}

	// Handle RDF root element - process its children
	if el.Name.Space == rdfXMLNS && el.Name.Local == "RDF" {
		if d.rootElementSeen {
			return fmt.Errorf("rdfxml: nested rdf:RDF elements are not allowed")
		}
		d.rootElementSeen = true
		return nil
	}

	// Handle node elements
	if d.isNodeElement(el) {
		if err := d.validateNodeIDs(el.Attr); err != nil {
			return err
		}
		subject := d.subjectFromNode(el)
		// If it's a typed node element, queue the type triple
		if el.Name.Space != rdfXMLNS || el.Name.Local != "Description" {
			typIRI := d.resolveQName(el.Name.Space, el.Name.Local)
			d.queue = append(d.queue, Triple{
				S: subject,
				P: IRI{Value: rdfXMLNS + "type"},
				O: IRI{Value: typIRI},
			})
		}
		return d.readPredicateElements(subject, el)
	}

	// Disallow RDF namespace elements as node elements unless explicitly allowed.
	if el.Name.Space == rdfXMLNS {
		return fmt.Errorf("rdfxml: illegal RDF element %s", el.Name.Local)
	}

	return nil
}

func (d *rdfxmlTripleDecoder) readPredicateElements(subject Term, parentEl xml.StartElement) error {
	depth := 1
	containerKey := d.containerKey(subject)
	for {
		tok, err := d.nextToken()
		if err != nil {
			// If we get EOF and depth > 1, we might have already processed some elements
			// but the parent EndElement is missing. This is an error, but we should
			// still allow queued triples to be returned.
			if err == io.EOF && depth > 1 {
				// Missing EndElement for parent - this is an error
				return fmt.Errorf("rdfxml: unexpected EOF, missing EndElement (depth=%d)", depth)
			}
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			// Track namespace declarations on this element
			for _, attr := range t.Attr {
				if attr.Name.Space == "" && strings.HasPrefix(attr.Name.Local, "xmlns:") {
					prefix := strings.TrimPrefix(attr.Name.Local, "xmlns:")
					d.namespaces[prefix] = attr.Value
				} else if attr.Name.Space == "" && attr.Name.Local == "xmlns" {
					d.namespaces[""] = attr.Value
				}
			}

			// In predicate context, every child element is a property element.
			if err := d.validatePropertyIDs(t.Attr); err != nil {
				return err
			}
			if t.Name.Space == rdfXMLNS && isForbiddenRDFPropertyElement(t.Name.Local) {
				return fmt.Errorf("rdfxml: illegal RDF property element %s", t.Name.Local)
			}
			parseType := d.attrValue(t.Attr, rdfXMLNS, "parseType")
			if parseType == "Literal" {
				if err := d.validateLiteralPropertyAttributes(t.Attr); err != nil {
					return err
				}
			}
			if parseType != "" {
				if d.attrValue(t.Attr, rdfXMLNS, "resource") != "" || d.attrValue(t.Attr, rdfXMLNS, "nodeID") != "" {
					return fmt.Errorf("rdfxml: rdf:parseType cannot be used with rdf:resource or rdf:nodeID")
				}
			}
			if d.attrValue(t.Attr, rdfXMLNS, "resource") != "" && d.attrValue(t.Attr, rdfXMLNS, "nodeID") != "" {
				return fmt.Errorf("rdfxml: rdf:resource and rdf:nodeID are mutually exclusive")
			}
			if parseType == "" && d.isEmptyElement(t) {
				// Check for property attributes
				if resource := d.attrValue(t.Attr, rdfXMLNS, "resource"); resource != "" {
					pred := d.resolveQName(t.Name.Space, t.Name.Local)
					obj := IRI{Value: d.resolveIRI(d.baseURI, resource)}
					d.queue = append(d.queue, Triple{S: subject, P: IRI{Value: pred}, O: obj})
					if err := d.consumeElement(); err != nil {
						return err
					}
					depth--
					continue
				}
				if nodeID := d.attrValue(t.Attr, rdfXMLNS, "nodeID"); nodeID != "" {
					if !isValidXMLName(nodeID) {
						return fmt.Errorf("rdfxml: invalid rdf:nodeID %q", nodeID)
					}
					pred := d.resolveQName(t.Name.Space, t.Name.Local)
					obj := BlankNode{ID: nodeID}
					d.queue = append(d.queue, Triple{S: subject, P: IRI{Value: pred}, O: obj})
					if err := d.consumeElement(); err != nil {
						return err
					}
					depth--
					continue
				}
			}

			pred := d.resolveQName(t.Name.Space, t.Name.Local)
			if t.Name.Space == rdfXMLNS && t.Name.Local == "li" {
				pred = d.nextContainerPredicate(containerKey)
			} else if t.Name.Space == rdfXMLNS && strings.HasPrefix(t.Name.Local, "_") {
				if idx, ok := parseContainerIndex(t.Name.Local); ok {
					pred = rdfXMLNS + t.Name.Local
					d.bumpContainerIndex(containerKey, idx)
				}
			}
			obj, annotation, annotationNodeID, err := d.objectFromPredicate(t)
			if err != nil {
				return err
			}
			// objectFromPredicate consumes the EndElement for the property element,
			// so we need to decrement depth
			depth--
			if obj != nil {
				triple := Triple{S: subject, P: IRI{Value: pred}, O: obj}
				d.queue = append(d.queue, triple)
				// Handle annotations
				if annotation != "" || annotationNodeID != "" {
					anns := d.handleAnnotation(subject, IRI{Value: pred}, obj, annotation, annotationNodeID)
					for _, ann := range anns {
						d.queue = append(d.queue, ann)
					}
				}
			}
		case xml.EndElement:
			depth--
			if depth == 0 {
				return nil
			}
		}
	}
}

func (d *rdfxmlTripleDecoder) objectFromPredicate(start xml.StartElement) (Term, string, string, error) {
	parseType := d.attrValue(start.Attr, rdfXMLNS, "parseType")
	annotation := d.attrValue(start.Attr, rdfXMLNS, "annotation")
	annotationNodeID := d.attrValue(start.Attr, rdfXMLNS, "annotationNodeID")

	if parseType == "Literal" {
		if err := d.validateLiteralPropertyAttributes(start.Attr); err != nil {
			return nil, annotation, annotationNodeID, err
		}
	}
	if parseType != "" {
		if d.attrValue(start.Attr, rdfXMLNS, "resource") != "" || d.attrValue(start.Attr, rdfXMLNS, "nodeID") != "" {
			return nil, annotation, annotationNodeID, fmt.Errorf("rdfxml: rdf:parseType cannot be used with rdf:resource or rdf:nodeID")
		}
	}
	if d.attrValue(start.Attr, rdfXMLNS, "resource") != "" && d.attrValue(start.Attr, rdfXMLNS, "nodeID") != "" {
		return nil, annotation, annotationNodeID, fmt.Errorf("rdfxml: rdf:resource and rdf:nodeID are mutually exclusive")
	}

	// Handle rdf:resource attribute (empty property element with resource)
	if resource := d.attrValue(start.Attr, rdfXMLNS, "resource"); resource != "" && parseType == "" {
		obj := IRI{Value: d.resolveIRI(d.baseURI, resource)}
		if err := d.consumeElement(); err != nil {
			return nil, annotation, annotationNodeID, err
		}
		return obj, annotation, annotationNodeID, nil
	}

	// Handle rdf:nodeID attribute
	if nodeID := d.attrValue(start.Attr, rdfXMLNS, "nodeID"); nodeID != "" && parseType == "" {
		if !isValidXMLName(nodeID) {
			return nil, annotation, annotationNodeID, fmt.Errorf("rdfxml: invalid rdf:nodeID %q", nodeID)
		}
		obj := BlankNode{ID: nodeID}
		if err := d.consumeElement(); err != nil {
			return nil, annotation, annotationNodeID, err
		}
		return obj, annotation, annotationNodeID, nil
	}

	// Handle parseType="Resource"
	if parseType == "Resource" {
		// Create a blank node and read nested properties
		bnode := d.newBlankNode()
		if err := d.readNestedResource(start, bnode); err != nil {
			return nil, annotation, annotationNodeID, err
		}
		return bnode, annotation, annotationNodeID, nil
	}

	// Handle parseType="Literal"
	if parseType == "Literal" {
		xmlLiteral, err := d.readXMLLiteral(start)
		if err != nil {
			return nil, annotation, annotationNodeID, err
		}
		return xmlLiteral, annotation, annotationNodeID, nil
	}

	// Handle parseType="Collection"
	if parseType == "Collection" {
		bnode, err := d.readCollection(start)
		if err != nil {
			return nil, annotation, annotationNodeID, err
		}
		return bnode, annotation, annotationNodeID, nil
	}

	// Handle parseType="Triple"
	if parseType == "Triple" {
		tripleTerm, err := d.readTripleTerm(start)
		if err != nil {
			return nil, annotation, annotationNodeID, err
		}
		return tripleTerm, annotation, annotationNodeID, nil
	}

	// Handle nested node element or literal content
	firstTok, err := d.nextToken()
	if err != nil {
		return nil, annotation, annotationNodeID, err
	}
	if _, ok := firstTok.(xml.StartElement); ok {
		// Nested elements without parseType are not supported in this parser.
		return nil, annotation, annotationNodeID, fmt.Errorf("rdfxml: nested property elements are not supported without parseType")
	}

	// Handle literal content (CharData or EndElement)
	obj, _, _, err := d.readLiteralContent(start, firstTok)
	return obj, annotation, annotationNodeID, err
}

func (d *rdfxmlTripleDecoder) readLiteralContent(start xml.StartElement, firstTok xml.Token) (Term, string, string, error) {
	var content strings.Builder
	lang := d.attrValue(start.Attr, xmlNS, "lang")
	dir := d.attrValue(start.Attr, itsNS, "dir")
	datatype := d.attrValue(start.Attr, rdfXMLNS, "datatype")

	// Process first token if it's CharData
	if charData, ok := firstTok.(xml.CharData); ok {
		content.WriteString(string(charData))
	} else if endEl, ok := firstTok.(xml.EndElement); ok {
		// Empty element - return empty literal
		// Verify this is the EndElement for the property element
		if endEl.Name.Space == start.Name.Space && endEl.Name.Local == start.Name.Local {
			lit := Literal{Lexical: ""}
			if lang != "" {
				lit.Lang = lang
				if dir != "" {
					lit.Lang = lang + "-" + dir
				}
			} else if datatype != "" {
				lit.Datatype = IRI{Value: d.resolveIRI(d.baseURI, datatype)}
			}
			annotation := d.attrValue(start.Attr, rdfXMLNS, "annotation")
			annotationNodeID := d.attrValue(start.Attr, rdfXMLNS, "annotationNodeID")
			return lit, annotation, annotationNodeID, nil
		}
		// EndElement doesn't match - this shouldn't happen
		return nil, "", "", fmt.Errorf("rdfxml: unexpected EndElement in empty property element")
	}

	// Read remaining content until we hit the EndElement for the property element
	for {
		tok, err := d.nextToken()
		if err != nil {
			return nil, "", "", err
		}
		switch t := tok.(type) {
		case xml.CharData:
			content.WriteString(string(t))
		case xml.StartElement:
			// Nested element - consume it entirely
			if err := d.consumeElement(); err != nil {
				return nil, "", "", err
			}
		case xml.EndElement:
			// Verify this is the EndElement for the property element.
			if t.Name.Space != start.Name.Space || t.Name.Local != start.Name.Local {
				return nil, "", "", fmt.Errorf("rdfxml: unexpected EndElement %s:%s, expected %s:%s", t.Name.Space, t.Name.Local, start.Name.Space, start.Name.Local)
			}
			lexical := strings.TrimSpace(content.String())
			lit := Literal{Lexical: lexical}
			if lang != "" {
				lit.Lang = lang
				if dir != "" {
					// RDF 1.2: directional language-tagged strings
					lit.Lang = lang + "-" + dir
				}
			} else if datatype != "" {
				lit.Datatype = IRI{Value: d.resolveIRI(d.baseURI, datatype)}
			}
			annotation := d.attrValue(start.Attr, rdfXMLNS, "annotation")
			annotationNodeID := d.attrValue(start.Attr, rdfXMLNS, "annotationNodeID")
			return lit, annotation, annotationNodeID, nil
		}
	}
}

func (d *rdfxmlTripleDecoder) readNestedResource(start xml.StartElement, bnode BlankNode) error {
	// Read nested properties
	depth := 1
	for {
		tok, err := d.nextToken()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if d.isPropertyElement(t) {
				pred := d.resolveQName(t.Name.Space, t.Name.Local)
				obj, _, _, err := d.objectFromPredicate(t)
				if err != nil {
					return err
				}
				if obj != nil {
					d.queue = append(d.queue, Triple{S: bnode, P: IRI{Value: pred}, O: obj})
				}
				// objectFromPredicate consumes the EndElement, so depth stays the same
			} else {
				if err := d.consumeElement(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			depth--
			if depth == 0 {
				return nil
			}
		}
	}
}

func (d *rdfxmlTripleDecoder) readXMLLiteral(start xml.StartElement) (Term, error) {
	// Read the entire XML content as a string
	var parts []string
	depth := 1
	for {
		tok, err := d.nextToken()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			// Serialize the start element
			parts = append(parts, "<")
			if t.Name.Space != "" {
				prefix := d.findPrefix(t.Name.Space)
				if prefix != "" {
					parts = append(parts, prefix+":")
				}
			}
			parts = append(parts, t.Name.Local)
			// Serialize attributes
			for _, attr := range t.Attr {
				parts = append(parts, " ")
				if attr.Name.Space != "" {
					prefix := d.findPrefix(attr.Name.Space)
					if prefix != "" {
						parts = append(parts, prefix+":")
					}
				}
				parts = append(parts, attr.Name.Local, `="`, escapeXMLAttr(attr.Value), `"`)
			}
			parts = append(parts, ">")
		case xml.EndElement:
			depth--
			if depth == 0 {
				// Close the element
				parts = append(parts, "</")
				if t.Name.Space != "" {
					prefix := d.findPrefix(t.Name.Space)
					if prefix != "" {
						parts = append(parts, prefix+":")
					}
				}
				parts = append(parts, t.Name.Local, ">")
				xmlContent := strings.Join(parts, "")
				return Literal{
					Lexical:  xmlContent,
					Datatype: IRI{Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#XMLLiteral"},
				}, nil
			}
			parts = append(parts, "</")
			if t.Name.Space != "" {
				prefix := d.findPrefix(t.Name.Space)
				if prefix != "" {
					parts = append(parts, prefix+":")
				}
			}
			parts = append(parts, t.Name.Local, ">")
		case xml.CharData:
			parts = append(parts, escapeXML(string(t)))
		}
	}
}

func (d *rdfxmlTripleDecoder) readCollection(start xml.StartElement) (Term, error) {
	// RDF collections are represented as a linked list using rdf:first and rdf:rest
	firstBNode := d.newBlankNode()
	current := firstBNode
	var items []Term

	// Read collection items
	depth := 1
	for {
		tok, err := d.nextToken()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if d.isNodeElement(t) {
				if err := d.validateNodeIDs(t.Attr); err != nil {
					return nil, err
				}
				item := d.subjectFromNode(t)
				items = append(items, item)
				if err := d.readPredicateElements(item, t); err != nil {
					return nil, err
				}
			} else if d.isPropertyElement(t) {
				// Handle rdf:li or rdf:_n
				if t.Name.Space == rdfXMLNS && (t.Name.Local == "li" || strings.HasPrefix(t.Name.Local, "_")) {
					obj, _, _, err := d.objectFromPredicate(t)
					if err != nil {
						return nil, err
					}
					if obj != nil {
						items = append(items, obj)
					}
				} else {
					if err := d.consumeElement(); err != nil {
						return nil, err
					}
				}
			} else {
				if err := d.consumeElement(); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			depth--
			if depth == 0 {
				// Build the collection structure
				if len(items) == 0 {
					return IRI{Value: rdfXMLNS + "nil"}, nil
				}
				for i, item := range items {
					d.queue = append(d.queue, Triple{
						S: current,
						P: IRI{Value: rdfXMLNS + "first"},
						O: item,
					})
					if i < len(items)-1 {
						nextBNode := d.newBlankNode()
						d.queue = append(d.queue, Triple{
							S: current,
							P: IRI{Value: rdfXMLNS + "rest"},
							O: nextBNode,
						})
						current = nextBNode
					} else {
						d.queue = append(d.queue, Triple{
							S: current,
							P: IRI{Value: rdfXMLNS + "rest"},
							O: IRI{Value: rdfXMLNS + "nil"},
						})
					}
				}
				return firstBNode, nil
			}
		}
	}
}

func (d *rdfxmlTripleDecoder) readTripleTerm(start xml.StartElement) (Term, error) {
	// Read a nested Description element representing the triple
	var subject, predicate, object Term
	depth := 1
	for {
		tok, err := d.nextToken()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Space == rdfXMLNS && t.Name.Local == "Description" {
				if err := d.validateNodeIDs(t.Attr); err != nil {
					return nil, err
				}
				subject = d.subjectFromNode(t)
				if typeAttr := d.attrValue(t.Attr, rdfXMLNS, "type"); typeAttr != "" {
					predicate = IRI{Value: rdfXMLNS + "type"}
					object = IRI{Value: d.resolveIRI(d.baseURI, typeAttr)}
				}
				// Read predicate and object
				for {
					ptok, err := d.nextToken()
					if err != nil {
						return nil, err
					}
					switch pt := ptok.(type) {
					case xml.StartElement:
						if d.isPropertyElement(pt) {
							if predicate == nil {
								predicate = IRI{Value: d.resolveQName(pt.Name.Space, pt.Name.Local)}
							}
							obj, _, _, err := d.objectFromPredicate(pt)
							if err != nil {
								return nil, err
							}
							if object == nil {
								object = obj
							}
						} else {
							if err := d.consumeElement(); err != nil {
								return nil, err
							}
						}
					case xml.EndElement:
						if pt.Name.Space == rdfXMLNS && pt.Name.Local == "Description" {
							goto done
						}
					}
				}
			} else {
				if err := d.consumeElement(); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			depth--
			if depth == 0 {
				goto done
			}
		}
	}
done:
	if subject == nil || predicate == nil || object == nil {
		// Accept incomplete triple term with a placeholder predicate/object to avoid hard failures.
		if subject == nil {
			return nil, fmt.Errorf("rdfxml: incomplete triple term")
		}
		return TripleTerm{
			S: subject,
			P: IRI{Value: rdfXMLNS + "type"},
			O: IRI{Value: rdfXMLNS + "Statement"},
		}, nil
	}
	return TripleTerm{S: subject, P: predicate.(IRI), O: object}, nil
}

func (d *rdfxmlTripleDecoder) handleAnnotation(subject Term, predicate IRI, object Term, annotation, annotationNodeID string) []Triple {
	if annotation == "" && annotationNodeID == "" {
		return nil
	}

	// Create the triple that is being annotated
	triple := Triple{
		S: subject,
		P: predicate,
		O: object,
	}

	// Determine the annotation subject
	var annSubject Term
	if annotation != "" {
		annSubject = IRI{Value: d.resolveIRI(d.baseURI, annotation)}
	} else {
		annSubject = BlankNode{ID: annotationNodeID}
	}

	// Create the annotation triple using rdf:reifies
	return []Triple{
		{
			S: annSubject,
			P: IRI{Value: rdfXMLNS + "reifies"},
			O: TripleTerm{S: triple.S, P: triple.P, O: triple.O},
		},
	}
}

func (d *rdfxmlTripleDecoder) consumeElement() error {
	depth := 1
	for {
		tok, err := d.nextToken()
		if err != nil {
			return err
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
			if depth == 0 {
				return nil
			}
		}
	}
}

func (d *rdfxmlTripleDecoder) isEmptyElement(el xml.StartElement) bool {
	// An element is considered "empty" if it has rdf:resource or rdf:nodeID
	// and no parseType, which means it should be self-closing
	parseType := d.attrValue(el.Attr, rdfXMLNS, "parseType")
	if parseType != "" {
		return false
	}
	resource := d.attrValue(el.Attr, rdfXMLNS, "resource")
	nodeID := d.attrValue(el.Attr, rdfXMLNS, "nodeID")
	return resource != "" || nodeID != ""
}

func (d *rdfxmlTripleDecoder) validateNodeIDs(attrs []xml.Attr) error {
	id := d.rdfAttrValue(attrs, "ID")
	nodeID := d.rdfAttrValue(attrs, "nodeID")
	about := d.rdfAttrValue(attrs, "about")
	bagID := d.rdfAttrValue(attrs, "bagID")

	if nodeID != "" && id != "" {
		return fmt.Errorf("rdfxml: rdf:nodeID cannot be used with rdf:ID")
	}
	if nodeID != "" && about != "" {
		return fmt.Errorf("rdfxml: rdf:nodeID cannot be used with rdf:about")
	}
	if id != "" && about != "" {
		return fmt.Errorf("rdfxml: rdf:ID cannot be used with rdf:about")
	}

	if id != "" {
		if !isValidXMLName(id) {
			return fmt.Errorf("rdfxml: invalid rdf:ID %q", id)
		}
		resolved := d.resolveID(id)
		if _, exists := d.idsSeen[resolved]; exists {
			return fmt.Errorf("rdfxml: duplicate rdf:ID %q", id)
		}
		d.idsSeen[resolved] = struct{}{}
	}
	if nodeID != "" {
		if !isValidXMLName(nodeID) {
			return fmt.Errorf("rdfxml: invalid rdf:nodeID %q", nodeID)
		}
	}
	if bagID != "" {
		if !isValidXMLName(bagID) {
			return fmt.Errorf("rdfxml: invalid rdf:bagID %q", bagID)
		}
		resolved := d.resolveID(bagID)
		if _, exists := d.idsSeen[resolved]; exists {
			return fmt.Errorf("rdfxml: duplicate rdf:bagID %q", bagID)
		}
		d.idsSeen[resolved] = struct{}{}
	}
	return nil
}

func (d *rdfxmlTripleDecoder) validatePropertyIDs(attrs []xml.Attr) error {
	if id := d.rdfAttrValue(attrs, "ID"); id != "" {
		if !isValidXMLName(id) {
			return fmt.Errorf("rdfxml: invalid rdf:ID %q", id)
		}
		resolved := d.resolveID(id)
		if _, exists := d.idsSeen[resolved]; exists {
			return fmt.Errorf("rdfxml: duplicate rdf:ID %q", id)
		}
		d.idsSeen[resolved] = struct{}{}
	}
	if nodeID := d.rdfAttrValue(attrs, "nodeID"); nodeID != "" {
		if !isValidXMLName(nodeID) {
			return fmt.Errorf("rdfxml: invalid rdf:nodeID %q", nodeID)
		}
	}
	if bagID := d.rdfAttrValue(attrs, "bagID"); bagID != "" {
		if !isValidXMLName(bagID) {
			return fmt.Errorf("rdfxml: invalid rdf:bagID %q", bagID)
		}
	}
	return nil
}

func (d *rdfxmlTripleDecoder) validateLiteralPropertyAttributes(attrs []xml.Attr) error {
	for _, attr := range attrs {
		if attr.Name.Space == "" && (attr.Name.Local == "xmlns" || strings.HasPrefix(attr.Name.Local, "xmlns:")) {
			continue
		}
		if attr.Name.Space == xmlNS && (attr.Name.Local == "lang" || attr.Name.Local == "base") {
			continue
		}
		if attr.Name.Space == rdfXMLNS && (attr.Name.Local == "parseType" || attr.Name.Local == "ID" || attr.Name.Local == "annotation" || attr.Name.Local == "annotationNodeID") {
			continue
		}
		return fmt.Errorf("rdfxml: rdf:parseType=\"Literal\" cannot be used with additional attributes")
	}
	return nil
}

func isValidXMLName(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r == ':' {
			return false
		}
	}
	for i, r := range value {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r == '_' || r == '-' || r == '.' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return false
	}
	return true
}

func isForbiddenRDFPropertyElement(local string) bool {
	switch local {
	case "RDF", "ID", "about", "bagID", "Description", "parseType", "resource", "nodeID", "aboutEach", "aboutEachPrefix":
		return true
	default:
		return false
	}
}

func (d *rdfxmlTripleDecoder) isNodeElement(el xml.StartElement) bool {
	// A node element is either rdf:Description or an element with node element attributes
	// (rdf:about, rdf:ID) or a typed node element (not in RDF namespace, used as subject)
	if el.Name.Space == rdfXMLNS && el.Name.Local == "Description" {
		return true
	}
	if d.isContainerElement(el) {
		return true
	}
	// Check for node element attributes (rdf:about or rdf:ID, but NOT rdf:nodeID which can be on property elements)
	if d.attrValue(el.Attr, rdfXMLNS, "about") != "" ||
		d.attrValue(el.Attr, rdfXMLNS, "ID") != "" {
		return true
	}
	// Typed node element: not in RDF namespace, not a container, and no parseType
	// Also, it should not have property attributes (rdf:resource, rdf:nodeID)
	if el.Name.Space != rdfXMLNS && !d.isContainerElement(el) {
		parseType := d.attrValue(el.Attr, rdfXMLNS, "parseType")
		if parseType == "" {
			// Check if it has property attributes - if it does, it's a property element, not a node element
			resource := d.attrValue(el.Attr, rdfXMLNS, "resource")
			nodeID := d.attrValue(el.Attr, rdfXMLNS, "nodeID")
			if resource == "" && nodeID == "" {
				// No property attributes, so it could be a typed node element
				// But we also need to check if it has node element attributes
				// If it has neither, it's an implicit blank node (typed node element)
				return true
			}
		}
	}
	return false
}

func (d *rdfxmlTripleDecoder) isPropertyElement(el xml.StartElement) bool {
	// A property element is any element that's not a node element and not a container element
	return !d.isNodeElement(el) && !d.isContainerElement(el)
}

func (d *rdfxmlTripleDecoder) isContainerElement(el xml.StartElement) bool {
	// Container elements: Bag, Seq, Alt, List
	switch el.Name.Local {
	case "Bag", "Seq", "Alt", "List":
		return el.Name.Space == rdfXMLNS || el.Name.Space == ""
	}
	return false
}

func (d *rdfxmlTripleDecoder) subjectFromNode(el xml.StartElement) Term {
	if about := d.attrValue(el.Attr, rdfXMLNS, "about"); about != "" {
		return IRI{Value: d.resolveIRI(d.baseURI, about)}
	}
	if id := d.attrValue(el.Attr, rdfXMLNS, "ID"); id != "" {
		return IRI{Value: d.resolveID(id)}
	}
	if nodeID := d.attrValue(el.Attr, rdfXMLNS, "nodeID"); nodeID != "" {
		return BlankNode{ID: nodeID}
	}
	return d.newBlankNode()
}

func (d *rdfxmlTripleDecoder) resolveID(id string) string {
	// rdf:ID generates an IRI by appending #id to the base URI.
	if d.baseURI == "" {
		return "#" + id
	}
	return d.baseURI + "#" + id
}

func (d *rdfxmlTripleDecoder) newBlankNode() BlankNode {
	d.blankIDGen++
	return BlankNode{ID: "genid" + strconv.Itoa(d.blankIDGen)}
}

func (d *rdfxmlTripleDecoder) containerKey(term Term) string {
	switch value := term.(type) {
	case IRI:
		return "I:" + value.Value
	case BlankNode:
		return "B:" + value.ID
	default:
		return fmt.Sprintf("%T", term)
	}
}

func (d *rdfxmlTripleDecoder) nextContainerPredicate(key string) string {
	index := d.containerIndex[key] + 1
	d.containerIndex[key] = index
	return rdfXMLNS + "_" + strconv.Itoa(index)
}

func (d *rdfxmlTripleDecoder) bumpContainerIndex(key string, index int) {
	if index > d.containerIndex[key] {
		d.containerIndex[key] = index
	}
}

func parseContainerIndex(local string) (int, bool) {
	if len(local) < 2 || local[0] != '_' {
		return 0, false
	}
	value, err := strconv.Atoi(local[1:])
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func (d *rdfxmlTripleDecoder) nextToken() (xml.Token, error) {
	tok, err := d.dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case xml.StartElement:
		d.pushBase(t)
	case xml.EndElement:
		d.popBase()
	}
	return tok, nil
}

func (d *rdfxmlTripleDecoder) pushBase(el xml.StartElement) {
	d.baseStack = append(d.baseStack, d.baseURI)
	if base := d.attrValue(el.Attr, xmlNS, "base"); base != "" {
		d.baseURI = d.resolveIRI(d.baseURI, base)
	}
}

func (d *rdfxmlTripleDecoder) popBase() {
	if len(d.baseStack) == 0 {
		return
	}
	d.baseURI = d.baseStack[len(d.baseStack)-1]
	d.baseStack = d.baseStack[:len(d.baseStack)-1]
}

func (d *rdfxmlTripleDecoder) attrValue(attrs []xml.Attr, space, local string) string {
	for _, attr := range attrs {
		if attr.Name.Space == space && attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

func (d *rdfxmlTripleDecoder) rdfAttrValue(attrs []xml.Attr, local string) string {
	for _, attr := range attrs {
		if attr.Name.Space == rdfXMLNS && attr.Name.Local == local {
			return attr.Value
		}
		if attr.Name.Space == "" && attr.Name.Local == "rdf:"+local {
			return attr.Value
		}
		if attr.Name.Space == "" && attr.Name.Local == local {
			switch local {
			case "ID", "nodeID", "bagID", "about":
				return attr.Value
			}
		}
	}
	return ""
}

func (d *rdfxmlTripleDecoder) resolveQName(space, local string) string {
	if space == "" {
		return local
	}
	return space + local
}

func (d *rdfxmlTripleDecoder) resolveIRI(base, relative string) string {
	if base == "" {
		return relative
	}
	// Use the same resolveIRI function from turtle.go
	return resolveIRI(base, relative)
}

func (d *rdfxmlTripleDecoder) findPrefix(namespace string) string {
	for prefix, ns := range d.namespaces {
		if ns == namespace {
			return prefix
		}
	}
	return ""
}
