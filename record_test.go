package multidns

import (
	"testing"
)

func BenchmarkMessage_MarshalBinary(b *testing.B) {
	m := Message{
		Header: Header{
			ID:                   1,
			IsReply:              true,
			Opcode:               1,
			IsAuthoritative:      true,
			IsTruncated:          true,
			IsRecursionDesired:   true,
			IsRecursionAvailable: true,
			ResponseCode:         0,
			QuestionCount:        1,
			AnswerCount:          1,
			AuthorityCount:       1,
			AdditionalCount:      1,
		},
		Question: []Query{
			NewQuery(NewDomain("aaa.bbb.ccc"), TypeA, ClassIN)},
		Answer: []Reply{&ResourceRecordA{
			ResourceRecord: ResourceRecord{
				Query:      NewQuery(NewDomain("aaa.bbb.ccc"), TypeA, ClassIN),
				TimeToLive: 1,
				Length:     4,
			},
			IP: [4]byte{1, 2, 3, 4}}},
		Authority: []Reply{&ResourceRecordCNAME{
			ResourceRecord: ResourceRecord{
				Query:      NewQuery(NewDomain("aaa.bbb.ccc"), TypeA, ClassIN),
				TimeToLive: 1,
				Length:     111,
			},
			Canonical: NewDomain("aaa.bbb.ccc"),
		}},
		Additional: []Reply{},
	}
	for i := 0; i < b.N; i++ {
		data, err := m.MarshalBinary()
		if err != nil {
			b.Fatal(err)
		}
		err = m.UnmarshalBinary(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
