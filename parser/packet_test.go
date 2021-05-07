package parser

var tests = []struct {
	Name   string
	Header Header
	Event  string
	Var    []interface{}
	Data   [][]byte
}{
	{"Empty",
		Header{Connect, 0, false, "", ""},
		"",
		nil,
		[][]byte{
			[]byte("0"),
		},
	},
	{"Data",
		Header{Error, 0, false, "", ""},
		"",
		[]interface{}{"error"},
		[][]byte{
			[]byte("4[\"error\"]\n"),
		},
	},
	{"BData",
		Header{Event, 0, false, "", ""},
		"msg",
		[]interface{}{
			&Buffer{Data: []byte{1, 2, 3}},
		},
		[][]byte{
			[]byte("51-[\"msg\",{\"_placeholder\":true,\"num\":0}]\n"),
			{1, 2, 3},
		},
	},
	{"ID",
		Header{Connect, 0, true, "", ""},
		"",
		nil,
		[][]byte{
			[]byte("00"),
		},
	},
	{"IDData",
		Header{Ack, 13, true, "", ""},
		"",
		[]interface{}{"error"},
		[][]byte{
			[]byte("313[\"error\"]\n"),
		},
	},
	{"IDBData",
		Header{Ack, 13, true, "", ""},
		"",
		[]interface{}{
			&Buffer{
				Data: []byte{1, 2, 3},
			},
		},
		[][]byte{
			[]byte("61-13[{\"_placeholder\":true,\"num\":0}]\n"),
			{1, 2, 3},
		},
	},
	{"Namespace",
		Header{Disconnect, 0, false, "/woot", ""},
		"",
		nil,
		[][]byte{
			[]byte("1/woot"),
		},
	},
	{"NamespaceData",
		Header{Event, 0, false, "/woot", ""},
		"msg",
		[]interface{}{
			1,
		},
		[][]byte{
			[]byte("2/woot,[\"msg\",1]\n"),
		},
	},
	{"NamespaceBData",
		Header{Event, 0, false, "/woot", ""},
		"msg",
		[]interface{}{
			&Buffer{Data: []byte{2, 3, 4}},
		},
		[][]byte{
			[]byte("51-/woot,[\"msg\",{\"_placeholder\":true,\"num\":0}]\n"),
			{2, 3, 4},
		},
	},
	{"NamespaceID",
		Header{Disconnect, 1, true, "/woot", ""},
		"",
		nil,
		[][]byte{
			[]byte("1/woot,1"),
		},
	},
	{"NamespaceIDData",
		Header{Event, 1, true, "/woot", ""},
		"msg",
		[]interface{}{
			1,
		},
		[][]byte{
			[]byte("2/woot,1[\"msg\",1]\n"),
		},
	},
	{"NamespaceIDBData", Header{Event, 1, true, "/woot", ""},
		"msg",
		[]interface{}{
			&Buffer{Data: []byte{2, 3, 4}},
		},
		[][]byte{
			[]byte("51-/woot,1[\"msg\",{\"_placeholder\":true,\"num\":0}]\n"),
			{2, 3, 4},
		},
	},
}
