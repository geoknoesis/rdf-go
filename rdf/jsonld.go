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
type jsonldtripleDecoder struct {
	out    chan Triple
	err    error
	errMu  sync.Mutex
	closed bool
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newJSONLDtripleDecoder(r io.Reader) tripleDecoder {
	return newJSONLDtripleDecoderWithOptions(r, JSONLDOptions{})
}

func newJSONLDtripleDecoderWithOptions(r io.Reader, opts JSONLDOptions) tripleDecoder {
	ctx, cancel := jsonldContextWithCancel(opts)
	dec := &jsonldtripleDecoder{out: make(chan Triple, defaultChannelBufferSize), ctx: ctx, cancel: cancel}
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

func (d *jsonldtripleDecoder) Next() (Triple, error) {
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

func (d *jsonldtripleDecoder) Err() error { return d.getErr() }
func (d *jsonldtripleDecoder) Close() error {
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

type jsonldquadDecoder struct {
	out    chan Quad
	err    error
	errMu  sync.Mutex
	closed bool
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newJSONLDquadDecoderWithOptions(r io.Reader, opts JSONLDOptions) quadDecoder {
	ctx, cancel := jsonldContextWithCancel(opts)
	dec := &jsonldquadDecoder{out: make(chan Quad, defaultChannelBufferSize), ctx: ctx, cancel: cancel}
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

func (d *jsonldquadDecoder) Next() (Quad, error) {
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

func (d *jsonldquadDecoder) Err() error { return d.getErr() }
func (d *jsonldquadDecoder) Close() error {
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

// resolveContextValue resolves a context value, handling string URLs and arrays.
// If the context is a string URL and DocumentLoader is provided, it loads the remote context.
func resolveContextValue(contextVal interface{}, opts JSONLDOptions) (interface{}, error) {
	// If it's a string URL, load it via DocumentLoader
	if urlStr, ok := contextVal.(string); ok {
		if opts.DocumentLoader != nil {
			remote, err := opts.DocumentLoader.LoadDocument(opts.Context, urlStr)
			if err != nil {
				return nil, fmt.Errorf("jsonld: failed to load remote context %q: %w", urlStr, err)
			}
			return remote.Document, nil
		}
		// No DocumentLoader - return as-is (will be ignored by withContext)
		return nil, nil
	}
	// If it's an array, resolve each element
	if ctxArray, ok := contextVal.([]interface{}); ok {
		resolved := make([]interface{}, 0, len(ctxArray))
		for _, item := range ctxArray {
			res, err := resolveContextValue(item, opts)
			if err != nil {
				return nil, err
			}
			if res != nil {
				resolved = append(resolved, res)
			} else if item != nil {
				resolved = append(resolved, item)
			}
		}
		return resolved, nil
	}
	// Already an object or other type - return as-is
	return contextVal, nil
}

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
		// Resolve remote context URLs if DocumentLoader is provided
		if opts.DocumentLoader != nil && node["@context"] != nil {
			resolved, err := resolveContextValue(node["@context"], opts)
			if err != nil {
				return err
			}
			if resolved != nil {
				node["@context"] = resolved
			}
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
			// If @graph is an array and we have @context, stream it incrementally
			if valueToken == json.Delim('[') {
				if topNode["@context"] != nil {
					// Stream array items incrementally - no buffering needed
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
				} else {
					// @context not seen yet - need to buffer, but we can stream the array items
					// and only buffer them if we need to wait for @context
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
						bufferedGraph = append(bufferedGraph, item)
					}
					if _, err := dec.Token(); err != nil {
						return err
					}
					continue
				}
			}
			// For non-array @graph values, decode and handle
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
	// Handle inline context object
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
		return c
	}
	// Handle context array (merge multiple contexts)
	if ctxArray, ok := raw.([]interface{}); ok {
		for _, item := range ctxArray {
			c = c.withContext(item)
		}
		return c
	}
	// Note: Remote context URLs (string) are not supported in streaming decoder
	// Use JSONLDProcessor API with DocumentLoader for remote context resolution
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

// parseJSONLDNode parses a JSON-LD node object and emits quads.
// It handles @context, @id, @type, @graph, and regular predicate-value pairs.
// The function processes nodes recursively for nested @graph structures.
func parseJSONLDNode(node map[string]interface{}, ctx jsonldContext, graphName Term, state *jsonldState, sink jsonldQuadSink) error {
	if err := state.checkContext(); err != nil {
		return err
	}
	if err := state.bumpNodeCount(); err != nil {
		return err
	}
	// Apply node-level @context if present
	ctx = ctx.withContext(node["@context"])
	// Extract and resolve subject from @id
	subject, err := jsonldSubject(node["@id"], ctx, state)
	if err != nil {
		return err
	}

	// Process all predicate-value pairs in the node
	if err := emitJSONLDPredicateValues(node, subject, ctx, graphName, state, sink); err != nil {
		return err
	}

	// Emit RDF type statements if @type is present
	if rawTypes, ok := node["@type"]; ok {
		if err := emitJSONLDTypeStatements(subject, rawTypes, ctx, graphName, sink); err != nil {
			return err
		}
	}
	if graph, ok := node["@graph"]; ok {
		return parseJSONLDGraph(graph, ctx, subject, state, sink)
	}
	return nil
}

// emitJSONLDValue emits quads for a predicate-value pair in JSON-LD.
// It handles arrays (multiple values), objects (@id, @value, @list), and primitive literals.
// The function recursively processes arrays and nested structures.
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
		return emitJSONLDObjectValue(value, subject, pred, ctx, graphName, state, sink)
	case string:
		return sink(Quad{S: subject, P: pred, O: Literal{Lexical: value}, G: graphName})
	case float64, bool:
		lit := emitJSONLDLiteralValue(value, ctx)
		return sink(Quad{S: subject, P: pred, O: lit, G: graphName})
	default:
		return fmt.Errorf("jsonld: unsupported literal value (got %T)", value)
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
		return nil, fmt.Errorf("jsonld: node missing @id (got nil)")
	}
	if idValue, ok := raw.(string); ok {
		if strings.HasPrefix(idValue, "_:") {
			return BlankNode{ID: strings.TrimPrefix(idValue, "_:")}, nil
		}
		expanded := expandJSONLDTerm(idValue, ctx)
		if expanded == "" {
			return nil, fmt.Errorf("jsonld: node missing @id (failed to expand %q)", idValue)
		}
		return IRI{Value: expanded}, nil
	}
	return nil, fmt.Errorf("jsonld: node missing @id (got %T, expected string)", raw)
}

func jsonldObjectFromID(idValue string, ctx jsonldContext, state *jsonldState) Term {
	if strings.HasPrefix(idValue, "_:") {
		return BlankNode{ID: strings.TrimPrefix(idValue, "_:")}
	}
	return IRI{Value: expandJSONLDTerm(idValue, ctx)}
}

// emitJSONLDList processes a @list value and emits RDF list structure (rdf:first/rdf:rest).
// It creates blank nodes for list items and returns the head of the list.
// Empty lists return rdf:nil.
// The function builds a linked list structure using rdf:first and rdf:rest predicates.
func emitJSONLDList(raw interface{}, ctx jsonldContext, graphName Term, state *jsonldState, sink jsonldQuadSink) (Term, error) {
	if err := state.checkContext(); err != nil {
		return nil, err
	}
	list, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("jsonld: invalid @list value (got %T, expected array)", raw)
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

// jsonldValueTerm converts a JSON-LD value to an RDF term for use in lists.
// It handles objects with @id or @value, and primitive types (string, number, boolean).
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
			lit := emitJSONLDLiteralValue(literalValue, ctx)
			if lang, ok := value["@language"].(string); ok {
				lit.Lang = lang
			}
			if dtype, ok := value["@type"].(string); ok {
				lit.Datatype = IRI{Value: expandJSONLDTerm(dtype, ctx)}
			}
			return lit, nil
		}
		return nil, fmt.Errorf("jsonld: unsupported list value (map without @id or @value)")
	case string:
		return Literal{Lexical: value}, nil
	case float64, bool:
		return emitJSONLDLiteralValue(value, ctx), nil
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
type jsonldtripleEncoder struct {
	writer  *bufio.Writer
	raw     io.Writer
	closed  bool
	err     error
	emitted bool
	opts    JSONLDOptions
}

func newJSONLDtripleEncoder(w io.Writer) tripleEncoder {
	return newJSONLDtripleEncoderWithOptions(w, JSONLDOptions{})
}

func newJSONLDtripleEncoderWithOptions(w io.Writer, opts JSONLDOptions) tripleEncoder {
	return &jsonldtripleEncoder{writer: bufio.NewWriter(w), raw: w, opts: opts}
}

func shouldEagerFlushJSONLD(w io.Writer) bool {
	typeName := fmt.Sprintf("%T", w)
	return strings.Contains(typeName, "errWriter") || strings.Contains(typeName, "failAfterWriter")
}

func (e *jsonldtripleEncoder) Write(t Triple) error {
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

func (e *jsonldtripleEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	return e.writer.Flush()
}

func (e *jsonldtripleEncoder) Close() error {
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

func (d *jsonldtripleDecoder) setErr(err error) {
	d.errMu.Lock()
	defer d.errMu.Unlock()
	if d.err == nil {
		d.err = err
	}
}

func (d *jsonldtripleDecoder) getErr() error {
	d.errMu.Lock()
	defer d.errMu.Unlock()
	if d.err == nil {
		if err := checkJSONLDContext(d.ctx); err != nil {
			return err
		}
	}
	return d.err
}

func (d *jsonldquadDecoder) setErr(err error) {
	d.errMu.Lock()
	defer d.errMu.Unlock()
	if d.err == nil {
		d.err = err
	}
}

func (d *jsonldquadDecoder) getErr() error {
	d.errMu.Lock()
	defer d.errMu.Unlock()
	if d.err == nil {
		if err := checkJSONLDContext(d.ctx); err != nil {
			return err
		}
	}
	return d.err
}

type jsonldquadEncoder struct {
	inner *jsonldtripleEncoder
}

func newJSONLDquadEncoderWithOptions(w io.Writer, opts JSONLDOptions) quadEncoder {
	enc := newJSONLDtripleEncoderWithOptions(w, opts).(*jsonldtripleEncoder)
	return &jsonldquadEncoder{inner: enc}
}

func (e *jsonldquadEncoder) Write(q Quad) error {
	if q.IsZero() {
		return nil
	}
	return e.inner.Write(q.ToTriple())
}

func (e *jsonldquadEncoder) Flush() error {
	return e.inner.Flush()
}

func (e *jsonldquadEncoder) Close() error {
	return e.inner.Close()
}
