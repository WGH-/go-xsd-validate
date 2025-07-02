// Package xsdvalidate is a go package for xsd validation that utilizes libxml2.

//The goal of this package is to preload xsd files into memory and to validate xml (fast) using libxml2, like post bodys of xml service endpoints or api routers. At the time of writing, similar packages I found on github either didn't provide error details or got stuck under load. In addition to providing error strings it also exposes some fields of libxml2 return structs.
package xsdvalidate

import "C"
import (
	"runtime"
	"sync"
	"time"
)

var onceInit = sync.OnceFunc(func() {
	libXml2Init()
})

// Options type for parser/validation options.
type Options uint8

// The parser options, ParsErrVerbose will slow down parsing considerably!
const (
	ParsErrDefault Options = 1 << iota // Default parser error output
	ParsErrVerbose                     // Verbose parser error output, considerably slower!
)

// Validation options for possible future enhancements.
const (
	ValidErrDefault Options = 128 << iota // Default validation error output
)

// Init initialializes libxml2.
//
// Deprecated: it's not necessary to call Init explicitly.
func Init() error {
	onceInit()
	return nil
}

// InitWithGc is the same as [Init].
//
// Deprecated: kept for backward compatibility.
func InitWithGc(d time.Duration) {
	Init()
}

// Cleanup does nothing, and kept for backward compatibility.
//
// Deprecated: this function is no-op.
func Cleanup() {}

// NewXmlHandlerMem creates a xml handler struct.
// If an error is returned it can be of type Libxml2Error or XmlParserError.
// Always use the Free() method when done using this handler or memory will be leaking.
// The go garbage collector will not collect the allocated resources.
func NewXmlHandlerMem(inXml []byte, options Options) (*XmlHandler, error) {
	onceInit()

	xPtr, err := parseXmlMem(inXml, options)
	if err != nil {
		return nil, err
	}
	res := &XmlHandler{docPtr: xPtr}
	res.cleanup = runtime.AddCleanup(res, freeDocPtr, xPtr)
	return res, nil
}

// NewXsdHandlerUrl creates a xsd handler struct.
// Always use Free() method when done using this handler or memory will be leaking.
// If an error is returned it can be of type Libxml2Error or XsdParserError.
// The go garbage collector will not collect the allocated resources.
func NewXsdHandlerUrl(url string, options Options) (*XsdHandler, error) {
	onceInit()

	sPtr, err := parseUrlSchema(url, options)
	if err != nil {
		return nil, err
	}
	res := &XsdHandler{schemaPtr: sPtr}
	res.cleanup = runtime.AddCleanup(res, freeSchemaPtr, sPtr)
	return res, nil
}

// NewXsdHandlerMem creates an xsd handler struct.
// Always use Free() method when done using this handler or memory will leak.
// If an error is returned it can be of type Libxml2Error or XsdParserError.
// The go garbage collector will not collect the allocated resources.
func NewXsdHandlerMem(inSchema []byte, options Options) (*XsdHandler, error) {
	onceInit()

	sPtr, err := parseMemSchema(inSchema, options)
	if err != nil {
		return nil, err
	}
	res := &XsdHandler{schemaPtr: sPtr}
	res.cleanup = runtime.AddCleanup(res, freeSchemaPtr, sPtr)
	return res, nil
}

// Validate validates an xmlHandler against an xsdHandler and returns a ValidationError.
// If an error is returned it is of type Libxml2Error, XsdParserError, XmlParserError or ValidationError.
// Both xmlHandler and xsdHandler have to be created first.
func (xsdHandler *XsdHandler) Validate(xmlHandler *XmlHandler, options Options) error {
	if xsdHandler == nil || xsdHandler.schemaPtr == nil {
		return XsdParserError{errorMessage{"Xsd handler not properly initialized"}}

	}
	if xmlHandler == nil || xmlHandler.docPtr == nil {
		return XmlParserError{errorMessage{"Xml handler not properly initialized"}}
	}
	return validateWithXsd(xmlHandler, xsdHandler)

}

// ValidateMem validates an xml byte slice against an xsdHandler.
// If an error is returned it can be of type Libxml2Error, XsdParserError, XmlParserError or ValidationError.
// The xsdHandler has to be created first.
func (xsdHandler *XsdHandler) ValidateMem(inXml []byte, options Options) error {
	if xsdHandler == nil || xsdHandler.schemaPtr == nil {
		return XsdParserError{errorMessage{"Xsd handler not properly initialized"}}

	}
	return validateBufWithXsd(inXml, options, xsdHandler)

}

// Free frees the wrapped schemaPtr, call this when this handler is not needed anymore.
func (xsdHandler *XsdHandler) Free() {
	xsdHandler.cleanup.Stop()
	freeSchemaPtr(xsdHandler.schemaPtr)
}

// Free frees the wrapped xml docPtr, call this when this handler is not needed anymore.
func (xmlHandler *XmlHandler) Free() {
	xmlHandler.cleanup.Stop()
	freeDocPtr(xmlHandler.docPtr)
}
