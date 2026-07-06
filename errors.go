// Copyright (c) the go-ruby-excon/excon authors
//
// SPDX-License-Identifier: BSD-3-Clause

package excon

import "fmt"

// Error is the root of Excon's error tree (Excon::Error, aliased in the gem as
// Excon::Errors::Error). Every excon error carries a human message; a status
// error additionally carries the [Request] and [Response] that triggered it, and
// a transport error wraps the underlying cause.
//
// The concrete kinds are distinguished by [Error.Kind]; the predicate helpers
// ([IsHTTPStatusError], [IsClientError], …) and the sentinel values
// ([ErrNotFound], …) match with errors.Is, walking the class hierarchy exactly
// as Ruby's rescue of an Excon::Error subclass does.
type Error struct {
	// Kind names the specific Excon error subclass (see the Err* sentinels).
	Kind ErrorKind
	// Message is the error text (Excon::Error#message).
	Message string
	// Request is the request context for a status error (nil otherwise).
	Request *Request
	// Response is the response context for a status error (nil otherwise).
	Response *Response
	// Cause is the underlying transport error for Socket/Timeout errors.
	Cause error
}

// ErrorKind identifies an Excon error subclass, named as in the gem.
type ErrorKind string

// The Excon error subclasses, named as in Excon::Error::*.
const (
	KindError ErrorKind = "Excon::Error"

	// Transport / non-status errors.
	KindSocket               ErrorKind = "Excon::Error::Socket"
	KindCertificate          ErrorKind = "Excon::Error::Certificate"
	KindTimeout              ErrorKind = "Excon::Error::Timeout"
	KindResponseParse        ErrorKind = "Excon::Error::ResponseParse"
	KindProxyConnectionError ErrorKind = "Excon::Error::ProxyConnectionError"
	KindProxyParse           ErrorKind = "Excon::Error::ProxyParse"
	KindTooManyRedirects     ErrorKind = "Excon::Error::TooManyRedirects"

	// HTTP status errors: base and range bases.
	KindHTTPStatus    ErrorKind = "Excon::Error::HTTPStatus"
	KindInformational ErrorKind = "Excon::Error::Informational"
	KindRedirection   ErrorKind = "Excon::Error::Redirection"
	KindClient        ErrorKind = "Excon::Error::Client"
	KindServer        ErrorKind = "Excon::Error::Server"

	// 4xx client status errors.
	KindBadRequest                   ErrorKind = "Excon::Error::BadRequest"
	KindUnauthorized                 ErrorKind = "Excon::Error::Unauthorized"
	KindPaymentRequired              ErrorKind = "Excon::Error::PaymentRequired"
	KindForbidden                    ErrorKind = "Excon::Error::Forbidden"
	KindNotFound                     ErrorKind = "Excon::Error::NotFound"
	KindMethodNotAllowed             ErrorKind = "Excon::Error::MethodNotAllowed"
	KindNotAcceptable                ErrorKind = "Excon::Error::NotAcceptable"
	KindProxyAuthenticationRequired  ErrorKind = "Excon::Error::ProxyAuthenticationRequired"
	KindRequestTimeout               ErrorKind = "Excon::Error::RequestTimeout"
	KindConflict                     ErrorKind = "Excon::Error::Conflict"
	KindGone                         ErrorKind = "Excon::Error::Gone"
	KindLengthRequired               ErrorKind = "Excon::Error::LengthRequired"
	KindPreconditionFailed           ErrorKind = "Excon::Error::PreconditionFailed"
	KindRequestEntityTooLarge        ErrorKind = "Excon::Error::RequestEntityTooLarge"
	KindRequestURITooLong            ErrorKind = "Excon::Error::RequestURITooLong"
	KindUnsupportedMediaType         ErrorKind = "Excon::Error::UnsupportedMediaType"
	KindRequestedRangeNotSatisfiable ErrorKind = "Excon::Error::RequestedRangeNotSatisfiable"
	KindExpectationFailed            ErrorKind = "Excon::Error::ExpectationFailed"
	KindUnprocessableEntity          ErrorKind = "Excon::Error::UnprocessableEntity"
	KindTooManyRequests              ErrorKind = "Excon::Error::TooManyRequests"

	// 5xx server status errors.
	KindInternalServerError ErrorKind = "Excon::Error::InternalServerError"
	KindNotImplemented      ErrorKind = "Excon::Error::NotImplemented"
	KindBadGateway          ErrorKind = "Excon::Error::BadGateway"
	KindServiceUnavailable  ErrorKind = "Excon::Error::ServiceUnavailable"
	KindGatewayTimeout      ErrorKind = "Excon::Error::GatewayTimeout"
)

// Sentinel errors for errors.Is matching. Each names an Excon error kind; a
// concrete [Error] with that Kind (or a descendant of it) matches via [Error.Is].
var (
	ErrError                = &Error{Kind: KindError, Message: string(KindError)}
	ErrSocket               = &Error{Kind: KindSocket, Message: string(KindSocket)}
	ErrCertificate          = &Error{Kind: KindCertificate, Message: string(KindCertificate)}
	ErrTimeout              = &Error{Kind: KindTimeout, Message: string(KindTimeout)}
	ErrResponseParse        = &Error{Kind: KindResponseParse, Message: string(KindResponseParse)}
	ErrProxyConnectionError = &Error{Kind: KindProxyConnectionError, Message: string(KindProxyConnectionError)}
	ErrProxyParse           = &Error{Kind: KindProxyParse, Message: string(KindProxyParse)}
	ErrTooManyRedirects     = &Error{Kind: KindTooManyRedirects, Message: string(KindTooManyRedirects)}

	ErrHTTPStatus    = &Error{Kind: KindHTTPStatus, Message: string(KindHTTPStatus)}
	ErrInformational = &Error{Kind: KindInformational, Message: string(KindInformational)}
	ErrRedirection   = &Error{Kind: KindRedirection, Message: string(KindRedirection)}
	ErrClient        = &Error{Kind: KindClient, Message: string(KindClient)}
	ErrServer        = &Error{Kind: KindServer, Message: string(KindServer)}

	ErrBadRequest                   = &Error{Kind: KindBadRequest, Message: string(KindBadRequest)}
	ErrUnauthorized                 = &Error{Kind: KindUnauthorized, Message: string(KindUnauthorized)}
	ErrPaymentRequired              = &Error{Kind: KindPaymentRequired, Message: string(KindPaymentRequired)}
	ErrForbidden                    = &Error{Kind: KindForbidden, Message: string(KindForbidden)}
	ErrNotFound                     = &Error{Kind: KindNotFound, Message: string(KindNotFound)}
	ErrMethodNotAllowed             = &Error{Kind: KindMethodNotAllowed, Message: string(KindMethodNotAllowed)}
	ErrNotAcceptable                = &Error{Kind: KindNotAcceptable, Message: string(KindNotAcceptable)}
	ErrProxyAuthenticationRequired  = &Error{Kind: KindProxyAuthenticationRequired, Message: string(KindProxyAuthenticationRequired)}
	ErrRequestTimeout               = &Error{Kind: KindRequestTimeout, Message: string(KindRequestTimeout)}
	ErrConflict                     = &Error{Kind: KindConflict, Message: string(KindConflict)}
	ErrGone                         = &Error{Kind: KindGone, Message: string(KindGone)}
	ErrLengthRequired               = &Error{Kind: KindLengthRequired, Message: string(KindLengthRequired)}
	ErrPreconditionFailed           = &Error{Kind: KindPreconditionFailed, Message: string(KindPreconditionFailed)}
	ErrRequestEntityTooLarge        = &Error{Kind: KindRequestEntityTooLarge, Message: string(KindRequestEntityTooLarge)}
	ErrRequestURITooLong            = &Error{Kind: KindRequestURITooLong, Message: string(KindRequestURITooLong)}
	ErrUnsupportedMediaType         = &Error{Kind: KindUnsupportedMediaType, Message: string(KindUnsupportedMediaType)}
	ErrRequestedRangeNotSatisfiable = &Error{Kind: KindRequestedRangeNotSatisfiable, Message: string(KindRequestedRangeNotSatisfiable)}
	ErrExpectationFailed            = &Error{Kind: KindExpectationFailed, Message: string(KindExpectationFailed)}
	ErrUnprocessableEntity          = &Error{Kind: KindUnprocessableEntity, Message: string(KindUnprocessableEntity)}
	ErrTooManyRequests              = &Error{Kind: KindTooManyRequests, Message: string(KindTooManyRequests)}

	ErrInternalServerError = &Error{Kind: KindInternalServerError, Message: string(KindInternalServerError)}
	ErrNotImplemented      = &Error{Kind: KindNotImplemented, Message: string(KindNotImplemented)}
	ErrBadGateway          = &Error{Kind: KindBadGateway, Message: string(KindBadGateway)}
	ErrServiceUnavailable  = &Error{Kind: KindServiceUnavailable, Message: string(KindServiceUnavailable)}
	ErrGatewayTimeout      = &Error{Kind: KindGatewayTimeout, Message: string(KindGatewayTimeout)}
)

// errorParents maps each kind to its parent kind in the Excon::Error hierarchy.
// Certificate descends from Socket; the range bases and every named status error
// descend (transitively) from HTTPStatus; the transport errors and HTTPStatus
// descend directly from Error. The root, KindError, has no parent.
var errorParents = map[ErrorKind]ErrorKind{
	KindSocket:               KindError,
	KindCertificate:          KindSocket,
	KindTimeout:              KindError,
	KindResponseParse:        KindError,
	KindProxyConnectionError: KindError,
	KindProxyParse:           KindError,
	KindTooManyRedirects:     KindError,

	KindHTTPStatus:    KindError,
	KindInformational: KindHTTPStatus,
	KindRedirection:   KindHTTPStatus,
	KindClient:        KindHTTPStatus,
	KindServer:        KindHTTPStatus,

	KindBadRequest:                   KindClient,
	KindUnauthorized:                 KindClient,
	KindPaymentRequired:              KindClient,
	KindForbidden:                    KindClient,
	KindNotFound:                     KindClient,
	KindMethodNotAllowed:             KindClient,
	KindNotAcceptable:                KindClient,
	KindProxyAuthenticationRequired:  KindClient,
	KindRequestTimeout:               KindClient,
	KindConflict:                     KindClient,
	KindGone:                         KindClient,
	KindLengthRequired:               KindClient,
	KindPreconditionFailed:           KindClient,
	KindRequestEntityTooLarge:        KindClient,
	KindRequestURITooLong:            KindClient,
	KindUnsupportedMediaType:         KindClient,
	KindRequestedRangeNotSatisfiable: KindClient,
	KindExpectationFailed:            KindClient,
	KindUnprocessableEntity:          KindClient,
	KindTooManyRequests:              KindClient,

	KindInternalServerError: KindServer,
	KindNotImplemented:      KindServer,
	KindBadGateway:          KindServer,
	KindServiceUnavailable:  KindServer,
	KindGatewayTimeout:      KindServer,
}

// statusErrorKinds maps a specific HTTP status code to its named Excon error
// subclass, matching Excon::Error's STATUS_ERRORS table.
var statusErrorKinds = map[int]ErrorKind{
	400: KindBadRequest,
	401: KindUnauthorized,
	402: KindPaymentRequired,
	403: KindForbidden,
	404: KindNotFound,
	405: KindMethodNotAllowed,
	406: KindNotAcceptable,
	407: KindProxyAuthenticationRequired,
	408: KindRequestTimeout,
	409: KindConflict,
	410: KindGone,
	411: KindLengthRequired,
	412: KindPreconditionFailed,
	413: KindRequestEntityTooLarge,
	414: KindRequestURITooLong,
	415: KindUnsupportedMediaType,
	416: KindRequestedRangeNotSatisfiable,
	417: KindExpectationFailed,
	422: KindUnprocessableEntity,
	429: KindTooManyRequests,
	500: KindInternalServerError,
	501: KindNotImplemented,
	502: KindBadGateway,
	503: KindServiceUnavailable,
	504: KindGatewayTimeout,
}

// Error implements the error interface (Excon::Error#message).
func (e *Error) Error() string { return e.Message }

// Unwrap exposes the underlying transport cause for errors.Is/As.
func (e *Error) Unwrap() error { return e.Cause }

// Is reports whether e matches target: true when target is a [*Error] whose Kind
// is e's Kind or an ancestor of it. So errors.Is(err, ErrClient) matches any 4xx
// status error, errors.Is(err, ErrHTTPStatus) matches any status error, and
// errors.Is(err, ErrError) matches every excon error — mirroring Ruby's rescue
// of a superclass.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	for k := e.Kind; ; {
		if k == t.Kind {
			return true
		}
		parent, ok := errorParents[k]
		if !ok {
			return false
		}
		k = parent
	}
}

// statusErrorKind maps a response status to the Excon error kind raised on an
// :expects mismatch: a specific named subclass when the code is in the table,
// otherwise the range base (Informational/Redirection/Client/Server), or the
// HTTPStatus base for a status outside 100..599.
func statusErrorKind(status int) ErrorKind {
	if k, ok := statusErrorKinds[status]; ok {
		return k
	}
	switch {
	case status >= 100 && status < 200:
		return KindInformational
	case status >= 300 && status < 400:
		return KindRedirection
	case status >= 400 && status < 500:
		return KindClient
	case status >= 500 && status < 600:
		return KindServer
	default:
		return KindHTTPStatus
	}
}

// newStatusError builds the status [Error] for an :expects mismatch, mirroring
// Excon::Error.status_error: the message is
// "Expected(<expects>) <=> Actual(<status> <reason>)" and the request/response
// context is attached.
func newStatusError(expects []int, req *Request, resp *Response) *Error {
	return &Error{
		Kind:     statusErrorKind(resp.status),
		Message:  fmt.Sprintf("Expected(%v) <=> Actual(%d %s)", expects, resp.status, resp.reason),
		Request:  req,
		Response: resp,
	}
}

// newTransportError builds a transport [Error] (Socket/Timeout) wrapping the
// adapter's underlying error as the cause.
func newTransportError(kind ErrorKind, cause error) *Error {
	msg := string(kind)
	if cause != nil {
		msg = cause.Error()
	}
	return &Error{Kind: kind, Message: msg, Cause: cause}
}

// IsHTTPStatusError reports whether err is an Excon HTTP status error (or any
// subclass), i.e. an error raised by an :expects mismatch.
func IsHTTPStatusError(err error) bool { return isKind(err, ErrHTTPStatus) }

// IsClientError reports whether err is an Excon 4xx client status error.
func IsClientError(err error) bool { return isKind(err, ErrClient) }

// IsServerError reports whether err is an Excon 5xx server status error.
func IsServerError(err error) bool { return isKind(err, ErrServer) }

// IsSocketError reports whether err is an Excon socket (transport) error.
func IsSocketError(err error) bool { return isKind(err, ErrSocket) }

// IsTimeout reports whether err is an Excon timeout error.
func IsTimeout(err error) bool { return isKind(err, ErrTimeout) }

// isKind is the errors.Is shim used by the predicate helpers.
func isKind(err error, sentinel *Error) bool {
	e, ok := err.(*Error)
	return ok && e.Is(sentinel)
}

// retryable reports whether err is a transport error Excon retries an idempotent
// request on (a socket or timeout error).
func retryable(err error) bool {
	e, ok := err.(*Error)
	return ok && (e.Is(ErrSocket) || e.Is(ErrTimeout))
}
