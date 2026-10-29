package postgres

// wktArg turns a WKT geometry into a query argument, mapping an empty value to
// NULL so nullable geometry columns can be left unset.
func wktArg(b []byte) *string {
	if len(b) == 0 {
		return nil
	}
	s := string(b)
	return &s
}

// wktBytes is the inverse of wktArg for values read with ST_AsText.
func wktBytes(s *string) []byte {
	if s == nil || *s == "" {
		return nil
	}
	return []byte(*s)
}
