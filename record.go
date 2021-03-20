package multidns

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"strings"
)

const (
	MessageQuery       uint8 = iota // a standard query
	OpCodeInverseQuery              // an inverse query
	OpCodeStatus                    // a server status request
)

const (
	CodeNoError        uint8 = iota // no error condition
	CodeFormatError                 // the name server was unable to interpret the query
	CodeServerFailure               // the name server was unable to process this query due to a problem with the name server
	CodeNameError                   // meaningful only for responses from an authoritative name server, this code signifies that the domain name referenced in the query does not exist
	CodeNotImplemented              // the name server does not support the requested kind of query
	CodeRefused                     // the name server refuses to perform the specified operation for policy reasons.For example, a name server may not wish to provide the information to the particular requester, or a name server may not wish to perform a particular operation
)

// Header section is always present.  The header includes fields that
// specify which of the remaining sections are present, and also specify
// whether the message is a query or a response, a standard query or some
// other opcode, etc.
type Header struct {
	// A 16 bit identifier assigned by the program that
	// generates any kind of query.  This identifier is copied
	// the corresponding reply and can be used by the requester
	// to match up replies to outstanding queries.
	ID uint16
	// A one bit field that specifies whether this message is a
	// query (false), or a response (true).
	IsReply bool
	// A four bit field that specifies kind of query in this
	// message.  This value is set by the originator of a query
	// and copied into the response.
	Opcode uint8
	// A one bit is valid in responses,
	// and specifies that the responding name server is an
	// authority for the domain name in question section.
	// Note that the contents of the answer section may have
	// multiple owner names because of aliases. The bit
	// corresponds to the name which matches the query name, or
	// the first owner name in the answer section.
	IsAuthoritative bool
	// A one bit specifies that this message was truncated
	// due to length greater than that permitted on the
	// transmission channel.
	IsTruncated bool
	// A one bit may be set in a query and
	// is copied into the response.If RD is set, it directs
	// the name server to pursue the query recursively.
	// Recursive query support is optional.
	IsRecursionDesired bool
	// A one but can be is set or cleared in a
	// response, and denotes whether recursive query support is
	// available in the name server.
	IsRecursionAvailable bool
	// A four bit field is set as part of responses.
	ResponseCode uint8
	// An unsigned 16 bit integer specifying the number of
	// entries in the question section.
	QuestionCount uint16
	// An unsigned 16 bit integer specifying the number of
	// resource records in the answer section.
	AnswerCount uint16
	// An unsigned 16 bit integer specifying the number of name
	// server resource records in the authority records
	// section.
	AuthorityCount uint16
	// An unsigned 16 bit integer specifying the number of
	// resource records in the additional records section.
	AdditionalCount uint16
}

const (
	QRBit = 1 << 7
	AABit = 1 << 2
	TCBit = 1 << 1
	RDBit = 1 << 0
	RABit = 1 << 7
	OFBit = 3 << 6
)

func (h *Header) ReadFrom(r io.Reader) (n int64, err error) {
	err = binary.Read(r, binary.BigEndian, &h.ID)
	if err != nil {
		return
	}
	b := []byte{0, 0}
	_, err = r.Read(b)
	if b[0]&QRBit == QRBit {
		h.IsReply = true
	}
	if b[0]&AABit == AABit {
		h.IsAuthoritative = true
	}
	if b[0]&TCBit == TCBit {
		h.IsTruncated = true
	}
	if b[0]&RDBit == RDBit {
		h.IsRecursionDesired = true
	}
	if b[1]&RABit == RABit {
		h.IsRecursionAvailable = true
	}
	h.Opcode = b[0] &^ QRBit >> 3
	h.ResponseCode = b[1] &^ RABit
	err = binary.Read(r, binary.BigEndian, &h.QuestionCount)
	if err != nil {
		return
	}
	err = binary.Read(r, binary.BigEndian, &h.AnswerCount)
	if err != nil {
		return
	}
	err = binary.Read(r, binary.BigEndian, &h.AuthorityCount)
	if err != nil {
		return
	}
	err = binary.Read(r, binary.BigEndian, &h.AdditionalCount)
	if err != nil {
		return
	}
	n = 12
	return
}

// WriteTo writes header into writer.
func (h *Header) WriteTo(w io.Writer) (n int64, err error) {
	err = binary.Write(w, binary.BigEndian, h.ID)
	if err != nil {
		return
	}
	//   0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
	// +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	// |QR|   Opcode  |AA|TC|RD|RA|    0   |   RCode   |
	// +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	b := []byte{h.Opcode << 3, h.ResponseCode}
	if h.IsReply {
		b[0] |= QRBit
	}
	if h.IsAuthoritative {
		b[0] |= AABit
	}
	if h.IsTruncated {
		b[0] |= TCBit
	}
	if h.IsRecursionDesired {
		b[0] |= RDBit
	}
	if h.IsRecursionAvailable {
		b[1] |= RABit
	}
	_, err = w.Write(b)
	if err != nil {
		return
	}
	err = binary.Write(w, binary.BigEndian, h.QuestionCount)
	if err != nil {
		return
	}
	err = binary.Write(w, binary.BigEndian, h.AnswerCount)
	if err != nil {
		return
	}
	err = binary.Write(w, binary.BigEndian, h.AuthorityCount)
	if err != nil {
		return
	}
	err = binary.Write(w, binary.BigEndian, h.AdditionalCount)
	if err != nil {
		return
	}
	n = 12
	return
}

// Label is a single length octet followed by that number of characters.  Label
// is treated as binary information, and can be up to 256 characters in
// length (including the length octet).
type Label struct {
	Length uint8
	Name   []byte
}

func (l *Label) ReadFrom(r io.Reader) (n int64, err error) {
	b := []byte{0}
	_, err = r.Read(b)
	if err != nil {
		return
	}
	n++
	l.Length = b[0]
	if b[0]&OFBit == OFBit {
		_, err = r.Read(b)
		if err != nil {
			return
		}
		l.Name = b
		n++
		return
	}
	var i int
	l.Name = make([]byte, l.Length)
	i, err = io.ReadFull(r, l.Name)
	if err != nil {
		return
	}
	n += int64(i)
	return
}

// WriteTo writes label into writer.
func (l *Label) WriteTo(w io.Writer) (n int64, err error) {
	_, err = w.Write([]byte{l.Length})
	if err != nil {
		return
	}
	var i int
	i, err = w.Write(l.Name)
	if err != nil {
		return
	}
	n = 1 + int64(i)
	return
}

// Offset
func (l *Label) Offset() (offset int64) {
	if l.Length&OFBit == OFBit {
		offset = int64(l.Length)
		offset &= ^OFBit
		offset <<= 8
		offset |= int64(l.Name[0])
	}
	return
}

// Domain is a name represented as a series of Labels, and
// terminated by a Label with zero length.
type Domain []Label

func (d *Domain) ReadFrom(r io.Reader) (n int64, err error) {
	var label Label
	n, err = label.ReadFrom(r)
	if err != nil {
		return
	}
	var i int64
	if o := label.Offset(); o != 0 {
		switch r := r.(type) {
		case io.ReadSeeker:
			i, err = r.Seek(0, io.SeekCurrent)
			if err != nil {
				return
			}
			defer func(i int64) {
				_, err = r.Seek(i, io.SeekStart)
			}(i)
			_, err = r.Seek(o, io.SeekStart)
			if err != nil {
				return
			}
			_, err = d.ReadFrom(r)
			if err != nil {
				return
			}
		default:
			err = fmt.Errorf("can't seek trougth reader")
		}
		return
	}
	d.Add(label)
	if label.Length == 0 {
		return
	}
	i, err = d.ReadFrom(r)
	if err != nil {
		return
	}
	n += i
	return
}

// WriteTo writes domain into writer.
func (d *Domain) WriteTo(w io.Writer) (n int64, err error) {
	var i int64
	for _, label := range *d {
		i, err = label.WriteTo(w)
		if err != nil {
			return
		}
		n += i
	}
	return
}

func (d *Domain) Add(label Label) {
	*d = append(*d, label)
}

func (d *Domain) Name() (name []string) {
	for _, label := range *d {
		name = append(name, string(label.Name))
	}
	return
}

// ResourceRecordTypes are used in resource records.  Note that these types are a
// subset of QuestionTypes.
const (
	TypeA                  uint16 = iota + 1 // a host address
	TypeNS                                   // an authoritative name server
	TypeMD                                   // a mail destination
	TypeMF                                   // a mail forwarder
	TypeCNAME                                // the canonical name for an alias
	TypeSOA                                  // marks the start of a zone of authority
	TypeMB                                   // a mailbox domain name
	TypeMG                                   // a mail group member
	TypeMR                                   // a mail rename domain name
	ResourceRecordTypeNULL                   // a null RR
	TypeWKS                                  // a well known service description
	TypePTR                                  // a domain name pointer
	TypeHINFO                                // host information
	TypeMINFO                                // mailbox or mail list information
	TypeMX                                   // mail exchange
	TypeTXT                                  // text strings
	TypePR
	TypeAFSDB
	_
	_
	_
	_
	_
	TypeSIG
	TypeKEY
	_
	_
	TypeAAAA
)

// QuestionTypes appear in the question part of a query.  They are a
// superset of ResourceRecordTypes, hence all ResourceRecordTypes are valid QuestionTypes.
const (
	TypeAXFR  uint16 = iota + 252 // a request for a transfer of an entire zone
	TypeMAILB                     // a request for mailbox-related records
	TypeMAILA                     // a request for mail agent RRs
	TypeALL                       // a request for all records
)

// ResourceRecordClasses appear in resource records.  Note that these types are a
// subset of QuestionClasses.
const (
	ClassIN uint16 = iota + 1 // the Internet
	ClassCS                   // the CSNET class
	ClassCH                   // the CHAOS class
	ClassHS                   // Hesiod
)

// QuestionClasses appear in the question section of a query.  They are a
// superset of ResourceRecordClasses, every ResourceRecordClasses is a valid QuestionClasses.
const (
	ClassALL uint16 = 255 // any class
)

// Query entry is used to carry the question section in most queries,
// i.e., the parameters that define what is being asked.
type Query struct {
	// An owner name, i.e., the name of the node to which this
	// resource record pertains.
	Domain
	// Two octets containing one of the ResourceRecordType codes.
	Type uint16
	// Two octets containing one of the ResourceRecordClass codes.
	Class uint16
}

func (q *Query) ReadFrom(r io.Reader) (n int64, err error) {
	n, err = q.Domain.ReadFrom(r)
	if err != nil {
		return
	}
	err = binary.Read(r, binary.BigEndian, &q.Type)
	if err != nil {
		return
	}
	err = binary.Read(r, binary.BigEndian, &q.Class)
	if err != nil {
		return
	}
	n += 4
	return
}

// WriteTo writes query into writer.
func (q *Query) WriteTo(w io.Writer) (n int64, err error) {
	n, err = q.Domain.WriteTo(w)
	if err != nil {
		return
	}
	err = binary.Write(w, binary.BigEndian, q.Type)
	if err != nil {
		return
	}
	err = binary.Write(w, binary.BigEndian, q.Class)
	if err != nil {
		return
	}
	n += 4
	return
}

// ResourceRecord entry is used to carry the answer, authority, and additional sections,
// all share the same format: a variable number of resource records, where the number of
// records is specified in the corresponding count field in the header.
type ResourceRecord struct {
	Query
	// A 32 bit unsigned integer that specifies the time
	// interval (in seconds) that the resource record may be
	// cached before it should be discarded.  Zero values are
	// interpreted to mean that the RR can only be used for the
	// transaction in progress, and should not be cached.
	TimeToLive Uint32
	// an unsigned 16 bit integer that specifies the length in
	// octets of the DATA field.
	Length Uint16
}

func (rr *ResourceRecord) ReadFrom(r io.Reader) (n int64, err error) {
	n, err = rr.Query.ReadFrom(r)
	if err != nil {
		return
	}
	err = binary.Read(r, binary.BigEndian, &rr.TimeToLive)
	if err != nil {
		return
	}
	err = binary.Read(r, binary.BigEndian, &rr.Length)
	if err != nil {
		return
	}
	n += 6
	return
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecord) WriteTo(w io.Writer) (n int64, err error) {
	n, err = rr.Query.WriteTo(w)
	if err != nil {
		return
	}
	err = binary.Write(w, binary.BigEndian, rr.TimeToLive)
	if err != nil {
		return
	}
	err = binary.Write(w, binary.BigEndian, rr.Length)
	if err != nil {
		return
	}
	n += 6
	return
}

type ResourceRecordCNAME struct {
	ResourceRecord
	// A domain name which specifies the canonical or primary
	// name for the owner.  The owner name is an alias.
	Canonical Domain
}

func (rr *ResourceRecordCNAME) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordCNAME) ReadFrom(r io.Reader) (n int64, err error) {
	return rr.Canonical.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordCNAME) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.Canonical}.WriteTo(w)
}

type ResourceRecordHINFO struct {
	ResourceRecord
	// A name which specifies the CPU type.
	CPU Label
	// A name which specifies the operating system type.
	OS Label
}

func (rr *ResourceRecordHINFO) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordHINFO) ReadFrom(r io.Reader) (n int64, err error) {
	return Section{
		&rr.CPU,
		&rr.OS}.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordHINFO) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.CPU,
		&rr.OS}.WriteTo(w)
}

type ResourceRecordMINFO struct {
	ResourceRecord
	// A domain name which specifies a mailbox which is
	// responsible for the mailing list or mailbox.  If this
	// domain name names the root, the owner of the MINFO RR is
	// responsible for itself.
	ResponsibleMailbox Domain
	// A domain name which specifies a mailbox which is to
	// receive error messages related to the mailing list or
	// mailbox specified by the owner of the MINFO RR.
	ErrorMailbox Domain
}

func (rr *ResourceRecordMINFO) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordMINFO) ReadFrom(r io.Reader) (n int64, err error) {
	return Section{
		&rr.ResponsibleMailbox,
		&rr.ErrorMailbox}.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordMINFO) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.ResponsibleMailbox,
		&rr.ErrorMailbox}.WriteTo(w)
}

type ResourceRecordMB struct {
	ResourceRecord
	// A domain name which specifies a host which has the
	// specified mailbox.
	Host Domain
}

func (rr *ResourceRecordMB) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordMB) ReadFrom(r io.Reader) (n int64, err error) {
	return rr.Host.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordMB) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.Host}.WriteTo(w)
}

type ResourceRecordMD struct {
	ResourceRecord
	// A domain name which specifies a host which has a mail
	// agent for the domain which should be able to deliver
	// mail for the domain.
	Host Domain
}

func (rr *ResourceRecordMD) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordMD) ReadFrom(r io.Reader) (n int64, err error) {
	return rr.Host.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordMD) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.Host}.WriteTo(w)
}

type ResourceRecordMF struct {
	ResourceRecord
	// A domain name which specifies a host which has a mail
	// agent for the domain which will accept mail for
	// forwarding to the domain.
	Host Domain
}

func (rr *ResourceRecordMF) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordMF) ReadFrom(r io.Reader) (n int64, err error) {
	return rr.Host.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordMF) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.Host}.WriteTo(w)
}

type ResourceRecordMG struct {
	ResourceRecord
	// A domain name which specifies a mailbox which is a
	// member of the mail group specified by the domain name.
	Mailbox Domain
}

func (rr *ResourceRecordMG) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordMG) ReadFrom(r io.Reader) (n int64, err error) {
	return rr.Mailbox.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordMG) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.Mailbox}.WriteTo(w)
}

type ResourceRecordMR struct {
	ResourceRecord
	//A domain name which specifies a mailbox which is the
	// proper rename of the specified mailbox.
	Mailbox Domain
}

func (rr *ResourceRecordMR) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordMR) ReadFrom(r io.Reader) (n int64, err error) {
	return rr.Mailbox.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordMR) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.Mailbox}.WriteTo(w)
}

type ResourceRecordMX struct {
	ResourceRecord
	// A 16 bit integer which specifies the preference given to
	// this RR among others at the same owner.  Lower values
	// are preferred.
	Preference Int16
	// A domain name which specifies a host willing to act as
	// a mail exchange for the owner name.
	Exchange Domain
}

func (rr *ResourceRecordMX) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordMX) ReadFrom(r io.Reader) (n int64, err error) {
	return Section{
		&rr.Preference,
		&rr.Exchange}.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordMX) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.Preference,
		&rr.Exchange}.WriteTo(w)
}

type Int16 int16

func (i *Int16) ReadFrom(r io.Reader) (n int64, err error) {
	return 2, binary.Read(r, binary.BigEndian, i)
}

// WriteTo writes resource record into writer.
func (i *Int16) WriteTo(w io.Writer) (n int64, err error) {
	return 2, binary.Write(w, binary.BigEndian, i)
}

type ResourceRecordNS struct {
	ResourceRecord
	// A domain name which specifies a host which should be
	// authoritative for the specified class and domain.
	Authoritative Domain
}

func (rr *ResourceRecordNS) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordNS) ReadFrom(r io.Reader) (n int64, err error) {
	return rr.Authoritative.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordNS) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.Authoritative}.WriteTo(w)
}

type ResourceRecordPTR struct {
	ResourceRecord
	// A domain name which points to some location in the
	// domain name space.
	Location Domain
}

func (rr *ResourceRecordPTR) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordPTR) ReadFrom(r io.Reader) (n int64, err error) {
	return rr.Location.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordPTR) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.Location}.WriteTo(w)
}

type ResourceRecordSOA struct {
	ResourceRecord
	// The domain name of the name server that was the
	// original or primary source of data for this zone.
	Original Domain
	// A domain name which specifies the mailbox of the
	// person responsible for this zone.
	Mailbox Domain
	// The unsigned 32 bit version number of the original copy
	// of the zone.  Zone transfers preserve this value.This
	// value wraps and should be compared using sequence space
	// arithmetic.
	Serial Uint32
	// A 32 bit time interval before the zone should be
	// refreshed. All times are in units of seconds.
	Refresh Uint32
	// A 32 bit time interval that should elapse before a
	// failed refresh should be retried.
	Retry Uint32
	// A 32 bit time value that specifies the upper limit on
	// the time interval that can elapse before the zone is no
	// longer authoritative.
	Expire Uint32
	// The unsigned 32 bit minimum TTL field that should be
	// exported with any RR from this zone.
	Minimum Uint32
}

func (rr *ResourceRecordSOA) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordSOA) ReadFrom(r io.Reader) (n int64, err error) {
	return Section{
		&rr.Original,
		&rr.Mailbox,
		&rr.Serial,
		&rr.Refresh,
		&rr.Retry,
		&rr.Expire,
		&rr.Minimum}.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordSOA) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.Original,
		&rr.Mailbox,
		&rr.Serial,
		&rr.Refresh,
		&rr.Retry,
		&rr.Expire,
		&rr.Minimum}.WriteTo(w)
}

type Uint16 uint16

func (u *Uint16) ReadFrom(r io.Reader) (n int64, err error) {
	return 2, binary.Read(r, binary.BigEndian, u)
}

// WriteTo writes resource record into writer.
func (u *Uint16) WriteTo(w io.Writer) (n int64, err error) {
	return 2, binary.Write(w, binary.BigEndian, u)
}

type Uint32 uint32

func (u *Uint32) ReadFrom(r io.Reader) (n int64, err error) {
	return 4, binary.Read(r, binary.BigEndian, u)
}

// WriteTo writes resource record into writer.
func (u *Uint32) WriteTo(w io.Writer) (n int64, err error) {
	return 4, binary.Write(w, binary.BigEndian, u)
}

type ResourceRecordTXT struct {
	ResourceRecord
	// One or more labels.
	Text Domain
}

func (rr *ResourceRecordTXT) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordTXT) ReadFrom(r io.Reader) (n int64, err error) {
	return rr.Text.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordTXT) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.Text}.WriteTo(w)
}

type ResourceRecordA struct {
	ResourceRecord
	// A 32 bit Internet address.
	IP IPv4
}

func (rr *ResourceRecordA) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordA) ReadFrom(r io.Reader) (n int64, err error) {
	return rr.IP.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordA) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.IP}.WriteTo(w)
}

type IPv4 [4]byte

func (b *IPv4) ReadFrom(r io.Reader) (n int64, err error) {
	return 4, binary.Read(r, binary.BigEndian, b[:])
}

func (b *IPv4) WriteTo(w io.Writer) (n int64, err error) {
	return 4, binary.Write(w, binary.BigEndian, b[:])
}

type ResourceRecordAAAA struct {
	ResourceRecord
	// A 128 bit Internet address.
	IP IPv6
}

func (rr *ResourceRecordAAAA) GoString() string {
	return fmt.Sprintf("%#v", *rr)
}

func (rr *ResourceRecordAAAA) ReadFrom(r io.Reader) (n int64, err error) {
	return rr.IP.ReadFrom(r)
}

// WriteTo writes resource record into writer.
func (rr *ResourceRecordAAAA) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&rr.ResourceRecord,
		&rr.IP}.WriteTo(w)
}

type IPv6 [16]byte

func (b *IPv6) ReadFrom(r io.Reader) (n int64, err error) {
	return 16, binary.Read(r, binary.BigEndian, b[:])
}

func (b *IPv6) WriteTo(w io.Writer) (n int64, err error) {
	return 16, binary.Write(w, binary.BigEndian, b[:])
}

type Question []Query

func (q *Question) ReadFrom(r io.Reader) (n int64, err error) {
	var i int64
	for j := range *q {
		i, err = (*q)[j].ReadFrom(r)
		if err != nil {
			return
		}
		n += i
	}
	return
}

// WriteTo writes questions into writer.
func (q *Question) WriteTo(w io.Writer) (n int64, err error) {
	var i int64
	for _, query := range *q {
		i, err = query.WriteTo(w)
		if err != nil {
			return
		}
		n += i
	}
	return
}

type Reply interface {
	io.ReaderFrom
	io.WriterTo
}

type Section []Reply

func (s Section) ReadFrom(r io.Reader) (n int64, err error) {
	var i int64
	for j := range s {
		var rr ResourceRecord
		i, err = rr.ReadFrom(r)
		if err != nil {
			return
		}
		n += i
		switch rr.Type {
		case TypeA:
			s[j] = &ResourceRecordA{ResourceRecord: rr}
		case TypeNS:
			s[j] = &ResourceRecordNS{ResourceRecord: rr}
		//case TypeMD:
		//case TypeMF:
		case TypeCNAME:
			s[j] = &ResourceRecordCNAME{ResourceRecord: rr}
		case TypeSOA:
			s[j] = &ResourceRecordSOA{ResourceRecord: rr}
		//case TypeMB:
		//case TypeMG:
		//case TypeMR:
		//case TypeWKS:
		case TypePTR:
			s[j] = &ResourceRecordPTR{ResourceRecord: rr}
		//case TypeHINFO:
		//case TypeMINFO:
		case TypeMX:
			s[j] = &ResourceRecordMX{ResourceRecord: rr}
		case TypeTXT:
			s[j] = &ResourceRecordTXT{ResourceRecord: rr}
		//case TypePR:
		//case TypeAFSDB:
		//case TypeSIG:
		//case TypeKEY:
		case TypeAAAA:
			s[j] = &ResourceRecordAAAA{ResourceRecord: rr}
		default:
			i, err = io.CopyN(ioutil.Discard, r, int64(rr.Length))
			if err != nil {
				return
			}
			n += i
			continue
		}
		i, err = s[j].ReadFrom(r)
		if err != nil {
			return
		}
		n += i
	}
	return
}

// WriteTo writes slice of writable to writer.
func (s Section) WriteTo(w io.Writer) (n int64, err error) {
	var i int64
	for _, reply := range s {
		i, err = reply.WriteTo(w)
		if err != nil {
			return
		}
		n += i
	}
	return
}

// All communications inside of the domain protocol are carried in a single
// format called a message.  The top level format of message is divided
// into 5 sections, some of which are empty in certain cases
type Message struct {
	Header
	Question           // the question for the name server
	Answer     Section // RRs answering the question
	Authority  Section // RRs pointing toward an authority
	Additional Section // RRs holding additional information
}

func (m *Message) ReadFrom(r io.Reader) (n int64, err error) {
	var i int64
	i, err = m.Header.ReadFrom(r)
	if err != nil {
		return
	}
	n += i
	m.Question = make([]Query, m.QuestionCount)
	i, err = m.Question.ReadFrom(r)
	if err != nil {
		return
	}
	n += i
	m.Answer = make([]Reply, m.AnswerCount)
	i, err = m.Answer.ReadFrom(r)
	if err != nil {
		return
	}
	n += i
	m.Authority = make([]Reply, m.AuthorityCount)
	i, err = m.Authority.ReadFrom(r)
	if err != nil {
		return
	}
	n += i
	m.Additional = make([]Reply, m.AdditionalCount)
	i, err = m.Additional.ReadFrom(r)
	if err != nil {
		return
	}
	n += i
	return
}

// WriteTo writes message into writer.
func (m *Message) WriteTo(w io.Writer) (n int64, err error) {
	return Section{
		&m.Header,
		&m.Question,
		m.Answer,
		m.Authority,
		m.Additional}.WriteTo(w)
}

func (m *Message) MarshalBinary() (data []byte, err error) {
	b := bytes.Buffer{}
	_, err = m.WriteTo(&b)
	if err != nil {
		return
	}
	data = b.Bytes()
	return
}

func (m *Message) UnmarshalBinary(data []byte) (err error) {
	_, err = m.ReadFrom(bytes.NewReader(data))
	if err == nil {
		err = m.Error()
	}
	return
}

func (m *Message) AddQuery(query Query) {
	m.Question = append(m.Question, query)
	m.QuestionCount++
}

func (m *Message) SetRecursionDesired(yes bool) {
	m.IsRecursionDesired = yes
}

func (m *Message) SetRandomID() {
	m.ID = uint16(rand.Int())
}

func (m *Message) SetID(id uint16) {
	if id > 0 {
		m.ID = id
		return
	}
	m.SetRandomID()
}

func (m *Message) SetType(op uint8) {
	m.Opcode = op
}

// NewDomain returns a new domain
func NewDomain(host string) (domain Domain) {
	if !strings.HasSuffix(host, ".") {
		host += "."
	}
	for _, name := range strings.Split(host, ".") {
		domain = append(domain, NewLabel(name))
	}
	return domain
}

// NewLabel returns a new label
func NewLabel(name string) Label {
	return Label{
		Length: uint8(len(name)),
		Name:   []byte(name),
	}
}

// NewQuery returns a new query
func NewQuery(domain Domain, typo uint16, class uint16) Query {
	return Query{
		Domain: domain,
		Type:   typo,
		Class:  class,
	}
}

// Write
func (m *Message) Write(c net.Conn) (int, error) {
	b, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	return c.Write(b)
}

// Read
func (m *Message) Read(c net.Conn) error {
	b := make([]byte, 2048)
	n, err := c.Read(b)
	if err != nil {
		return err
	}
	return m.UnmarshalBinary(b[:n])
}

var errors = map[uint8]error{
	CodeNoError:        nil,
	CodeFormatError:    fmt.Errorf("the name server was unable to interpret the query"),
	CodeServerFailure:  fmt.Errorf("the name server was unable to process this query"),
	CodeNameError:      fmt.Errorf("the domain name referenced in the query does not exist"),
	CodeNotImplemented: fmt.Errorf("the name server does not support the requested query"),
	CodeRefused:        fmt.Errorf("the name server refuses to perform the specified operation"),
}

func (m *Message) Error() error {
	return errors[m.ResponseCode]
}

func (m *Message) Section() []Section {
	return []Section{m.Answer, m.Authority, m.Additional}
}

func (m *Message) A() (ip []net.IP) {
	for _, s := range m.Section() {
		for _, rr := range s {
			switch v := rr.(type) {
			case *ResourceRecordA:
				ip = append(ip, v.IP[:])
			case *ResourceRecordAAAA:
				ip = append(ip, v.IP[:])
			}
		}
	}
	return
}

func (m *Message) CNAME() (name []string) {
	for _, s := range m.Section() {
		for _, rr := range s {
			switch v := rr.(type) {
			case *ResourceRecordCNAME:
				name = append(name, strings.Join(v.Canonical.Name(), "."))
			}
		}
	}
	return
}
