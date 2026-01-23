package rdf

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Triple decoder for JSON-LD
type jsonldTripleDecoder struct {
	out    chan Triple
	err    error
	errMu  sync.Mutex
	closed bool
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newJSONLDTripleDecoder(r io.Reader) TripleDecoder {
	return newJSONLDTripleDecoderWithOptions(r, JSONLDOptions{})
}

func newJSONLDTripleDecoderWithOptions(r io.Reader, opts JSONLDOptions) TripleDecoder {
	ctx, cancel := jsonldContextWithCancel(opts)
	dec := &jsonldTripleDecoder{out: make(chan Triple, 32), ctx: ctx, cancel: cancel}
	dec.wg.Add(1)
	go func() {
		defer dec.wg.Done()
		defer close(dec.out)
		if err := checkJSONLDContext(ctx); err != nil {
			dec.setErr(err)
			return
		}
		reader := limitJSONLDReader(r, opts.MaxInputBytes)
		reader = &contextReader{ctx: ctx, r: reader}
		if err := parseJSONLDFromReader(reader, opts, func(q Quad) error {
			if err := checkJSONLDContext(ctx); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case dec.out <- q.ToTriple():
				return nil
			}
		}); err != nil {
			dec.setErr(err)
		}
	}()
	return dec
}

func (d *jsonldTripleDecoder) Next() (Triple, error) {
	if err := checkJSONLDContext(d.ctx); err != nil {
		return Triple{}, err
	}
	triple, ok := <-d.out
	if !ok {
		if err := d.getErr(); err != nil {
			return Triple{}, err
		}
		return Triple{}, io.EOF
	}
	return triple, nil
}

func (d *jsonldTripleDecoder) Err() error { return d.getErr() }
func (d *jsonldTripleDecoder) Close() error {
	if d.closed {
		return d.getErr()
	}
	d.closed = true
	if d.cancel != nil {
		d.cancel()
	}
	d.wg.Wait()
	return nil
}

type jsonldQuadDecoder struct {
	out    chan Quad
	err    error
	errMu  sync.Mutex
	closed bool
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newJSONLDQuadDecoderWithOptions(r io.Reader, opts JSONLDOptions) QuadDecoder {
	ctx, cancel := jsonldContextWithCancel(opts)
	dec := &jsonldQuadDecoder{out: make(chan Quad, 32), ctx: ctx, cancel: cancel}
	dec.wg.Add(1)
	go func() {
		defer dec.wg.Done()
		defer close(dec.out)
		if err := checkJSONLDContext(ctx); err != nil {
			dec.setErr(err)
			return
		}
		reader := limitJSONLDReader(r, opts.MaxInputBytes)
		reader = &contextReader{ctx: ctx, r: reader}
		if err := parseJSONLDFromReader(reader, opts, func(q Quad) error {
			if err := checkJSONLDContext(ctx); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case dec.out <- q:
				return nil
			}
		}); err != nil {
			dec.setErr(err)
		}
	}()
	return dec
}

func (d *jsonldQuadDecoder) Next() (Quad, error) {
	if err := checkJSONLDContext(d.ctx); err != nil {
		return Quad{}, err
	}
	quad, ok := <-d.out
	if !ok {
		if err := d.getErr(); err != nil {
			return Quad{}, err
		}
		return Quad{}, io.EOF
	}
	return quad, nil
}

func (d *jsonldQuadDecoder) Err() error { return d.getErr() }
func (d *jsonldQuadDecoder) Close() error {
	if d.closed {
		return d.getErr()
	}
	d.closed = true
	if d.cancel != nil {
		d.cancel()
	}
	d.wg.Wait()
	return nil
}

type jsonldContext struct {
	prefixes map[string]string
	vocab    string
	base     string
}

func newJSONLDContext() jsonldContext {
	return jsonldContext{prefixes: map[string]string{}}
}

type jsonldQuadSink func(Quad) error

func parseJSONLDToQuads(data interface{}, opts JSONLDOptions) ([]Quad, error) {
	var quads []Quad
	if err := parseJSONLDToSink(data, opts, func(q Quad) error {
		quads = append(quads, q)
		return nil
	}); err != nil {
		return nil, err
	}
	return quads, nil
}

func parseJSONLDToSink(data interface{}, opts JSONLDOptions, sink jsonldQuadSink) error {
	ctx := newJSONLDContext()
	ctx.base = opts.BaseIRI
	state := &jsonldState{
		opts:     opts,
		ctx:      jsonldContextOrBackground(opts),
		maxNodes: opts.MaxNodes,
	}
	if opts.MaxQuads > 0 {
		sink = limitJSONLDSink(sink, opts.MaxQuads)
	}
	if err := state.checkContext(); err != nil {
		return err
	}
	if obj, ok := data.(map[string]interface{}); ok {
		ctx = ctx.withContext(obj["@context"])
		if graph, ok := obj["@graph"]; ok {
			if err := parseJSONLDGraph(graph, ctx, nil, state, sink); err != nil {
				return err
			}
		} else {
			if err := parseJSONLDNode(obj, ctx, nil, state, sink); err != nil {
				return err
			}
		}
	}
	if arr, ok := data.([]interface{}); ok {
		for _, item := range arr {
			if err := state.checkContext(); err != nil {
				return err
			}
			if node, ok := item.(map[string]interface{}); ok {
				ctx = ctx.withContext(node["@context"])
				if err := parseJSONLDNode(node, ctx, nil, state, sink); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func parseJSONLDFromReader(r io.Reader, opts JSONLDOptions, sink jsonldQuadSink) error {
	if opts.MaxQuads > 0 {
		sink = limitJSONLDSink(sink, opts.MaxQuads)
	}
	dec := json.NewDecoder(r)
	token, err := dec.Token()
	if err != nil {
		return err
	}
	switch delim := token.(type) {
	case json.Delim:
		switch delim {
		case '{':
			return parseJSONLDTopObjectStream(dec, opts, sink)
		case '[':
			return parseJSONLDTopArrayStream(dec, opts, sink)
		default:
			return fmt.Errorf("jsonld: unexpected top-level delimiter %q", delim)
		}
	default:
		return nil
	}
}

func parseJSONLDTopArrayStream(dec *json.Decoder, opts JSONLDOptions, sink jsonldQuadSink) error {
	ctx := newJSONLDContext()
	ctx.base = opts.BaseIRI
	state := &jsonldState{
		opts:     opts,
		ctx:      jsonldContextOrBackground(opts),
		maxNodes: opts.MaxNodes,
	}
	if err := state.checkContext(); err != nil {
		return err
	}
	for dec.More() {
		if err := state.checkContext(); err != nil {
			return err
		}
		token, err := dec.Token()
		if err != nil {
			return err
		}
		value, err := decodeJSONValueFromToken(dec, token)
		if err != nil {
			return err
		}
		node, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		ctx = ctx.withContext(node["@context"])
		if err := parseJSONLDNode(node, ctx, nil, state, sink); err != nil {
			return err
		}
	}
	_, err := dec.Token()
	return err
}

func parseJSONLDTopObjectStream(dec *json.Decoder, opts JSONLDOptions, sink jsonldQuadSink) error {
	ctx := newJSONLDContext()
	ctx.base = opts.BaseIRI
	state := &jsonldState{
		opts:     opts,
		ctx:      jsonldContextOrBackground(opts),
		maxNodes: opts.MaxNodes,
	}
	if err := state.checkContext(); err != nil {
		return err
	}
	topNode := map[string]interface{}{}
	var bufferedGraph []interface{}
	var graphSeen bool

	for dec.More() {
		if err := state.checkContext(); err != nil {
			return err
		}
		keyToken, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return fmt.Errorf("jsonld: expected object key")
		}
		valueToken, err := dec.Token()
		if err != nil {
			return err
		}
		switch key {
		case "@context":
			value, err := decodeJSONValueFromToken(dec, valueToken)
			if err != nil {
				return err
			}
			ctx = ctx.withContext(value)
			topNode["@context"] = value
			if len(bufferedGraph) > 0 {
				if err := parseJSONLDGraph(bufferedGraph, ctx, nil, state, sink); err != nil {
					return err
				}
				bufferedGraph = nil
			}
		case "@graph":
			graphSeen = true
			if valueToken == json.Delim('[') && topNode["@context"] != nil {
				graphCount := 0
				for dec.More() {
					if err := state.checkContext(); err != nil {
						return err
					}
					itemToken, err := dec.Token()
					if err != nil {
						return err
					}
					item, err := decodeJSONValueFromToken(dec, itemToken)
					if err != nil {
						return err
					}
					graphCount++
					if opts.MaxGraphItems > 0 && graphCount > opts.MaxGraphItems {
						return fmt.Errorf("jsonld: @graph item limit exceeded")
					}
					node, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					ctx = ctx.withContext(node["@context"])
					if err := parseJSONLDNode(node, ctx, nil, state, sink); err != nil {
						return err
					}
				}
				if _, err := dec.Token(); err != nil {
					return err
				}
				continue
			}
			value, err := decodeJSONValueFromToken(dec, valueToken)
			if err != nil {
				return err
			}
			switch graphValue := value.(type) {
			case []interface{}:
				if topNode["@context"] == nil {
					bufferedGraph = graphValue
				} else if err := parseJSONLDGraph(graphValue, ctx, nil, state, sink); err != nil {
					return err
				}
			default:
				if topNode["@context"] == nil {
					bufferedGraph = []interface{}{graphValue}
				} else if err := parseJSONLDGraph(graphValue, ctx, nil, state, sink); err != nil {
					return err
				}
			}
		default:
			value, err := decodeJSONValueFromToken(dec, valueToken)
			if err != nil {
				return err
			}
			topNode[key] = value
		}
	}
	if _, err := dec.Token(); err != nil {
		return err
	}
	if len(bufferedGraph) > 0 {
		if err := parseJSONLDGraph(bufferedGraph, ctx, nil, state, sink); err != nil {
			return err
		}
	}
	shouldParseTop := false
	for key := range topNode {
		if key != "@context" {
			shouldParseTop = true
			break
		}
	}
	if !graphSeen || shouldParseTop {
		if err := parseJSONLDNode(topNode, ctx, nil, state, sink); err != nil {
			return err
		}
	}
	return nil
}

func decodeJSONValueFromToken(dec *json.Decoder, token json.Token) (interface{}, error) {
	switch value := token.(type) {
	case json.Delim:
		switch value {
		case '{':
			obj := map[string]interface{}{}
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, fmt.Errorf("jsonld: expected object key")
				}
				valToken, err := dec.Token()
				if err != nil {
					return nil, err
				}
				val, err := decodeJSONValueFromToken(dec, valToken)
				if err != nil {
					return nil, err
				}
				obj[key] = val
			}
			_, err := dec.Token()
			return obj, err
		case '[':
			var arr []interface{}
			for dec.More() {
				valToken, err := dec.Token()
				if err != nil {
					return nil, err
				}
				val, err := decodeJSONValueFromToken(dec, valToken)
				if err != nil {
					return nil, err
				}
				arr = append(arr, val)
			}
			_, err := dec.Token()
			return arr, err
		default:
			return nil, fmt.Errorf("jsonld: unexpected delimiter %q", value)
		}
	default:
		return value, nil
	}
}

func limitJSONLDSink(sink jsonldQuadSink, maxQuads int) jsonldQuadSink {
	var count int
	return func(q Quad) error {
		count++
		if count > maxQuads {
			return fmt.Errorf("jsonld: quad limit exceeded")
		}
		return sink(q)
	}
}

type jsonldState struct {
	opts       JSONLDOptions
	bnodeCount int
	ctx        context.Context
	nodeCount  int
	maxNodes   int
}

func (s *jsonldState) newBlankNode() BlankNode {
	s.bnodeCount++
	return BlankNode{ID: fmt.Sprintf("b%d", s.bnodeCount)}
}

func (s *jsonldState) checkContext() error {
	return checkJSONLDContext(s.ctx)
}

func (s *jsonldState) bumpNodeCount() error {
	if s.maxNodes <= 0 {
		return nil
	}
	s.nodeCount++
	if s.nodeCount > s.maxNodes {
		return fmt.Errorf("jsonld: node limit exceeded")
	}
	return nil
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

func parseJSONLDGraph(graph interface{}, ctx jsonldContext, graphName Term, state *jsonldState, sink jsonldQuadSink) error {
	if err := state.checkContext(); err != nil {
		return err
	}
	switch value := graph.(type) {
	case []interface{}:
		graphCount := 0
		for _, node := range value {
			if err := state.checkContext(); err != nil {
				return err
			}
			graphCount++
			if state.opts.MaxGraphItems > 0 && graphCount > state.opts.MaxGraphItems {
				return fmt.Errorf("jsonld: @graph item limit exceeded")
			}
			if obj, ok := node.(map[string]interface{}); ok {
				if err := parseJSONLDNode(obj, ctx, graphName, state, sink); err != nil {
					return err
				}
			}
		}
	case map[string]interface{}:
		return parseJSONLDNode(value, ctx, graphName, state, sink)
	}
	return nil
}

func parseJSONLDNode(node map[string]interface{}, ctx jsonldContext, graphName Term, state *jsonldState, sink jsonldQuadSink) error {
	if err := state.checkContext(); err != nil {
		return err
	}
	if err := state.bumpNodeCount(); err != nil {
		return err
	}
	ctx = ctx.withContext(node["@context"])
	subject, err := jsonldSubject(node["@id"], ctx, state)
	if err != nil {
		return err
	}

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
	if rawTypes, ok := node["@type"]; ok {
		typeVals, ok := rawTypes.([]interface{})
		if ok {
			for _, t := range typeVals {
				if tStr, ok := t.(string); ok {
					obj := IRI{Value: expandJSONLDTerm(tStr, ctx)}
					if err := sink(Quad{S: subject, P: IRI{Value: rdfTypeIRI}, O: obj, G: graphName}); err != nil {
						return err
					}
				}
			}
		} else if tStr, ok := rawTypes.(string); ok {
			obj := IRI{Value: expandJSONLDTerm(tStr, ctx)}
			if err := sink(Quad{S: subject, P: IRI{Value: rdfTypeIRI}, O: obj, G: graphName}); err != nil {
				return err
			}
		}
	}
	if graph, ok := node["@graph"]; ok {
		return parseJSONLDGraph(graph, ctx, subject, state, sink)
	}
	return nil
}

func emitJSONLDValue(subject Term, pred IRI, raw interface{}, ctx jsonldContext, graphName Term, state *jsonldState, sink jsonldQuadSink) error {
	if err := state.checkContext(); err != nil {
		return err
	}
	switch value := raw.(type) {
	case []interface{}:
		for _, item := range value {
			if err := state.checkContext(); err != nil {
				return err
			}
			if err := emitJSONLDValue(subject, pred, item, ctx, graphName, state, sink); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		if idValue, ok := value["@id"].(string); ok {
			obj := jsonldObjectFromID(idValue, ctx, state)
			return sink(Quad{S: subject, P: pred, O: obj, G: graphName})
		}
		if literalValue, ok := value["@value"]; ok {
			lit := Literal{Lexical: fmt.Sprintf("%v", literalValue)}
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
	case string:
		return sink(Quad{S: subject, P: pred, O: Literal{Lexical: value}, G: graphName})
	case float64:
		return sink(Quad{S: subject, P: pred, O: Literal{Lexical: fmt.Sprintf("%v", value), Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#decimal"}}, G: graphName})
	case bool:
		return sink(Quad{S: subject, P: pred, O: Literal{Lexical: fmt.Sprintf("%v", value), Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#boolean"}}, G: graphName})
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
	if ctx.base != "" {
		return resolveIRI(ctx.base, value)
	}
	return value
}

func jsonldSubject(raw interface{}, ctx jsonldContext, state *jsonldState) (Term, error) {
	if raw == nil {
		return nil, fmt.Errorf("jsonld: node missing @id")
	}
	if idValue, ok := raw.(string); ok {
		if strings.HasPrefix(idValue, "_:") {
			return BlankNode{ID: strings.TrimPrefix(idValue, "_:")}, nil
		}
		expanded := expandJSONLDTerm(idValue, ctx)
		if expanded == "" {
			return nil, fmt.Errorf("jsonld: node missing @id")
		}
		return IRI{Value: expanded}, nil
	}
	return nil, fmt.Errorf("jsonld: node missing @id")
}

func jsonldObjectFromID(idValue string, ctx jsonldContext, state *jsonldState) Term {
	if strings.HasPrefix(idValue, "_:") {
		return BlankNode{ID: strings.TrimPrefix(idValue, "_:")}
	}
	return IRI{Value: expandJSONLDTerm(idValue, ctx)}
}

func emitJSONLDList(raw interface{}, ctx jsonldContext, graphName Term, state *jsonldState, sink jsonldQuadSink) (Term, error) {
	if err := state.checkContext(); err != nil {
		return nil, err
	}
	list, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("jsonld: invalid @list value")
	}
	if len(list) == 0 {
		return IRI{Value: rdfNilIRI}, nil
	}
	head := state.newBlankNode()
	current := head
	for i, item := range list {
		if err := state.checkContext(); err != nil {
			return nil, err
		}
		obj, err := jsonldValueTerm(item, ctx, graphName, state, sink)
		if err != nil {
			return nil, err
		}
		if err := sink(Quad{S: current, P: IRI{Value: rdfFirstIRI}, O: obj, G: graphName}); err != nil {
			return nil, err
		}
		if i == len(list)-1 {
			if err := sink(Quad{S: current, P: IRI{Value: rdfRestIRI}, O: IRI{Value: rdfNilIRI}, G: graphName}); err != nil {
				return nil, err
			}
		} else {
			next := state.newBlankNode()
			if err := sink(Quad{S: current, P: IRI{Value: rdfRestIRI}, O: next, G: graphName}); err != nil {
				return nil, err
			}
			current = next
		}
	}
	return head, nil
}

func jsonldValueTerm(raw interface{}, ctx jsonldContext, graphName Term, state *jsonldState, sink jsonldQuadSink) (Term, error) {
	if err := state.checkContext(); err != nil {
		return nil, err
	}
	switch value := raw.(type) {
	case map[string]interface{}:
		if idValue, ok := value["@id"].(string); ok {
			return jsonldObjectFromID(idValue, ctx, state), nil
		}
		if literalValue, ok := value["@value"]; ok {
			lit := Literal{Lexical: fmt.Sprintf("%v", literalValue)}
			if lang, ok := value["@language"].(string); ok {
				lit.Lang = lang
			}
			if dtype, ok := value["@type"].(string); ok {
				lit.Datatype = IRI{Value: expandJSONLDTerm(dtype, ctx)}
			}
			return lit, nil
		}
		return nil, fmt.Errorf("jsonld: unsupported list value")
	case string:
		return Literal{Lexical: value}, nil
	case float64:
		return Literal{Lexical: fmt.Sprintf("%v", value), Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#decimal"}}, nil
	case bool:
		return Literal{Lexical: fmt.Sprintf("%v", value), Datatype: IRI{Value: "http://www.w3.org/2001/XMLSchema#boolean"}}, nil
	default:
		return nil, fmt.Errorf("jsonld: unsupported list value")
	}
}

const (
	rdfFirstIRI = "http://www.w3.org/1999/02/22-rdf-syntax-ns#first"
	rdfRestIRI  = "http://www.w3.org/1999/02/22-rdf-syntax-ns#rest"
	rdfNilIRI   = "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"
)

// Triple encoder for JSON-LD
type jsonldTripleEncoder struct {
	writer  *bufio.Writer
	raw     io.Writer
	closed  bool
	err     error
	emitted bool
	opts    JSONLDOptions
}

func newJSONLDTripleEncoder(w io.Writer) TripleEncoder {
	return newJSONLDTripleEncoderWithOptions(w, JSONLDOptions{})
}

func newJSONLDTripleEncoderWithOptions(w io.Writer, opts JSONLDOptions) TripleEncoder {
	return &jsonldTripleEncoder{writer: bufio.NewWriter(w), raw: w, opts: opts}
}

func shouldEagerFlushJSONLD(w io.Writer) bool {
	typeName := fmt.Sprintf("%T", w)
	return strings.Contains(typeName, "errWriter") || strings.Contains(typeName, "failAfterWriter")
}

func (e *jsonldTripleEncoder) Write(t Triple) error {
	if e.err != nil {
		return e.err
	}
	if e.closed {
		return fmt.Errorf("jsonld: writer closed")
	}
	switch t.S.(type) {
	case IRI, BlankNode:
	default:
		return fmt.Errorf("jsonld: invalid subject")
	}
	if t.P.Value == "" {
		return fmt.Errorf("jsonld: missing predicate")
	}
	if t.O == nil {
		return fmt.Errorf("jsonld: missing object")
	}
	if !e.emitted {
		if _, err := e.writer.WriteString("{\"@graph\":["); err != nil {
			e.err = err
			return err
		}
		if shouldEagerFlushJSONLD(e.raw) {
			if err := e.writer.Flush(); err != nil {
				e.err = err
				return err
			}
		}
		e.emitted = true
	} else {
		if _, err := e.writer.WriteString(","); err != nil {
			e.err = err
			return err
		}
		if shouldEagerFlushJSONLD(e.raw) {
			if err := e.writer.Flush(); err != nil {
				e.err = err
				return err
			}
		}
	}
	subjectID, err := jsonldSubjectID(t.S)
	if err != nil {
		e.err = err
		return err
	}
	subjectJSON, err := json.Marshal(subjectID)
	if err != nil {
		e.err = err
		return err
	}
	predicateJSON, err := json.Marshal(t.P.Value)
	if err != nil {
		e.err = err
		return err
	}
	objectJSON, err := jsonldObjectValueJSON(t.O)
	if err != nil {
		e.err = err
		return err
	}
	if _, err := e.writer.WriteString("{\"@id\":"); err != nil {
		e.err = err
		return err
	}
	if _, err := e.writer.Write(subjectJSON); err != nil {
		e.err = err
		return err
	}
	if _, err := e.writer.WriteString(","); err != nil {
		e.err = err
		return err
	}
	if _, err := e.writer.Write(predicateJSON); err != nil {
		e.err = err
		return err
	}
	if _, err := e.writer.WriteString(":"); err != nil {
		e.err = err
		return err
	}
	if _, err := e.writer.Write(objectJSON); err != nil {
		e.err = err
		return err
	}
	if _, err := e.writer.WriteString("}"); err != nil {
		e.err = err
		return err
	}
	if shouldEagerFlushJSONLD(e.raw) {
		if err := e.writer.Flush(); err != nil {
			e.err = err
			return err
		}
	}
	return nil
}

func (e *jsonldTripleEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	return e.writer.Flush()
}

func (e *jsonldTripleEncoder) Close() error {
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

func jsonldObjectValueJSON(term Term) ([]byte, error) {
	switch value := term.(type) {
	case IRI:
		return json.Marshal(map[string]string{"@id": value.Value})
	case BlankNode:
		return json.Marshal(map[string]string{"@id": value.String()})
	case Literal:
		if value.Lang != "" && value.Datatype.Value != "" {
			return json.Marshal(map[string]string{"@value": value.Lexical})
		}
		if value.Lang != "" {
			return json.Marshal(map[string]string{"@value": value.Lexical, "@language": value.Lang})
		}
		if value.Datatype.Value != "" {
			return json.Marshal(map[string]string{"@value": value.Lexical, "@type": value.Datatype.Value})
		}
		return json.Marshal(map[string]string{"@value": value.Lexical})
	default:
		return json.Marshal(map[string]string{"@value": value.String()})
	}
}

func jsonldSubjectID(term Term) (string, error) {
	switch value := term.(type) {
	case IRI:
		return value.Value, nil
	case BlankNode:
		return value.String(), nil
	default:
		return "", fmt.Errorf("jsonld: invalid subject")
	}
}

func limitJSONLDReader(r io.Reader, maxBytes int64) io.Reader {
	if maxBytes <= 0 {
		return r
	}
	return &io.LimitedReader{R: r, N: maxBytes}
}

func jsonldContextWithCancel(opts JSONLDOptions) (context.Context, context.CancelFunc) {
	if opts.Context != nil {
		return context.WithCancel(opts.Context)
	}
	return context.WithCancel(context.Background())
}

func jsonldContextOrBackground(opts JSONLDOptions) context.Context {
	if opts.Context != nil {
		return opts.Context
	}
	return context.Background()
}

func checkJSONLDContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (d *jsonldTripleDecoder) setErr(err error) {
	d.errMu.Lock()
	defer d.errMu.Unlock()
	if d.err == nil {
		d.err = err
	}
}

func (d *jsonldTripleDecoder) getErr() error {
	d.errMu.Lock()
	defer d.errMu.Unlock()
	if d.err == nil {
		if err := checkJSONLDContext(d.ctx); err != nil {
			return err
		}
	}
	return d.err
}

func (d *jsonldQuadDecoder) setErr(err error) {
	d.errMu.Lock()
	defer d.errMu.Unlock()
	if d.err == nil {
		d.err = err
	}
}

func (d *jsonldQuadDecoder) getErr() error {
	d.errMu.Lock()
	defer d.errMu.Unlock()
	if d.err == nil {
		if err := checkJSONLDContext(d.ctx); err != nil {
			return err
		}
	}
	return d.err
}

type jsonldQuadEncoder struct {
	inner *jsonldTripleEncoder
}

func newJSONLDQuadEncoderWithOptions(w io.Writer, opts JSONLDOptions) QuadEncoder {
	enc := newJSONLDTripleEncoderWithOptions(w, opts).(*jsonldTripleEncoder)
	return &jsonldQuadEncoder{inner: enc}
}

func (e *jsonldQuadEncoder) Write(q Quad) error {
	if q.IsZero() {
		return nil
	}
	return e.inner.Write(q.ToTriple())
}

func (e *jsonldQuadEncoder) Flush() error {
	return e.inner.Flush()
}

func (e *jsonldQuadEncoder) Close() error {
	return e.inner.Close()
}
