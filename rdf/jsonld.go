package rdf

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type jsonldDecoder struct {
	quads []Quad
	index int
	err   error
}

func newJSONLDDecoder(r io.Reader) Decoder {
	dec := &jsonldDecoder{}
	if err := dec.load(r); err != nil {
		dec.err = err
	}
	return dec
}

func (d *jsonldDecoder) Next() (Quad, error) {
	if d.err != nil {
		return Quad{}, d.err
	}
	if d.index >= len(d.quads) {
		return Quad{}, io.EOF
	}
	q := d.quads[d.index]
	d.index++
	return q, nil
}

func (d *jsonldDecoder) Err() error { return d.err }
func (d *jsonldDecoder) Close() error {
	return nil
}

func (d *jsonldDecoder) load(r io.Reader) error {
	var data interface{}
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return err
	}
	ctx := newJSONLDContext()
	quads := make([]Quad, 0)
	if obj, ok := data.(map[string]interface{}); ok {
		ctx = ctx.withContext(obj["@context"])
		if graph, ok := obj["@graph"]; ok {
			if err := parseJSONLDGraph(graph, ctx, &quads); err != nil {
				return err
			}
		} else {
			if err := parseJSONLDNode(obj, ctx, &quads); err != nil {
				return err
			}
		}
		d.quads = quads
		return nil
	}
	if arr, ok := data.([]interface{}); ok {
		for _, item := range arr {
			if node, ok := item.(map[string]interface{}); ok {
				ctx = ctx.withContext(node["@context"])
				if err := parseJSONLDNode(node, ctx, &quads); err != nil {
					return err
				}
			}
		}
		d.quads = quads
		return nil
	}
	d.quads = quads
	return nil
}

type jsonldContext struct {
	prefixes map[string]string
	vocab    string
}

func newJSONLDContext() jsonldContext {
	return jsonldContext{prefixes: map[string]string{}}
}

func (c jsonldContext) withContext(raw interface{}) jsonldContext {
	if raw == nil {
		return c
	}
	if ctxMap, ok := raw.(map[string]interface{}); ok {
		for key, value := range ctxMap {
			if key == "@vocab" {
				if str, ok := value.(string); ok {
					c.vocab = str
				}
				continue
			}
			if str, ok := value.(string); ok {
				c.prefixes[key] = str
			}
		}
	}
	return c
}

func parseJSONLDGraph(graph interface{}, ctx jsonldContext, quads *[]Quad) error {
	switch value := graph.(type) {
	case []interface{}:
		for _, node := range value {
			if obj, ok := node.(map[string]interface{}); ok {
				if err := parseJSONLDNode(obj, ctx, quads); err != nil {
					return err
				}
			}
		}
	case map[string]interface{}:
		return parseJSONLDNode(value, ctx, quads)
	}
	return nil
}

func parseJSONLDNode(node map[string]interface{}, ctx jsonldContext, quads *[]Quad) error {
	ctx = ctx.withContext(node["@context"])
	subjectIRI := ""
	if idValue, ok := node["@id"].(string); ok {
		subjectIRI = expandJSONLDTerm(idValue, ctx)
	}
	if subjectIRI == "" {
		return fmt.Errorf("jsonld: node missing @id")
	}
	subject := IRI{Value: subjectIRI}

	for key, raw := range node {
		if strings.HasPrefix(key, "@") {
			continue
		}
		pred := IRI{Value: expandJSONLDTerm(key, ctx)}
		if pred.Value == "" {
			return fmt.Errorf("jsonld: cannot resolve predicate %q", key)
		}
		if err := emitJSONLDValue(subject, pred, raw, ctx, quads); err != nil {
			return err
		}
	}
	return nil
}

func emitJSONLDValue(subject Term, pred IRI, raw interface{}, ctx jsonldContext, quads *[]Quad) error {
	switch value := raw.(type) {
	case []interface{}:
		for _, item := range value {
			if err := emitJSONLDValue(subject, pred, item, ctx, quads); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		if idValue, ok := value["@id"].(string); ok {
			obj := IRI{Value: expandJSONLDTerm(idValue, ctx)}
			*quads = append(*quads, Quad{S: subject, P: pred, O: obj})
			return nil
		}
		if literalValue, ok := value["@value"]; ok {
			lit := Literal{Lexical: fmt.Sprintf("%v", literalValue)}
			*quads = append(*quads, Quad{S: subject, P: pred, O: lit})
			return nil
		}
		return fmt.Errorf("jsonld: unsupported object value")
	case string:
		*quads = append(*quads, Quad{S: subject, P: pred, O: Literal{Lexical: value}})
	case float64:
		*quads = append(*quads, Quad{S: subject, P: pred, O: Literal{Lexical: fmt.Sprintf("%v", value), Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#decimal"}}})
	case bool:
		*quads = append(*quads, Quad{S: subject, P: pred, O: Literal{Lexical: fmt.Sprintf("%v", value), Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#boolean"}}})
	default:
		return fmt.Errorf("jsonld: unsupported literal value")
	}
	return nil
}

func expandJSONLDTerm(value string, ctx jsonldContext) string {
	if strings.Contains(value, ":") {
		parts := strings.SplitN(value, ":", 2)
		if base, ok := ctx.prefixes[parts[0]]; ok {
			return base + parts[1]
		}
		return value
	}
	if ctx.vocab != "" {
		return ctx.vocab + value
	}
	return value
}

type jsonldEncoder struct {
	writer  *bufio.Writer
	closed  bool
	err     error
	emitted bool
}

func newJSONLDEncoder(w io.Writer) Encoder {
	return &jsonldEncoder{writer: bufio.NewWriter(w)}
}

func (e *jsonldEncoder) Write(q Quad) error {
	if e.err != nil {
		return e.err
	}
	if e.closed {
		return fmt.Errorf("jsonld: writer closed")
	}
	if !e.emitted {
		if _, err := e.writer.WriteString("{\"@graph\":["); err != nil {
			e.err = err
			return err
		}
		e.emitted = true
	} else {
		if _, err := e.writer.WriteString(","); err != nil {
			e.err = err
			return err
		}
	}
	fragment := fmt.Sprintf("{\"@id\":%q,%q:%s}", renderTerm(q.S), q.P.Value, jsonldObjectValue(q.O))
	_, err := e.writer.WriteString(fragment)
	if err != nil {
		e.err = err
	}
	return err
}

func (e *jsonldEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	return e.writer.Flush()
}

func (e *jsonldEncoder) Close() error {
	if e.closed {
		return e.err
	}
	e.closed = true
	if e.emitted {
		if _, err := e.writer.WriteString("]}"); err != nil {
			e.err = err
			return err
		}
	}
	return e.Flush()
}

func jsonldObjectValue(term Term) string {
	switch value := term.(type) {
	case IRI:
		return fmt.Sprintf("{\"@id\":%q}", value.Value)
	case Literal:
		return fmt.Sprintf("{\"@value\":%q}", value.Lexical)
	default:
		return fmt.Sprintf("{\"@value\":%q}", value.String())
	}
}
