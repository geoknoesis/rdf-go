package rdf

func (c *turtleCursor) ensureLineEnd() error {
	c.skipWS()
	if c.pos < len(c.input) {
		return c.errorf("unexpected content after '.'")
	}
	return nil
}
