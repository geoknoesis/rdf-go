package rdf

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// handleNamespaceDeclarations processes namespace declarations from XML attributes
// and updates the decoder's namespace map.
func (d *rdfxmltripleDecoder) handleNamespaceDeclarations(attrs []xml.Attr) {
	for _, attr := range attrs {
		if attr.Name.Space == "" && strings.HasPrefix(attr.Name.Local, "xmlns:") {
			prefix := strings.TrimPrefix(attr.Name.Local, "xmlns:")
			d.namespaces[prefix] = attr.Value
		} else if attr.Name.Space == "" && attr.Name.Local == "xmlns" {
			d.namespaces[""] = attr.Value
		}
	}
}

// validatePropertyElement performs validation checks on a property element.
func (d *rdfxmltripleDecoder) validatePropertyElement(el xml.StartElement) error {
	if err := d.validatePropertyIDs(el.Attr); err != nil {
		return err
	}
	if el.Name.Space == rdfXMLNS && isForbiddenRDFPropertyElement(el.Name.Local) {
		return d.wrapRDFXMLError(fmt.Errorf("illegal RDF property element %s", el.Name.Local))
	}
	return nil
}

// validateParseTypeAttributes validates parseType and related attributes.
func (d *rdfxmltripleDecoder) validateParseTypeAttributes(attrs []xml.Attr, parseType string) error {
	if parseType == "Literal" {
		if err := d.validateLiteralPropertyAttributes(attrs); err != nil {
			return err
		}
	}
	if parseType != "" {
		resource := d.attrValue(attrs, rdfXMLNS, "resource")
		nodeID := d.attrValue(attrs, rdfXMLNS, "nodeID")
		if resource != "" || nodeID != "" {
			return d.wrapRDFXMLError(fmt.Errorf("rdf:parseType cannot be used with rdf:resource or rdf:nodeID"))
		}
	}
	resource := d.attrValue(attrs, rdfXMLNS, "resource")
	nodeID := d.attrValue(attrs, rdfXMLNS, "nodeID")
	if resource != "" && nodeID != "" {
		return d.wrapRDFXMLError(fmt.Errorf("rdf:resource and rdf:nodeID are mutually exclusive"))
	}
	return nil
}

// handleEmptyPropertyElementWithAttributes processes empty property elements
// that have rdf:resource or rdf:nodeID attributes.
func (d *rdfxmltripleDecoder) handleEmptyPropertyElementWithAttributes(
	el xml.StartElement,
	subject Term,
) (bool, error) {
	resource := d.attrValue(el.Attr, rdfXMLNS, "resource")
	if resource != "" {
		pred := d.resolveQName(el.Name.Space, el.Name.Local)
		obj := IRI{Value: d.resolveIRI(d.baseURI, resource)}
		d.queue = append(d.queue, Triple{S: subject, P: IRI{Value: pred}, O: obj})
		if err := d.consumeElement(); err != nil {
			return false, err
		}
		return true, nil
	}

	nodeID := d.attrValue(el.Attr, rdfXMLNS, "nodeID")
	if nodeID != "" {
		if !isValidXMLName(nodeID) {
			return false, d.wrapRDFXMLError(fmt.Errorf("invalid rdf:nodeID %q", nodeID))
		}
		pred := d.resolveQName(el.Name.Space, el.Name.Local)
		obj := BlankNode{ID: nodeID}
		d.queue = append(d.queue, Triple{S: subject, P: IRI{Value: pred}, O: obj})
		if err := d.consumeElement(); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// resolveContainerPredicate resolves the predicate for container membership properties.
// Returns the resolved predicate and whether the container index was updated.
func (d *rdfxmltripleDecoder) resolveContainerPredicate(
	el xml.StartElement,
	containerKey string,
) (string, bool) {
	if !d.expandContainers {
		return d.resolveQName(el.Name.Space, el.Name.Local), false
	}

	if el.Name.Space == rdfXMLNS && el.Name.Local == "li" {
		return d.nextContainerPredicate(containerKey), true
	}

	if el.Name.Space == rdfXMLNS && strings.HasPrefix(el.Name.Local, "_") {
		if idx, ok := parseContainerIndex(el.Name.Local); ok {
			d.bumpContainerIndex(containerKey, idx)
			return rdfXMLNS + el.Name.Local, true
		}
	}

	return d.resolveQName(el.Name.Space, el.Name.Local), false
}

// processPropertyElement processes a property element and adds the resulting triple to the queue.
// It handles container membership expansion and annotations.
func (d *rdfxmltripleDecoder) processPropertyElement(
	el xml.StartElement,
	subject Term,
	containerKey string,
) error {
	pred, _ := d.resolveContainerPredicate(el, containerKey)
	obj, annotation, annotationNodeID, err := d.objectFromPredicate(el)
	if err != nil {
		return err
	}

	if obj == nil {
		return nil
	}

	triple := Triple{S: subject, P: IRI{Value: pred}, O: obj}
	d.queue = append(d.queue, triple)

	if annotation != "" || annotationNodeID != "" {
		anns := d.handleAnnotation(subject, IRI{Value: pred}, obj, annotation, annotationNodeID)
		for _, ann := range anns {
			d.queue = append(d.queue, ann)
		}
	}

	return nil
}
