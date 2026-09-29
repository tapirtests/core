package diag

import (
	"strconv"
	"strings"
)

// Pointer is a JSON Pointer (RFC 6901) to a value inside a document,
// e.g. "/scenarios/productLifecycle/steps/2/inputs/age".
//
// Pointers are immutable; Key and Index return new pointers:
//
//	diag.Root.Key("scenarios").Key(id).Key("steps").Index(2)
type Pointer string

// Root points to the whole document.
const Root Pointer = ""

var pointerEscaper = strings.NewReplacer("~", "~0", "/", "~1")

// Key returns a pointer to the object member named key.
func (p Pointer) Key(key string) Pointer {
	return p + "/" + Pointer(pointerEscaper.Replace(key))
}

// Index returns a pointer to the array element at index i.
func (p Pointer) Index(i int) Pointer {
	return p + "/" + Pointer(strconv.Itoa(i))
}

// Location tells where a problem is.
//
// Pointer is the primary address: UIs work with the parsed model, not with
// the text, and use it to highlight a field. Line and Column are optional
// (0 means unknown) and help humans jump to the place in a text editor.
type Location struct {
	// File is the path of the document, e.g. "tapir.json". Empty if unknown.
	File    string  `json:"file,omitempty"`
	Pointer Pointer `json:"pointer,omitempty"`
	// Line and Column are 1-based; 0 means unknown.
	Line   int `json:"line,omitempty"`
	Column int `json:"column,omitempty"`
}

// At returns a location of the value at pointer p inside file.
func At(file string, p Pointer) Location {
	return Location{File: file, Pointer: p}
}

// String formats the location for humans. Line and column win over the
// pointer because they can be followed in an editor:
//
//	tapir.json:148:17
//	tapir.json#/scenarios/login
//	/scenarios/login
func (l Location) String() string {
	if l.Line > 0 {
		s := l.File + ":" + strconv.Itoa(l.Line)
		if l.Column > 0 {
			s += ":" + strconv.Itoa(l.Column)
		}
		return s
	}
	if l.File != "" && l.Pointer != Root {
		return l.File + "#" + string(l.Pointer)
	}
	if l.File != "" {
		return l.File
	}
	return string(l.Pointer)
}
