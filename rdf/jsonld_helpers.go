package rdf

import (
	"fmt"
	"strings"
)

// emitJSONLDLiteralValue handles literal value emission for JSON-LD.
// It creates a Literal term from various primitive types (string, float64, bool).
func emitJSONLDLiteralValue(value interface{}, ctx jsonldContext) Literal {
	lit := Literal{Lexical: fmt.Sprintf("%v", value)}

	switch value.(type) {
	case float64:
		lit.Datatype = IRI{Value: "http://www.w3.org/2001/XMLSchema#decimal"}
	case bool:
		lit.Datatype = IRI{Value: "http://www.w3.org/2001/XMLSchema#boolean"}
	}

	return lit
}

// emitJSONLDObjectValue handles object value emission for JSON-LD.
// It processes map[string]interface{} values that represent objects with @id, @value, or @list.
func emitJSONLDObjectValue(value map[string]interface{}, subject Term, pred IRI, ctx jsonldContext, graphName Term, state *jsonldState, sink jsonldQuadSink) error {
	if idValue, ok := value["@id"].(string); ok {
		obj := jsonldObjectFromID(idValue, ctx, state)
		return sink(Quad{S: subject, P: pred, O: obj, G: graphName})
	}

	if literalValue, ok := value["@value"]; ok {
		lit := emitJSONLDLiteralValue(literalValue, ctx)
		if lang, ok := value["@language"].(string); ok {
			lit.Lang = lang
		}
		if dtype, ok := value["@type"].(string); ok {
			lit.Datatype = IRI{Value: expandJSONLDTerm(dtype, ctx)}
		}
		return sink(Quad{S: subject, P: pred, O: lit, G: graphName})
	}

	if listValue, ok := value["@list"]; ok {
		listObj, err := emitJSONLDList(listValue, ctx, graphName, state, sink)
		if err != nil {
			return err
		}
		return sink(Quad{S: subject, P: pred, O: listObj, G: graphName})
	}

	return fmt.Errorf("jsonld: unsupported object value")
}

// emitJSONLDTypeStatements emits RDF type statements for @type values in a node.
// It handles both single string values and arrays of type strings.
func emitJSONLDTypeStatements(subject Term, rawTypes interface{}, ctx jsonldContext, graphName Term, sink jsonldQuadSink) error {
	typeVals, ok := rawTypes.([]interface{})
	if ok {
		// Handle array of types
		for _, t := range typeVals {
			if tStr, ok := t.(string); ok {
				obj := IRI{Value: expandJSONLDTerm(tStr, ctx)}
				if err := sink(Quad{S: subject, P: IRI{Value: rdfTypeIRI}, O: obj, G: graphName}); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// Handle single type string
	if tStr, ok := rawTypes.(string); ok {
		obj := IRI{Value: expandJSONLDTerm(tStr, ctx)}
		return sink(Quad{S: subject, P: IRI{Value: rdfTypeIRI}, O: obj, G: graphName})
	}

	return nil
}

// emitJSONLDPredicateValues processes all predicate-value pairs in a node.
// It skips @-prefixed keys and emits quads for each predicate.
func emitJSONLDPredicateValues(node map[string]interface{}, subject Term, ctx jsonldContext, graphName Term, state *jsonldState, sink jsonldQuadSink) error {
	for key, raw := range node {
		if err := state.checkContext(); err != nil {
			return err
		}
		if strings.HasPrefix(key, "@") {
			continue
		}
		pred := IRI{Value: expandJSONLDTerm(key, ctx)}
		if pred.Value == "" {
			return fmt.Errorf("jsonld: cannot resolve predicate %q", key)
		}
		if err := emitJSONLDValue(subject, pred, raw, ctx, graphName, state, sink); err != nil {
			return err
		}
	}
	return nil
}
