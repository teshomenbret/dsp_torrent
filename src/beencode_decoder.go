package leecher

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type structBuilder struct {
	val  reflect.Value
	map_ reflect.Value
	key  reflect.Value
}

var nobuilder *structBuilder
var bufioReaderPool sync.Pool

func isfloat(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func setfloat(v reflect.Value, f float64) {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		v.SetFloat(f)
	}
}

func setint(val reflect.Value, i int64) {
	switch v := val; v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(i))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v.SetUint(uint64(i))
	case reflect.Interface:
		v.Set(reflect.ValueOf(i))
	default:
		panic("setint called for bogus type: " + val.Kind().String())
	}
}

// If updating b.val is not enough to update the original,
// copy a changed b.val out to the original.
func (b *structBuilder) Flush() {
	if b == nil {
		return
	}
	if b.map_.IsValid() {
		b.map_.SetMapIndex(b.key, b.val)
	}
}

func (b *structBuilder) Int64(i int64) {
	if b == nil {
		return
	}
	if !b.val.CanSet() {
		x := 0
		b.val = reflect.ValueOf(&x).Elem()
	}
	v := b.val
	if isfloat(v) {
		setfloat(v, float64(i))
	} else {
		setint(v, i)
	}
}

func (b *structBuilder) Uint64(i uint64) {
	if b == nil {
		return
	}
	if !b.val.CanSet() {
		x := uint64(0)
		b.val = reflect.ValueOf(&x).Elem()
	}
	v := b.val
	if isfloat(v) {
		setfloat(v, float64(i))
	} else {
		setint(v, int64(i))
	}
}

func (b *structBuilder) Float64(f float64) {
	if b == nil {
		return
	}
	if !b.val.CanSet() {
		x := float64(0)
		b.val = reflect.ValueOf(&x).Elem()
	}
	v := b.val
	if isfloat(v) {
		setfloat(v, f)
	} else {
		setint(v, int64(f))
	}
}

func (b *structBuilder) String(s string) {
	if b == nil {
		return
	}

	switch b.val.Kind() {
	case reflect.String:
		if !b.val.CanSet() {
			x := ""
			b.val = reflect.ValueOf(&x).Elem()

		}
		b.val.SetString(s)
	case reflect.Interface:
		b.val.Set(reflect.ValueOf(s))
	}
}

func (b *structBuilder) Array() {
	if b == nil {
		return
	}
	if v := b.val; v.Kind() == reflect.Slice {
		if v.IsNil() {
			v.Set(reflect.MakeSlice(v.Type(), 0, 8))
		}
	}
}

func (b *structBuilder) Elem(i int) builder {
	if b == nil || i < 0 {
		return nobuilder
	}
	switch v := b.val; v.Kind() {
	case reflect.Array:
		if i < v.Len() {
			return &structBuilder{val: v.Index(i)}
		}
	case reflect.Slice:
		if i >= v.Cap() {
			n := v.Cap()
			if n < 8 {
				n = 8
			}
			for n <= i {
				n *= 2
			}
			nv := reflect.MakeSlice(v.Type(), v.Len(), n)
			reflect.Copy(nv, v)
			v.Set(nv)
		}
		if v.Len() <= i && i < v.Cap() {
			v.SetLen(i + 1)
		}
		if i < v.Len() {
			return &structBuilder{val: v.Index(i)}
		}
	}
	return nobuilder
}

func (b *structBuilder) Map() {
	if b == nil {
		return
	}
	if v := b.val; v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.Zero(v.Type().Elem()).Addr())
			b.Flush()
		}
		b.map_ = reflect.Value{}
		b.val = v.Elem()
	}
	if v := b.val; v.Kind() == reflect.Map && v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}
}

func (b *structBuilder) Key(k string) builder {
	if b == nil {
		return nobuilder
	}
	switch v := reflect.Indirect(b.val); v.Kind() {
	case reflect.Struct:
		t := v.Type()
		// Case-insensitive field lookup.
		k = strings.ToLower(k)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			key := bencodeKey(field, nil)
			if strings.ToLower(key) == k ||
				strings.ToLower(field.Name) == k {
				return &structBuilder{val: v.Field(i)}
			}
		}
	case reflect.Map:
		t := v.Type()
		if t.Key() != reflect.TypeOf(k) {
			break
		}
		key := reflect.ValueOf(k)
		elem := v.MapIndex(key)
		if !elem.IsValid() {
			v.SetMapIndex(key, reflect.Zero(t.Elem()))
			elem = v.MapIndex(key)
		}
		return &structBuilder{val: elem, map_: v, key: key}
	}
	return nobuilder
}

func UnmarshalResponse(r io.Reader, val interface{}) (err error) {
	// If e represents a value, the answer won't get back to the
	// caller.  Make sure it's a pointer.
	if reflect.TypeOf(val).Kind() != reflect.Ptr {
		err = errors.New("attempt to unmarshal into a non-pointer")
		return
	}
	// err = unmarshalValue(r, reflect.Indirect(reflect.ValueOf(val)))

	v := reflect.Indirect(reflect.ValueOf(val))
	var b *structBuilder

	// XXX: Decide if the extra codnitions are needed. Affect map?
	if ptr := v; ptr.Kind() == reflect.Ptr {
		if slice := ptr.Elem(); slice.Kind() == reflect.Slice || slice.Kind() == reflect.Int || slice.Kind() == reflect.String {
			b = &structBuilder{val: slice}
		}
	}

	if b == nil {
		b = &structBuilder{val: v}
	}

	err = parse(r, b)

	return
}

type MarshalError struct {
	T reflect.Type
}

func (e *MarshalError) Error() string {
	return "bencode cannot encode value of type " + e.T.String()
}

func writeArrayOrSlice(w io.Writer, val reflect.Value) (err error) {
	_, err = fmt.Fprint(w, "l")
	if err != nil {
		return
	}
	for i := 0; i < val.Len(); i++ {
		if err := writeValue(w, val.Index(i)); err != nil {
			return err
		}
	}

	_, err = fmt.Fprint(w, "e")
	if err != nil {
		return
	}
	return nil
}

type stringValue struct {
	key       string
	value     reflect.Value
	omitEmpty bool
}

type stringValueArray []stringValue

// Satisfy sort.Interface

func (a stringValueArray) Len() int { return len(a) }

func (a stringValueArray) Less(i, j int) bool { return a[i].key < a[j].key }

func (a stringValueArray) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func writeSVList(w io.Writer, svList stringValueArray) (err error) {
	sort.Sort(svList)

	for _, sv := range svList {
		if sv.isValueNil() {
			continue // Skip null values
		}
		s := sv.key
		_, err = fmt.Fprintf(w, "%d:%s", len(s), s)
		if err != nil {
			return
		}

		if err = writeValue(w, sv.value); err != nil {
			return
		}
	}
	return
}

func writeMap(w io.Writer, val reflect.Value) (err error) {
	key := val.Type().Key()
	if key.Kind() != reflect.String {
		return &MarshalError{val.Type()}
	}
	_, err = fmt.Fprint(w, "d")
	if err != nil {
		return
	}

	keys := val.MapKeys()

	// Sort keys

	svList := make(stringValueArray, len(keys))
	for i, key := range keys {
		svList[i].key = key.String()
		svList[i].value = val.MapIndex(key)
	}

	err = writeSVList(w, svList)
	if err != nil {
		return
	}

	_, err = fmt.Fprint(w, "e")
	if err != nil {
		return
	}
	return
}

func bencodeKey(field reflect.StructField, sv *stringValue) (key string) {
	key = field.Name
	tag := field.Tag
	if len(tag) > 0 {
		// Backwards compatability
		// If there's a bencode key/value entry in the tag, use it.
		var tagOpt tagOptions
		key, tagOpt = parseTag(tag.Get("bencode"))
		if len(key) == 0 {
			key = tag.Get("bencode")
			if len(key) == 0 && !strings.Contains(string(tag), ":") {
				// If there is no ":" in the tag, assume it is an old-style tag.
				key = string(tag)
			} else {
				key = field.Name
			}
		}
		if sv != nil && tagOpt.Contains("omitempty") {
			sv.omitEmpty = true
		}
	}
	if sv != nil {
		sv.key = key
	}
	return
}

type tagOptions string

func parseTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, tagOptions("")
}

func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}

func writeStruct(w io.Writer, val reflect.Value) (err error) {
	_, err = fmt.Fprint(w, "d")
	if err != nil {
		return
	}

	typ := val.Type()

	numFields := val.NumField()
	svList := make(stringValueArray, numFields)

	for i := 0; i < numFields; i++ {
		field := typ.Field(i)
		bencodeKey(field, &svList[i])
		if svList[i].key == "-" {
			svList[i].value = reflect.Value{}
		} else {
			svList[i].value = val.Field(i)
		}
	}

	err = writeSVList(w, svList)
	if err != nil {
		return
	}

	_, err = fmt.Fprint(w, "e")
	if err != nil {
		return
	}
	return
}

func writeValue(w io.Writer, val reflect.Value) (err error) {
	if !val.IsValid() {
		err = errors.New("can't write null value")
		return
	}

	switch v := val; v.Kind() {
	case reflect.String:
		s := v.String()
		_, err = fmt.Fprintf(w, "%d:%s", len(s), s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		_, err = fmt.Fprintf(w, "i%de", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		_, err = fmt.Fprintf(w, "i%de", v.Uint())
	case reflect.Array:
		err = writeArrayOrSlice(w, v)
	case reflect.Slice:
		switch val.Type().String() {
		case "[]uint8":
			// special case as byte-string
			s := string(v.Bytes())
			_, err = fmt.Fprintf(w, "%d:%s", len(s), s)
		default:
			err = writeArrayOrSlice(w, v)
		}
	case reflect.Map:
		err = writeMap(w, v)
	case reflect.Struct:
		err = writeStruct(w, v)
	case reflect.Interface:
		err = writeValue(w, v.Elem())
	default:
		err = &MarshalError{val.Type()}
	}
	return
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func (sv stringValue) isValueNil() bool {
	if !sv.value.IsValid() || (sv.omitEmpty && isEmptyValue(sv.value)) {
		return true
	}
	switch v := sv.value; v.Kind() {
	case reflect.Interface:
		return !v.Elem().IsValid()
	}
	return false
}

func DescodeMarshal(w io.Writer, val interface{}) error {
	return writeValue(w, reflect.ValueOf(val))
}

type builder interface {
	// Set value
	Int64(i int64)
	Uint64(i uint64)
	Float64(f float64)
	String(s string)
	Array()
	Map()

	// Create sub-Builders
	Elem(i int) builder
	Key(s string) builder
	Flush()
}

type Reader interface {
	io.Reader
	io.ByteScanner
}

func decodeInt64(r *bufio.Reader, delim byte) (data int64, err error) {
	buf, err := readSlice(r, delim)
	if err != nil {
		return
	}
	data, err = strconv.ParseInt(string(buf), 10, 64)
	return
}

// Read bytes up until delim, return slice without delimiter byte.
func readSlice(r *bufio.Reader, delim byte) (data []byte, err error) {
	if data, err = r.ReadSlice(delim); err != nil {
		return
	}
	lenData := len(data)
	if lenData > 0 {
		data = data[:lenData-1]
	} else {
		panic("bad r.ReadSlice() length")
	}
	return
}

func decodeString(r *bufio.Reader) (data string, err error) {
	length, err := decodeInt64(r, ':')
	if err != nil {
		return
	}
	if length < 0 {
		err = errors.New("bad string length")
		return
	}

	// Can we peek that much data out of r?
	if peekBuf, peekErr := r.Peek(int(length)); peekErr == nil {
		data = string(peekBuf)
		_, err = r.Discard(int(length))
		return
	}

	var buf = make([]byte, length)
	_, err = readFull(r, buf)
	if err != nil {
		return
	}
	data = string(buf)
	return
}

// Like io.ReadFull, but takes a bufio.Reader.
func readFull(r *bufio.Reader, buf []byte) (n int, err error) {
	return readAtLeast(r, buf, len(buf))
}

// Like io.ReadAtLeast, but takes a bufio.Reader.
func readAtLeast(r *bufio.Reader, buf []byte, min int) (n int, err error) {
	if len(buf) < min {
		return 0, io.ErrShortBuffer
	}
	for n < min && err == nil {
		var nn int
		nn, err = r.Read(buf[n:])
		n += nn
	}
	if n >= min {
		err = nil
	} else if n > 0 && err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return
}

func parseFromReader(r *bufio.Reader, build builder) (err error) {
	c, err := r.ReadByte()
	if err != nil {
		goto exit
	}
	switch {
	case c >= '0' && c <= '9':
		// String
		err = r.UnreadByte()
		if err != nil {
			goto exit
		}
		var str string
		str, err = decodeString(r)
		if err != nil {
			goto exit
		}
		build.String(str)

	case c == 'd':
		// dictionary

		build.Map()
		for {
			c, err = r.ReadByte()
			if err != nil {
				goto exit
			}
			if c == 'e' {
				break
			}
			err = r.UnreadByte()
			if err != nil {
				goto exit
			}
			var key string
			key, err = decodeString(r)
			if err != nil {
				goto exit
			}
			// TODO: in pendantic mode, check for keys in ascending order.
			err = parseFromReader(r, build.Key(key))
			if err != nil {
				goto exit
			}
		}

	case c == 'i':
		var buf []byte
		buf, err = readSlice(r, 'e')
		if err != nil {
			goto exit
		}
		var str string
		var i int64
		var i2 uint64
		var f float64
		str = string(buf)
		// If the number is exactly an integer, use that.
		if i, err = strconv.ParseInt(str, 10, 64); err == nil {
			build.Int64(i)
		} else if i2, err = strconv.ParseUint(str, 10, 64); err == nil {
			build.Uint64(i2)
		} else if f, err = strconv.ParseFloat(str, 64); err == nil {
			build.Float64(f)
		} else {
			err = errors.New("bad integer")
		}

	case c == 'l':
		// array
		build.Array()
		n := 0
		for {
			c, err = r.ReadByte()
			if err != nil {
				goto exit
			}
			if c == 'e' {
				break
			}
			err = r.UnreadByte()
			if err != nil {
				goto exit
			}
			err = parseFromReader(r, build.Elem(n))
			if err != nil {
				goto exit
			}
			n++
		}
	default:
		err = fmt.Errorf("unexpected character: '%v'", c)
	}
exit:
	build.Flush()
	return
}

func parse(reader io.Reader, builder builder) (err error) {
	r, ok := reader.(*bufio.Reader)
	if !ok {
		r = newBufioReader(reader)
		defer bufioReaderPool.Put(r)
	}

	return parseFromReader(r, builder)
}

func newBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	return bufio.NewReader(r)
}
