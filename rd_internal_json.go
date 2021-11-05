package rd

/*
What is this?

	* Simple JSON parser for `rd.Set` / `rd.Json.Haser`.
		* Collects top-level object keys.
		* Discards all other data.

Why?

	* Got baited by benchmarks.
	* Seems to perform much better than "encoding/json":
		* Around x3-4 faster in our benchmarks.
		* Way fewer allocations. The resulting map is the only alloc.
*/

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// Input should be empty or valid JSON containing a top-level object.
// Output is the set of top-level keys.
func parseSet(src string) Set {
	par := par{src: src}
	par.top()
	return par.out
}

// Short for "parser".
type par struct {
	src string // Short for "source".
	cur int    // Short for "cursor".
	lvl int    // Short for "level".
	out Set    // Short for "output".
}

func (self *par) top() {
	if self.next() && self.peek() == '{' {
		self.cur++
		self.obj()
	}
}

func (self *par) any() {
	self.next()
	char := self.peek()

	if digits.has(char) {
		self.cur++
		self.num()
		return
	}

	switch char {
	case '{':
		self.cur++
		self.obj()
	case '[':
		self.cur++
		self.arr()
	case '"':
		self.cur++
		self.str()
	case 'n':
		self.cur++
		self.ident(`ull`)
	case 't':
		self.cur++
		self.ident(`rue`)
	case 'f':
		self.cur++
		self.ident(`alse`)
	case '-':
		self.cur++
		self.beforeNum()
	default:
		panic(self.err())
	}
}

func (self *par) obj() {
	self.lvl++

	const (
		beforeKey = iota
		afterKey
		afterColon
		afterValue
		afterComma
	)

	mode := beforeKey

	for self.next() {
		switch mode {
		case beforeKey:
			goto beforeKey
		case afterKey:
			goto afterKey
		case afterColon:
			goto afterColon
		case afterValue:
			goto afterValue
		case afterComma:
			goto afterComma
		default:
			goto unreachable
		}

	beforeKey:
		{
			switch self.peek() {
			case '}':
				self.cur++
				self.lvl--
				return

			case '"':
				self.cur++
				self.key()
				mode = afterKey
				continue

			default:
				panic(self.err())
			}
		}

	afterKey:
		{
			switch self.peek() {
			case ':':
				self.cur++
				mode = afterColon
				continue

			default:
				panic(self.err())
			}
		}

	afterColon:
		self.any()
		mode = afterValue
		continue

	afterValue:
		{
			switch self.peek() {
			case '}':
				self.cur++
				self.lvl--
				return

			case ',':
				self.cur++
				mode = afterComma
				continue

			default:
				panic(self.err())
			}
		}

	afterComma:
		if self.peek() == '"' {
			mode = beforeKey
			continue
		}
		panic(self.err())

	unreachable:
		panic(errUnreachable)
	}

	panic(errJsonEof)
}

func (self *par) key() {
	pos := self.cur
	self.str()
	if self.lvl == 1 {
		self.add(self.src[pos : self.cur-1])
	}
}

func (self *par) arr() {
	self.lvl++

	const (
		beforeVal = iota
		afterVal
		afterComma
	)

	mode := beforeVal

	for self.next() {
		switch mode {
		case beforeVal:
			goto beforeVal
		case afterVal:
			goto afterVal
		case afterComma:
			goto afterComma
		default:
			goto unreachable
		}

	beforeVal:
		if self.peek() == ']' {
			self.cur++
			self.lvl--
			return
		}

		self.any()
		mode = afterVal
		continue

	afterVal:
		{
			switch self.peek() {
			case ']':
				self.cur++
				self.lvl--
				return

			case ',':
				self.cur++
				mode = afterComma
				continue

			default:
				panic(self.err())
			}
		}

	afterComma:
		self.any()
		mode = afterVal
		continue

	unreachable:
		panic(errUnreachable)
	}

	panic(errJsonEof)
}

func (self *par) str() {
	for self.more() {
		switch self.peek() {
		case '"':
			self.cur++
			return
		case '\\':
			self.cur++
			self.esc()
		default:
			self.skipChar()
		}
	}
	panic(errJsonEof)
}

// Semi-placeholder. Lacks support for Unicode escapes.
func (self *par) esc() { self.skip() }

func (self *par) beforeNum() {
	if !digits.has(self.peek()) {
		panic(self.err())
	}
	self.cur++
	self.num()
}

func (self *par) num() {
	for self.more() {
		char := self.peek()

		if delims.has(char) {
			return
		}

		if digits.has(char) {
			self.cur++
			continue
		}

		if char == '.' {
			self.cur++
			goto mant
		}

		if exps.has(char) {
			self.cur++
			goto exp
		}

		panic(self.err())
	}
	return

mant:
	{
		char := self.peek()
		if !digits.has(char) {
			panic(self.err())
		}
		self.cur++

		for self.more() {
			char := self.peek()

			if delims.has(char) {
				return
			}

			if digits.has(char) {
				self.cur++
				continue
			}

			if exps.has(char) {
				self.cur++
				goto exp
			}

			panic(self.err())
		}
		return
	}

exp:
	{
		if signs.has(self.peek()) {
			self.cur++
		}
		goto expRest
	}

expRest:
	char := self.peek()
	if !digits.has(char) {
		panic(self.err())
	}
	self.cur++

	for self.more() {
		char := self.peek()

		if delims.has(char) {
			return
		}

		if digits.has(char) {
			self.cur++
			continue
		}

		panic(self.err())
	}
}

func (self *par) ident(prefix string) {
	if strings.HasPrefix(self.rest(), prefix) {
		self.cur += len(prefix)
		if !self.more() || delims.has(self.peek()) {
			return
		}
	}
	panic(self.err())
}

func (self *par) more() bool {
	return self.cur < len(self.src)
}

func (self *par) next() bool {
	for self.cur < len(self.src) {
		if whitespace.has(self.src[self.cur]) {
			self.cur++
			continue
		}
		return true
	}
	return false
}

func (self *par) rest() string {
	return self.src[self.cur:]
}

func (self *par) peek() byte {
	if !self.more() {
		panic(errJsonEof)
	}
	return self.src[self.cur]
}

func (self *par) skip() { self.cur++ }

func (self *par) skipChar() {
	_, size := utf8.DecodeRuneInString(self.rest())
	self.cur += size
}

func (self *par) add(key string) {
	if self.out == nil {
		self.out = make(Set, 16)
	}
	self.out.Add(key)
}

func (self *par) err() error {
	rest := strings.TrimSpace(self.rest())

	if len(rest) > 0 {
		return fmt.Errorf(
			`invalid JSON syntax in position %v: unexpected %q`,
			self.cur, rest,
		)
	}

	return fmt.Errorf(`unexpected JSON %w in position %v`, io.EOF, self.cur)
}

type charset [256]bool

func (self *charset) has(val byte) bool { return self[val] }

func (self *charset) addStr(vals string) *charset {
	for _, val := range vals {
		self[val] = true
	}
	return self
}

func (self *charset) addSet(vals *charset) *charset {
	for i, val := range vals {
		if val {
			self[i] = true
		}
	}
	return self
}

var (
	digits     = new(charset).addStr(`0123456789`)
	whitespace = new(charset).addStr("\r\n\t\v ")
	delims     = new(charset).addSet(whitespace).addStr(`{}[]",`)
	exps       = new(charset).addStr(`Ee`)
	signs      = new(charset).addStr(`+-`)
)
