package kgo

import (
	"strconv"
	"strings"
	"time"
)

// NOTE: this code was originally copied from github.com/twmb/go-strftime.

// We cap a field width so that one conversion cannot allocate gigabytes per
// record.
const strftimeMaxWidth = 1e6

// strftimeAppendFormat appends t to dst according to the input strftime
// format in the C locale (LC_TIME=C), including the _, -, 0, ^, and # flags,
// field widths, and the %E and %O modifiers. We also support BSD's %+ and %v.
func strftimeAppendFormat(dst []byte, format string, t time.Time) []byte {
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			dst = append(dst, format[i])
			continue
		}

		start := i
		s := strftimeSpec{width: -1}
	flags:
		for i++; i < len(format); i++ {
			switch format[i] {
			case '_', '-', '0':
				s.pad = format[i]
			case '^':
				s.upper = true
			case '#':
				s.swap = true
			default:
				break flags
			}
		}
		if i < len(format) && '0' <= format[i] && format[i] <= '9' {
			s.width = 0
			for ; i < len(format) && '0' <= format[i] && format[i] <= '9'; i++ {
				s.width = min(s.width*10+int(format[i]-'0'), strftimeMaxWidth)
			}
		}
		var mod byte
		if i < len(format) && (format[i] == 'E' || format[i] == 'O') {
			mod = format[i]
			i++
		}

		if i == len(format) { // incomplete conversion at the end
			return s.appendStr(dst, format[start:])
		}
		dst = s.appendConv(dst, format[start:i+1], format[i], mod, t)
	}
	return dst
}

type strftimeSpec struct {
	width int  // -1 if unset
	pad   byte // '_', '-', '0', or 0 for the conversion's default
	upper bool // '^'
	swap  bool // '#'
	lower bool // set by %P, and by %p and %Z with #
}

// Appends conversion c with modifier mod. If c is unknown or rejects mod, we
// print the conversion as written in conv, still padded and case converted.
func (s *strftimeSpec) appendConv(dst []byte, conv string, c, mod byte, t time.Time) []byte {
	// %#Eb prints %#EB.
	if s.swap && (c == 'b' || c == 'h') {
		s.upper = true
	}
	if mod == 'E' && strings.IndexByte("aAbBdDeFgGhHIjklmMSUvVwW+", c) >= 0 ||
		mod == 'O' && strings.IndexByte("aAcDFvxXY+", c) >= 0 {
		return s.appendStr(dst, conv)
	}

	switch c {
	case '%':
		// %E% and %O% print only their last %.
		return s.appendStr(dst, "%")
	case 'a', 'A': // abbrev day, full day
		if s.swap {
			s.upper = true
		}
		day := t.Weekday().String()
		if c == 'a' {
			day = day[:3]
		}
		return s.appendStr(dst, day)
	case 'b', 'h', 'B': // abbrev month, h is equivalent to b, full month
		if s.swap {
			s.upper = true
		}
		month := t.Month().String()
		if c != 'B' {
			month = month[:3]
		}
		return s.appendStr(dst, month)
	case 'c': // preferred date and time representation
		return s.appendSub(dst, "%a %b %e %H:%M:%S %Y", t)
	case 'C': // century, rounded down so that 100*%C + %y is the year
		year := t.Year()
		century := year / 100
		if year%100 < 0 {
			century--
		}
		return s.appendNum(dst, century, 1)
	case 'd': // day of month
		return s.appendNum(dst, t.Day(), 2)
	case 'D':
		return s.appendSub(dst, "%m/%d/%y", t)
	case 'e': // day of month, space padded
		return s.appendSpaceNum(dst, t.Day(), 2)
	case 'F': // iso8601 date
		return s.appendSub(dst, "%Y-%m-%d", t)
	case 'g', 'G', 'V': // iso8601 week-based year without and with century, week number
		year, week := t.ISOWeek()
		switch c {
		case 'g':
			return s.appendNum(dst, (year%100+100)%100, 2)
		case 'G':
			return s.appendNum(dst, year, 1)
		}
		return s.appendNum(dst, week, 2)
	case 'H': // hour on 24hr clock
		return s.appendNum(dst, t.Hour(), 2)
	case 'I': // hour on 12hr clock
		return s.appendNum(dst, hour12(t.Hour()), 2)
	case 'j': // day of year
		return s.appendNum(dst, t.YearDay(), 3)
	case 'k': // hour on 24hr clock, space padded
		return s.appendSpaceNum(dst, t.Hour(), 2)
	case 'l': // hour on 12hr clock, space padded
		return s.appendSpaceNum(dst, hour12(t.Hour()), 2)
	case 'm': // month
		return s.appendNum(dst, int(t.Month()), 2)
	case 'M': // minute
		return s.appendNum(dst, t.Minute(), 2)
	case 'n':
		return s.appendStr(dst, "\n")
	case 'p', 'P': // AM or PM, P is lowercase
		if c == 'P' || s.swap {
			s.lower = true
		}
		if t.Hour() < 12 {
			return s.appendStr(dst, "AM")
		}
		return s.appendStr(dst, "PM")
	case 'r':
		return s.appendSub(dst, "%I:%M:%S %p", t)
	case 'R':
		return s.appendSub(dst, "%H:%M", t)
	case 's': // seconds since epoch, padded only to an explicit width
		var buf [20]byte
		b := strconv.AppendInt(buf[:0], t.Unix(), 10)
		return append(s.appendPad(dst, len(b)), b...)
	case 'S': // second
		return s.appendNum(dst, t.Second(), 2)
	case 't':
		return s.appendStr(dst, "\t")
	case 'T':
		return s.appendSub(dst, "%H:%M:%S", t)
	case 'u': // day of week, Monday 1 to Sunday 7
		return s.appendNum(dst, (int(t.Weekday())+6)%7+1, 1)
	case 'U': // week of year, days before the first Sunday are week 0
		return s.appendNum(dst, (t.YearDay()-1-int(t.Weekday())+7)/7, 2)
	case 'v': // BSD's %e-%b-%Y
		return s.appendSub(dst, "%e-%b-%Y", t)
	case 'w': // day of week, Sunday 0 to Saturday 6
		return s.appendNum(dst, int(t.Weekday()), 1)
	case 'W': // week of year, days before the first Monday are week 0
		return s.appendNum(dst, (t.YearDay()-1-(int(t.Weekday())+6)%7+7)/7, 2)
	case 'x': // date in the C locale
		return s.appendSub(dst, "%m/%d/%y", t)
	case 'X': // time in the C locale
		return s.appendSub(dst, "%H:%M:%S", t)
	case 'y': // year without century
		return s.appendNum(dst, (t.Year()%100+100)%100, 2)
	case 'Y': // year
		return s.appendNum(dst, t.Year(), 1)
	case 'z': // +hhmm or -hhmm offset from utc
		_, off := t.Zone()
		sign := "+"
		if off < 0 {
			sign, off = "-", -off
		}
		dst = s.appendStr(dst, sign)
		off /= 60
		return s.appendNum(dst, off/60*100+off%60, 4)
	case 'Z': // timezone
		if s.swap {
			s.lower = true
		}
		name, _ := t.Zone()
		return s.appendStr(dst, name)
	case '+': // BSD's date and time in date(1) format
		return s.appendSub(dst, "%a %b %e %H:%M:%S %Z %Y", t)
	}
	return s.appendStr(dst, conv)
}

// Pads content of length n to the width, with zeros for the 0 flag and spaces
// otherwise.
func (s *strftimeSpec) appendPad(dst []byte, n int) []byte {
	c := byte(' ')
	if s.pad == '0' {
		c = '0'
	}
	for ; n < s.width; n++ {
		dst = append(dst, c)
	}
	return dst
}

// Appends str padded to the width, then lowercased or uppercased if asked.
func (s *strftimeSpec) appendStr(dst []byte, str string) []byte {
	dst = s.appendPad(dst, len(str))
	start := len(dst)
	dst = append(dst, str...)
	switch {
	case s.lower:
		for i := start; i < len(dst); i++ {
			if 'A' <= dst[i] && dst[i] <= 'Z' {
				dst[i] += 'a' - 'A'
			}
		}
	case s.upper:
		strftimeUpper(dst[start:])
	}
	return dst
}

// Appends n with at least digits digits, or the width if larger, padded with
// zeros after any minus sign, or with spaces for the _ flag. The - flag drops
// this padding, but we still pad to an explicit width with spaces.
func (s *strftimeSpec) appendNum(dst []byte, n, digits int) []byte {
	if s.pad == 0 && s.width < 0 && n >= 0 { // the common case, no flags
		if digits >= 4 && n < 1000 {
			dst = append(dst, '0')
		}
		if digits >= 3 && n < 100 {
			dst = append(dst, '0')
		}
		if digits >= 2 && n < 10 {
			dst = append(dst, '0')
		}
		return strconv.AppendInt(dst, int64(n), 10)
	}
	var buf [20]byte
	b := strconv.AppendInt(buf[:0], int64(n), 10)
	if s.pad == '-' {
		return append(s.appendPad(dst, len(b)), b...)
	}
	pad := max(digits, s.width) - len(b)
	c := byte('0')
	if s.pad == '_' {
		c = ' '
	} else if pad > 0 && b[0] == '-' {
		dst = append(dst, '-')
		b = b[1:]
	}
	for ; pad > 0; pad-- {
		dst = append(dst, c)
	}
	return append(dst, b...)
}

// Appends n as appendNum does, but pads with spaces unless the 0 or - flag
// is given.
func (s *strftimeSpec) appendSpaceNum(dst []byte, n, digits int) []byte {
	if s.pad != '0' && s.pad != '-' {
		s.pad = '_'
	}
	return s.appendNum(dst, n, digits)
}

// Appends the expansion of a composite conversion such as %F, padded to the
// width as a whole. Only the ^ flag applies to the expansion.
func (s *strftimeSpec) appendSub(dst []byte, sub string, t time.Time) []byte {
	start := len(dst)
	dst = strftimeAppendFormat(dst, sub, t)
	if pad := s.width - (len(dst) - start); pad > 0 {
		dst = append(dst, make([]byte, pad)...)
		copy(dst[start+pad:], dst[start:])
		c := byte(' ')
		if s.pad == '0' {
			c = '0'
		}
		for i := start; i < start+pad; i++ {
			dst[i] = c
		}
	}
	if s.upper {
		strftimeUpper(dst[start:])
	}
	return dst
}

func strftimeUpper(b []byte) {
	for i, c := range b {
		if 'a' <= c && c <= 'z' {
			b[i] -= 'a' - 'A'
		}
	}
}

// hour12 returns hour h of a 24hr clock as an hour of a 12hr clock, 1 to 12.
func hour12(h int) int {
	if h %= 12; h == 0 {
		return 12
	}
	return h
}
