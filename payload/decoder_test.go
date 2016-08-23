package payload

//
// func TestDecoderPartRead(t *testing.T) {
// 	at := assert.New(t)
// 	tests := []struct {
// 		supportBinary bool
// 		packets       []Packet
// 		data          []byte
// 	}{
// 		{true, []Packet{
// 			{base.FrameString, base.OPEN, []byte{}},
// 			{base.FrameBinary, base.MESSAGE, []byte("hel")},
// 			{base.FrameString, base.MESSAGE, []byte("你")},
// 			{base.FrameBinary, base.MESSAGE, []byte("hel")},
// 			{base.FrameString, base.MESSAGE, []byte("你")},
// 			{base.FrameString, base.PING, []byte("pro")},
// 		}, []byte{
// 			0x00, 0x01, 0xff, '0',
// 			0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
// 			0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
// 			0x01, 0x07, 0xff, 0x04, 'h', 'e', 'l', 'l', 'o', '\n',
// 			0x00, 0x08, 0xff, '4', 0xe4, 0xbd, 0xa0, 0xe5, 0xa5, 0xbd, '\n',
// 			0x00, 0x06, 0xff, '2', 'p', 'r', 'o', 'b', 'e',
// 		}},
//
// 		{false, []Packet{
// 			{base.FrameString, base.OPEN, []byte{}},
// 			{base.FrameBinary, base.MESSAGE, []byte("hel")},
// 			{base.FrameString, base.MESSAGE, []byte("你")},
// 			{base.FrameBinary, base.MESSAGE, []byte("hel")},
// 			{base.FrameString, base.MESSAGE, []byte("你")},
// 			{base.FrameString, base.PING, []byte("pro")},
// 		}, []byte("1:010:b4aGVsbG8K8:4你好\n10:b4aGVsbG8K8:4你好\n6:2probe")},
// 	}
//
// 	for _, test := range tests {
// 		conn := newFakeReader(test.supportBinary, test.data)
// 		r := NewDecoder(conn)
// 		var packets []Packet
// 		for {
// 			ft, pt, fr, err := r.NextReader()
// 			if err != nil {
// 				at.Equal(io.EOF, err)
// 				break
// 			}
// 			var data [3]byte
// 			n, err := io.ReadFull(fr, data[:])
// 			if err == io.EOF {
// 				n = 0
// 			} else {
// 				at.Nil(err)
// 			}
// 			packet := Packet{
// 				ft:   ft,
// 				pt:   pt,
// 				data: data[:n],
// 			}
// 			packets = append(packets, packet)
// 		}
// 		at.Equal(test.packets, packets)
// 	}
// }
//
