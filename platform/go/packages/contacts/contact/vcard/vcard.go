// Package vcard maps a contact.Contact to and from a vCard 4.0 record (RFC 6350).
// It is HAND-ROLLED from the stdlib (strings/bufio) — no external vCard
// dependency — and is PURE: no I/O, no clock, no storage. It imports pkg/contact
// (and its value-object primitives) but never modifies them.
//
// Round-trip is the contract: FromVCARD(ToVCARD(c)) reproduces every wire-relevant
// field. The full field set is:
//
//   - structured name (N), formatted name (FN), email (EMAIL), phone (TEL),
//     postal address (ADR), birthday (BDAY), categories (CATEGORIES), notes (NOTE),
//     and social profiles (X-SBX-SOCIAL + interop URL).
//   - Latitude + Longitude → the standard GEO property
//     (RFC 6350 §6.5.2: GEO:geo:<lat>,<lng>). The floats are formatted with
//     strconv.FormatFloat(f,'f',-1,64) so they round-trip bit-exact; GEO is omitted
//     when both are zero. GEO is a sibling of ADR, not a component of it — a contact
//     may carry coordinates with no postal address and vice versa.
//   - SubDistrict → X-SBX-SUBDISTRICT (Thai ตำบล / taluk / barangay — there is no
//     standard ADR component for the level below locality).
//   - Nationality → X-SBX-NATIONALITY (no standard vCard property exists).
//   - PhotoID (*string) → X-SBX-PHOTO-ID (the UUID ref to the stored image, NOT the
//     image bytes; omitted when nil and parsed back to a *string).
//   - Data (ExtensionData, a JSONB map) → X-SBX-DATA carrying the map as compact
//     JSON (the value is then TEXT-escaped). The JSON bytes round-trip losslessly;
//     callers comparing Go-side must normalize (a JSON number decodes as float64),
//     so the wire form — not reflect.DeepEqual on the map — is the equality the sync
//     layer relies on. Omitted when the map is empty.
//
// Child-note identity (Note.ID, Note.ContactID, Note.CreatedAt) is NOT part of the
// vCard contract — only the NOTE text is; on import the joined NOTE becomes a single
// Note body. The Contact Document base (ID, CreatedAt, UpdatedAt, DeletedAt)
// likewise has no vCard slot and is reconciled by the sync layer out of band. These
// exclusions are intentional, never silent.
//
// Social profiles are the subtle part. RFC 6350 has no first-class "social handle"
// property, and a contact social may carry only a platform+handle (no URL). To
// round-trip losslessly the mapper writes a custom X-SBX-SOCIAL line
// (X-SBX-SOCIAL;TYPE=<platform>:<handle-or-url>) for EVERY social, and additionally
// a standard URL line for socials that have a URL (so foreign apps still see a
// clickable profile). RFC 6350 §6.10 permits experimental X- properties, so the
// output stays valid. On parse the X- lines are authoritative; bare URL lines
// without a matching X- line are ignored (they are the interoperability duplicate).
//
// Line folding (<=75 octets, CRLF + leading space) and TEXT escaping (backslash,
// comma, semicolon, newline) follow RFC 6350 §3.2 / §3.4 — identical mechanics to
// iCalendar — and the parser unfolds + unescapes symmetrically.
package vcard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/shredbx/sbx-core/pkg/contact"
	"github.com/shredbx/sbx-core/pkg/socialnetwork"
)

// crlf is the mandatory line break between content lines (RFC 6350 §3.2).
const crlf = "\r\n"

// maxOctets is the soft line-length limit for folding (RFC 6350 §3.2).
const maxOctets = 75

// =============================================================================
// MARSHAL — Contact -> vCard
// =============================================================================

// ToVCARD marshals a contact into a vCard 4.0 record. The output ends with a
// trailing CRLF. Optional fields with no value are omitted entirely.
func ToVCARD(c contact.Contact) string {
	var b strings.Builder
	writeLine(&b, "BEGIN:VCARD")
	writeLine(&b, "VERSION:4.0")

	// FN (formatted name) is REQUIRED in vCard 4.0 (§6.2.1). Build it from the
	// honorific prefix + the full name so the prefix is not lost in FN.
	writeProp(&b, "FN", formattedName(c))

	// N (structured name): Surname;Given;Additional(middle);Prefix(title);Suffix.
	// Each component is escaped individually; the ';' separators are structural.
	writeStructured(&b, "N", "", []string{c.Surname, c.GivenName, c.MiddleName, c.Title, ""})

	if c.Email != "" {
		writeProp(&b, "EMAIL", c.Email)
	}

	if !c.Phone().IsZero() {
		// TEL value carries the country code and number separated by a single space
		// (phonenumber.Format()). The space is an unambiguous, RFC-valid boundary
		// that makes the country-code/number split lossless on parse — concatenating
		// them would make the boundary unrecoverable (e.g. "+66" vs "+668"). When the
		// country code is empty the value is just the number (no leading space).
		params := ""
		if c.PhoneType != "" {
			params = "TYPE=" + escapeParam(c.PhoneType)
		}
		writeRawProp(&b, "TEL", params, telValue(c))
	}

	if hasPostal(c) {
		// ADR: po-box;extended(unit);street;locality(city);region(province);postal;country.
		// Gated on the postal components only — coordinates ride on GEO below, so a
		// coordinate-only contact does NOT emit an all-empty ADR.
		writeStructured(&b, "ADR", "TYPE=home", []string{
			"", c.Unit, c.Street, c.City, c.Province, c.PostalCode, c.Country,
		})
	}

	// SubDistrict has no standard ADR component (it sits below locality) — carry it
	// in a custom property so it is not lost between street and city.
	if c.SubDistrict != "" {
		writeProp(&b, "X-SBX-SUBDISTRICT", c.SubDistrict)
	}

	// GEO (standard, RFC 6350 §6.5.2) — a sibling of ADR. FormatFloat with precision
	// -1 emits the shortest exact decimal so the float round-trips bit-exact. Omitted
	// when both coordinates are zero (no spurious GEO:geo:0,0).
	if c.Latitude != 0 || c.Longitude != 0 {
		writeRawProp(&b, "GEO", "", "geo:"+formatCoord(c.Latitude)+","+formatCoord(c.Longitude))
	}

	if c.DateOfBirth != "" {
		writeProp(&b, "BDAY", c.DateOfBirth)
	}

	if c.Nationality != "" {
		writeProp(&b, "X-SBX-NATIONALITY", c.Nationality)
	}

	if len(c.CategoryCodes) > 0 {
		// CATEGORIES is a comma-separated TEXT list (§6.7.1) — each code escaped,
		// joined with literal commas (the list separator).
		escaped := make([]string, 0, len(c.CategoryCodes))
		for _, code := range c.CategoryCodes {
			escaped = append(escaped, escapeText(code))
		}
		writeFolded(&b, "CATEGORIES:"+strings.Join(escaped, ","))
	}

	for _, sn := range c.SocialNetworks {
		if sn.IsZero() {
			continue
		}
		// Lossless: platform in TYPE, handle (or URL when handle is empty) in value.
		val := sn.Handle
		if val == "" {
			val = sn.URL
		}
		writeRawProp(&b, "X-SBX-SOCIAL", "TYPE="+escapeParam(sn.Platform), escapeText(val))
		if sn.URL != "" {
			// Interop duplicate: a standard URL line foreign apps understand.
			writeRawProp(&b, "URL", "TYPE="+escapeParam(sn.Platform), sn.URL)
		}
	}

	// PhotoID → the UUID ref (not the image). Omitted when nil so a contact with no
	// photo round-trips back to a nil pointer.
	if c.PhotoID != nil {
		writeProp(&b, "X-SBX-PHOTO-ID", *c.PhotoID)
	}

	// Data (JSONB map) → compact JSON in a custom property, then TEXT-escaped. The
	// wire bytes round-trip losslessly; omitted when the map is empty.
	if len(c.Data) > 0 {
		if js := marshalData(c.Data); js != "" {
			writeProp(&b, "X-SBX-DATA", js)
		}
	}

	if note := joinNotes(c); note != "" {
		writeProp(&b, "NOTE", note)
	}

	writeLine(&b, "END:VCARD")
	return b.String()
}

// formattedName builds the FN value: "Title Given Middle Surname", omitting any
// empty part. Falls back to the contact's DisplayName when no parts are set.
func formattedName(c contact.Contact) string {
	parts := make([]string, 0, 4)
	for _, p := range []string{c.Title, c.GivenName, c.MiddleName, c.Surname} {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return c.DisplayName()
	}
	return strings.Join(parts, " ")
}

// joinNotes concatenates the bodies of a contact's ordered notes with a newline.
// The single combined NOTE round-trips back to one Note on import.
func joinNotes(c contact.Contact) string {
	if len(c.Notes) == 0 {
		return ""
	}
	parts := make([]string, 0, len(c.Notes))
	for _, n := range c.Notes {
		parts = append(parts, n.Body)
	}
	return strings.Join(parts, "\n")
}

// =============================================================================
// UNMARSHAL — vCard -> Contact
// =============================================================================

// FromVCARD parses a single vCard 4.0 record into a Contact. BEGIN/END:VCARD and
// VERSION are required. Unknown properties are ignored.
func FromVCARD(s string) (contact.Contact, error) {
	lines, err := unfold(s)
	if err != nil {
		return contact.Contact{}, err
	}

	var c contact.Contact
	var haveBegin, haveEnd, haveVersion bool

	for _, line := range lines {
		name, params, value := splitContentLine(line)
		switch name {
		case "BEGIN":
			if strings.EqualFold(value, "VCARD") {
				haveBegin = true
			}
		case "END":
			if strings.EqualFold(value, "VCARD") {
				haveEnd = true
			}
		case "VERSION":
			haveVersion = true
		case "FN":
			// FN is derived from N on export; nothing to store separately. The
			// structured N below is authoritative for the name parts.
		case "N":
			// Surname;Given;Additional;Prefix;Suffix
			comps := splitStructured(value)
			c.Surname = comp(comps, 0)
			c.GivenName = comp(comps, 1)
			c.MiddleName = comp(comps, 2)
			c.Title = comp(comps, 3)
		case "EMAIL":
			c.Email = unescapeText(value)
		case "TEL":
			cc, num := splitPhone(value)
			c.PhoneCountryCode = cc
			c.PhoneNumber = num
			if tp := params["TYPE"]; tp != "" {
				c.PhoneType = tp
			}
		case "ADR":
			comps := splitStructured(value)
			// po;ext(unit);street;locality(city);region(province);postal;country
			c.Unit = comp(comps, 1)
			c.Street = comp(comps, 2)
			c.City = comp(comps, 3)
			c.Province = comp(comps, 4)
			c.PostalCode = comp(comps, 5)
			c.Country = comp(comps, 6)
		case "X-SBX-SUBDISTRICT":
			c.SubDistrict = unescapeText(value)
		case "GEO":
			lat, lng, perr := parseGeo(value)
			if perr != nil {
				return contact.Contact{}, fmt.Errorf("vcard: GEO: %w", perr)
			}
			c.Latitude = lat
			c.Longitude = lng
		case "BDAY":
			c.DateOfBirth = unescapeText(value)
		case "X-SBX-NATIONALITY":
			c.Nationality = unescapeText(value)
		case "X-SBX-PHOTO-ID":
			id := unescapeText(value)
			c.PhotoID = &id
		case "X-SBX-DATA":
			data, perr := unmarshalData(unescapeText(value))
			if perr != nil {
				return contact.Contact{}, fmt.Errorf("vcard: X-SBX-DATA: %w", perr)
			}
			c.Data = data
		case "CATEGORIES":
			c.CategoryCodes = splitCategories(value)
		case "X-SBX-SOCIAL":
			sn := socialnetwork.SocialNetwork{
				Platform: params["TYPE"],
				Handle:   unescapeText(value),
			}
			c.SocialNetworks = append(c.SocialNetworks, sn)
		case "URL":
			// Interop duplicate of X-SBX-SOCIAL: attach the URL to the matching
			// social (same platform) if one was parsed; otherwise ignore (the X-
			// line is authoritative for the social set).
			attachURL(&c, params["TYPE"], value)
		case "NOTE":
			body := unescapeText(value)
			if body != "" {
				c.Notes = append(c.Notes, contact.Note{Body: body})
			}
		}
	}

	if !haveBegin || !haveEnd {
		return contact.Contact{}, fmt.Errorf("vcard: missing BEGIN/END:VCARD")
	}
	if !haveVersion {
		return contact.Contact{}, fmt.Errorf("vcard: missing required VERSION")
	}
	return c, nil
}

// attachURL sets the URL on the most recent parsed social whose platform matches
// typ and whose URL is still empty. A no-op when no such social exists.
func attachURL(c *contact.Contact, typ, url string) {
	for i := range c.SocialNetworks {
		if c.SocialNetworks[i].Platform == typ && c.SocialNetworks[i].URL == "" {
			c.SocialNetworks[i].URL = url
			return
		}
	}
}

// telValue formats the TEL value as "<country code> <number>" (a single space
// boundary), or just the number when the country code is empty. Mirrors
// phonenumber.Format() but tolerates a missing country code (lenient contacts).
func telValue(c contact.Contact) string {
	cc := strings.TrimSpace(c.PhoneCountryCode)
	num := strings.TrimSpace(c.PhoneNumber)
	if cc == "" {
		return num
	}
	return cc + " " + num
}

// splitPhone reverses telValue: it splits a TEL value on the FIRST space into the
// country code and the number. A value with no space is returned wholly as the
// number (country code empty) — matching the lenient contact phone convention and
// making the round-trip of telValue() exact.
func splitPhone(v string) (countryCode, number string) {
	v = strings.TrimSpace(v)
	if sp := strings.IndexByte(v, ' '); sp >= 0 {
		return v[:sp], strings.TrimSpace(v[sp+1:])
	}
	return "", v
}

// =============================================================================
// GEO, EXTENSION DATA — non-text custom carriers
// =============================================================================

// hasPostal reports whether any postal ADR component is set. It gates the ADR line
// independently of coordinates (which ride on GEO), so a coordinate-only contact
// does not emit an all-empty ADR that would parse back as no address.
func hasPostal(c contact.Contact) bool {
	return c.Unit != "" || c.Street != "" || c.City != "" ||
		c.Province != "" || c.PostalCode != "" || c.Country != ""
}

// formatCoord renders a coordinate as the shortest decimal that parses back to the
// exact same float64 (FormatFloat 'f' with precision -1, RFC 6350 GEO uses a plain
// decimal). This is what makes GEO a bit-exact round-trip.
func formatCoord(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// parseGeo parses an RFC 6350 GEO value "geo:<lat>,<lng>" back into the coordinate
// pair. The "geo:" URI scheme prefix is optional (lenient toward foreign emitters);
// both components must be valid floats.
func parseGeo(v string) (lat, lng float64, err error) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "geo:")
	parts := strings.SplitN(v, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("want geo:<lat>,<lng>, got %q", v)
	}
	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("latitude %q: %w", parts[0], err)
	}
	lng, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("longitude %q: %w", parts[1], err)
	}
	return lat, lng, nil
}

// marshalData serializes the ExtensionData map to compact JSON for an X-SBX-DATA
// value. json.Marshal sorts map keys, so the output is stable across runs (the
// round-trip is byte-stable for a given map). An empty map yields "".
func marshalData(d contact.ExtensionData) string {
	if len(d) == 0 {
		return ""
	}
	b, err := json.Marshal(d)
	if err != nil {
		// ExtensionData holds arbitrary values; an unmarshalable value (e.g. a
		// channel injected by a caller) is a programming error, not wire input.
		return ""
	}
	return string(b)
}

// unmarshalData reverses marshalData. An empty/blank value yields a nil map (not an
// error); malformed JSON is reported so a corrupt X-SBX-DATA surfaces.
func unmarshalData(s string) (contact.ExtensionData, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var d contact.ExtensionData
	if err := json.Unmarshal([]byte(s), &d); err != nil {
		return nil, err
	}
	return d, nil
}

// =============================================================================
// STRUCTURED VALUES — component lists separated by unescaped ';'
// =============================================================================

// writeStructured writes "NAME[;params]:c1;c2;c3" with each component escaped and
// the ';' separators structural. Trailing empty components are kept (positional).
func writeStructured(b *strings.Builder, name, params string, comps []string) {
	escaped := make([]string, len(comps))
	for i, c := range comps {
		escaped[i] = escapeComponent(c)
	}
	head := name
	if params != "" {
		head += ";" + params
	}
	writeFolded(b, head+":"+strings.Join(escaped, ";"))
}

// splitStructured splits a structured value on UNESCAPED semicolons into its
// positional components, unescaping each.
func splitStructured(v string) []string {
	raw := splitUnescaped(v, ';')
	out := make([]string, len(raw))
	for i, r := range raw {
		out[i] = unescapeText(r)
	}
	return out
}

// splitCategories splits a CATEGORIES value on UNESCAPED commas, unescaping each
// and dropping empty entries.
func splitCategories(v string) []string {
	raw := splitUnescaped(v, ',')
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		s := unescapeText(r)
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

// comp safely indexes a component slice, returning "" when i is out of range.
func comp(comps []string, i int) string {
	if i < 0 || i >= len(comps) {
		return ""
	}
	return comps[i]
}

// =============================================================================
// CONTENT-LINE WRITERS (shared mechanics with RFC 5545 folding/escaping)
// =============================================================================

func writeLine(b *strings.Builder, line string) {
	b.WriteString(line)
	b.WriteString(crlf)
}

// writeProp writes "NAME:<escaped value>" and folds the line.
func writeProp(b *strings.Builder, name, value string) {
	writeFolded(b, name+":"+escapeText(value))
}

// writeRawProp writes "NAME[;params]:<value>" without escaping the value (caller
// pre-escapes when needed) and folds the line. An empty params is omitted.
func writeRawProp(b *strings.Builder, name, params, value string) {
	head := name
	if params != "" {
		head += ";" + params
	}
	writeFolded(b, head+":"+value)
}

// writeFolded folds a single assembled content line to <=75 octets (RFC 6350
// §3.2): a CRLF is inserted and each continuation begins with a single space. The
// fold never splits a multi-byte UTF-8 sequence.
func writeFolded(b *strings.Builder, line string) {
	if len(line) <= maxOctets {
		b.WriteString(line)
		b.WriteString(crlf)
		return
	}
	bytes := []byte(line)
	first := true
	for len(bytes) > 0 {
		limit := maxOctets
		prefix := ""
		if !first {
			prefix = " "
			limit = maxOctets - 1
		}
		cut := limit
		if cut > len(bytes) {
			cut = len(bytes)
		} else {
			for cut > 0 && bytes[cut]&0xC0 == 0x80 {
				cut--
			}
		}
		b.WriteString(prefix)
		b.Write(bytes[:cut])
		b.WriteString(crlf)
		bytes = bytes[cut:]
		first = false
	}
}

// =============================================================================
// CONTENT-LINE PARSING
// =============================================================================

// unfold normalizes line endings and unfolds continuation lines (a line starting
// with space or tab continues the previous, RFC 6350 §3.2). Returns the logical
// lines with no trailing CRLF.
func unfold(s string) ([]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	var logical []string
	for scanner.Scan() {
		raw := strings.TrimSuffix(scanner.Text(), "\r")
		if raw == "" {
			continue
		}
		if (raw[0] == ' ' || raw[0] == '\t') && len(logical) > 0 {
			logical[len(logical)-1] += raw[1:]
			continue
		}
		logical = append(logical, raw)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("vcard: scan: %w", err)
	}
	return logical, nil
}

// splitContentLine splits "NAME[;PARAM=VALUE…]:VALUE" into name, params, value.
// The name/value split is the first UNQUOTED colon.
func splitContentLine(line string) (name string, params map[string]string, value string) {
	params = map[string]string{}
	colon := indexUnquoted(line, ':')
	if colon < 0 {
		return strings.ToUpper(line), params, ""
	}
	head := line[:colon]
	value = line[colon+1:]

	// A property may have a group prefix "group.NAME" (RFC 6350 §3.3) — drop it.
	parts := strings.Split(head, ";")
	rawName := parts[0]
	if dot := strings.LastIndex(rawName, "."); dot >= 0 {
		rawName = rawName[dot+1:]
	}
	name = strings.ToUpper(rawName)
	for _, p := range parts[1:] {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			params[strings.ToUpper(kv[0])] = strings.Trim(kv[1], `"`)
		}
	}
	return name, params, value
}

// indexUnquoted returns the index of the first sep outside a double-quoted run.
func indexUnquoted(s string, sep byte) int {
	inQuote := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuote = !inQuote
		case sep:
			if !inQuote {
				return i
			}
		}
	}
	return -1
}

// splitUnescaped splits s on every occurrence of sep that is NOT preceded by an
// (unescaped) backslash. A doubled backslash (\\) does not escape the separator.
func splitUnescaped(s string, sep byte) []string {
	var out []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			// Keep the escape pair verbatim; unescapeText decodes it later.
			cur.WriteByte(s[i])
			cur.WriteByte(s[i+1])
			i++
			continue
		}
		if s[i] == sep {
			out = append(out, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(s[i])
	}
	out = append(out, cur.String())
	return out
}

// =============================================================================
// TEXT ESCAPING — RFC 6350 §3.4 (identical to RFC 5545 TEXT)
// =============================================================================

// escapeText escapes a TEXT value: backslash, newline, comma, semicolon. Used for
// scalar values (and list/structured components, which additionally treat the
// list/component separators as literal via escapeComponent below).
func escapeText(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		";", `\;`,
		",", `\,`,
		"\r\n", `\n`,
		"\n", `\n`,
		"\r", `\n`,
	)
	return r.Replace(s)
}

// escapeComponent escapes a structured-value component. Identical to escapeText —
// the ';' (component sep) and ',' (value list sep) are both escaped so any literal
// occurrence inside a component survives.
func escapeComponent(s string) string { return escapeText(s) }

// escapeParam escapes a parameter value minimally: a literal ';' or ',' or ':' in
// a param value would otherwise break parsing. Backslash-escape them; this keeps
// the writer's own output round-trippable. (Full RFC 6350 param quoting is broader
// but unnecessary for our controlled inputs.)
func escapeParam(s string) string {
	r := strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, ":", `\:`)
	return r.Replace(s)
}

// unescapeText reverses escapeText/escapeComponent. An escaped backslash (\\) is
// consumed as one backslash; \n and \N decode to a newline.
func unescapeText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n', 'N':
			b.WriteByte('\n')
		case '\\', ';', ',', ':':
			b.WriteByte(s[i])
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
