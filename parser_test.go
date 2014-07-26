package engineio

import (
	"bytes"
	"io"
	"runtime"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPacketType(t *testing.T) {

	Convey("Byte to type", t, func() {

		Convey("Open", func() {
			t, err := byteToType(0)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, OPEN)
		})

		Convey("Close", func() {
			t, err := byteToType(1)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, CLOSE)
		})

		Convey("Ping", func() {
			t, err := byteToType(2)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, PING)
		})

		Convey("Pong", func() {
			t, err := byteToType(3)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, PONG)
		})

		Convey("Message", func() {
			t, err := byteToType(4)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, MESSAGE)
		})

		Convey("Upgrade", func() {
			t, err := byteToType(5)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, UPGRADE)
		})

		Convey("Noop", func() {
			t, err := byteToType(6)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, NOOP)
		})

		Convey("Error", func() {
			_, err := byteToType(7)
			So(err, ShouldNotBeNil)
		})

	})

	Convey("Type to byte", t, func() {

		Convey("Open", func() {
			So(OPEN.Byte(), ShouldEqual, 0)
		})

		Convey("Close", func() {
			So(CLOSE.Byte(), ShouldEqual, 1)
		})

		Convey("Ping", func() {
			So(PING.Byte(), ShouldEqual, 2)
		})

		Convey("Pong", func() {
			So(PONG.Byte(), ShouldEqual, 3)
		})

		Convey("Message", func() {
			So(MESSAGE.Byte(), ShouldEqual, 4)
		})

		Convey("Upgrade", func() {
			So(UPGRADE.Byte(), ShouldEqual, 5)
		})

		Convey("Noop", func() {
			So(NOOP.Byte(), ShouldEqual, 6)
		})

	})

}

func TestStringParser(t *testing.T) {
	type Test struct {
		name   string
		t      PacketType
		data   []byte
		output string
	}
	var tests = []Test{
		{"without data", OPEN, nil, "0"},
		{"with data", MESSAGE, []byte("测试"), "\x34\xe6\xb5\x8b\xe8\xaf\x95"},
	}

	for _, test := range tests {
		buf := bytes.NewBuffer(nil)

		Convey("Given a packet type "+test.name, t, func() {

			Convey("Create encoder", func() {
				encoder, err := newStringEncoder(buf, test.t)
				So(err, ShouldBeNil)
				So(encoder, ShouldImplement, (*io.WriteCloser)(nil))

				Convey("Encoded", func() {
					for d := test.data; len(d) > 0; {
						n, err := encoder.Write(d)
						So(err, ShouldBeNil)
						d = d[n:]
					}

					Convey("End", func() {
						err := encoder.Close()
						So(err, ShouldBeNil)
						So(buf.String(), ShouldEqual, test.output)
					})
				})
			})

			Convey("Create decoder", func() {
				decoder, err := newDecoder(buf)
				So(err, ShouldBeNil)
				So(decoder, ShouldImplement, (*io.ReadCloser)(nil))
				So(decoder.MessageType(), ShouldEqual, MessageText)

				Convey("Decoded", func() {
					So(decoder.Type(), ShouldEqual, test.t)

					decoded := make([]byte, len(test.data)+1)
					n, err := decoder.Read(decoded)
					if n > 0 {
						So(err, ShouldBeNil)
						So(decoded[:n], ShouldResemble, test.data)
					}

					Convey("End", func() {
						_, err := decoder.Read(decoded[:])
						So(err, ShouldEqual, io.EOF)
					})
				})
			})
		})
	}
}

func TestBinaryParser(t *testing.T) {
	type Test struct {
		name   string
		t      PacketType
		data   []byte
		output string
	}
	var tests = []Test{
		{"without data", OPEN, nil, "\x00"},
		{"with data", MESSAGE, []byte("测试"), "\x04\xe6\xb5\x8b\xe8\xaf\x95"},
	}
	for _, test := range tests {
		buf := bytes.NewBuffer(nil)

		Convey("Given a packet type "+test.name, t, func() {

			Convey("Create Encoder", func() {
				encoder, err := newBinaryEncoder(buf, test.t)
				So(err, ShouldBeNil)
				So(encoder, ShouldImplement, (*io.WriteCloser)(nil))

				Convey("Encoded", func() {
					for d := test.data; len(d) > 0; {
						n, err := encoder.Write(d)
						So(err, ShouldBeNil)
						d = d[n:]
					}

					Convey("End", func() {
						err := encoder.Close()
						So(err, ShouldBeNil)
						So(buf.String(), ShouldEqual, test.output)
					})
				})
			})

			Convey("Create decoder", func() {
				decoder, err := newDecoder(buf)
				So(err, ShouldBeNil)
				So(decoder, ShouldImplement, (*io.ReadCloser)(nil))
				So(decoder.MessageType(), ShouldEqual, MessageBinary)

				Convey("Decoded", func() {
					So(decoder.Type(), ShouldEqual, test.t)
					decoded := make([]byte, len(test.data)+1)
					n, err := decoder.Read(decoded[:])
					if n > 0 {
						So(err, ShouldBeNil)
						So(decoded[:n], ShouldResemble, test.data)
					}

					Convey("End", func() {
						_, err := decoder.Read(decoded[:])
						So(err, ShouldEqual, io.EOF)
					})
				})
			})

		})
	}
}

func TestBase64Parser(t *testing.T) {
	type Test struct {
		name   string
		t      PacketType
		data   []byte
		output string
	}
	var tests = []Test{
		{"without data", OPEN, nil, "b0"},
		{"with data", MESSAGE, []byte("测试"), "b45rWL6K+V"},
	}
	for _, test := range tests {
		buf := bytes.NewBuffer(nil)

		Convey("Given a packet type "+test.name, t, func() {

			Convey("Create Encoder", func() {
				encoder, err := newB64Encoder(buf, test.t)
				So(err, ShouldBeNil)
				So(encoder, ShouldImplement, (*io.WriteCloser)(nil))

				Convey("Encoded", func() {
					for d := test.data; len(d) > 0; {
						n, err := encoder.Write(d)
						So(err, ShouldBeNil)
						d = d[n:]
					}

					Convey("End", func() {
						err := encoder.Close()
						So(err, ShouldBeNil)
						So(buf.String(), ShouldEqual, test.output)
					})
				})
			})

			Convey("Create decoder", func() {
				decoder, err := newDecoder(buf)
				So(err, ShouldBeNil)
				So(decoder, ShouldImplement, (*io.ReadCloser)(nil))
				So(decoder.MessageType(), ShouldEqual, MessageBinary)

				Convey("Decoded", func() {
					So(decoder.Type(), ShouldEqual, test.t)
					decoded := make([]byte, len(test.data)+1)
					n, err := decoder.Read(decoded[:])
					if n > 0 {
						So(err, ShouldBeNil)
						So(decoded[:n], ShouldResemble, test.data)
					}

					Convey("End", func() {
						_, err := decoder.Read(decoded[:])
						So(err, ShouldEqual, io.EOF)
					})
				})
			})

		})
	}
}

func TestStringPayload(t *testing.T) {
	type packet struct {
		Type     PacketType
		Data     []byte
		IsString bool
	}
	type Test struct {
		name    string
		packets []packet
		output  string
	}
	var tests = []Test{
		{"all in one", []packet{packet{OPEN, nil, true}, packet{MESSAGE, []byte("测试"), true}, packet{MESSAGE, []byte("测试"), false}}, "\x31\x3a\x30\x37\x3a\x34\xe6\xb5\x8b\xe8\xaf\x95\x31\x30\x3a\x62\x34\x35\x72\x57\x4c\x36\x4b\x2b\x56"},
	}
	for _, test := range tests {
		buf := bytes.NewBuffer(nil)

		Convey("Given an array of packet "+test.name, t, func() {

			Convey("Create encoder", func() {
				encoder := newStringPayloadEncoder()

				Convey("Encoded", func() {
					for _, p := range test.packets {
						var e io.WriteCloser
						var err error
						if p.IsString {
							e, err = encoder.NextString(p.Type)
						} else {
							e, err = encoder.NextBinary(p.Type)
						}
						So(err, ShouldBeNil)
						for d := p.Data; len(d) > 0; {
							n, err := e.Write(d)
							So(err, ShouldBeNil)
							d = d[n:]
						}
						err = e.Close()
						So(err, ShouldBeNil)
					}

					Convey("End", func() {
						err := encoder.EncodeTo(buf)
						So(err, ShouldBeNil)
						So(buf.String(), ShouldEqual, test.output)
					})
				})
			})

			Convey("Create decoder", func() {
				decoder := newPayloadDecoder(buf)

				Convey("Decode", func() {
					for i := 0; ; i++ {
						d, err := decoder.Next()
						if err == io.EOF {
							break
						}
						So(err, ShouldBeNil)
						So(d.Type(), ShouldEqual, test.packets[i].Type)

						if l := len(test.packets[i].Data); l > 0 {
							buf := make([]byte, len(test.packets[i].Data)+1)
							n, err := d.Read(buf)
							if n > 0 {
								So(err, ShouldBeNil)
								So(buf[:n], ShouldResemble, test.packets[i].Data)
							}
							_, err = d.Read(buf)
							So(err, ShouldEqual, io.EOF)
						}
						err = d.Close()
						So(err, ShouldBeNil)
					}
				})
			})
		})
	}
}

func TestBinaryPayload(t *testing.T) {
	type packet struct {
		Type     PacketType
		Data     []byte
		IsString bool
	}
	type Test struct {
		name    string
		packets []packet
		output  string
	}
	var tests = []Test{
		{"all in one", []packet{packet{OPEN, nil, true}, packet{MESSAGE, []byte("测试"), true}, packet{MESSAGE, []byte("测试"), false}}, "\x00\x01\xff\x30\x00\x07\xff\x34\xe6\xb5\x8b\xe8\xaf\x95\x01\x07\xff\x04\xe6\xb5\x8b\xe8\xaf\x95"},
	}
	for _, test := range tests {
		buf := bytes.NewBuffer(nil)

		Convey("Given an array of packet "+test.name, t, func() {

			Convey("Create encoder", func() {
				encoder := newBinaryPayloadEncoder()

				Convey("Encoded", func() {
					for _, p := range test.packets {
						var e io.WriteCloser
						var err error
						if p.IsString {
							e, err = encoder.NextString(p.Type)
						} else {
							e, err = encoder.NextBinary(p.Type)
						}
						So(err, ShouldBeNil)
						for d := p.Data; len(d) > 0; {
							n, err := e.Write(d)
							So(err, ShouldBeNil)
							d = d[n:]
						}
						err = e.Close()
						So(err, ShouldBeNil)
					}

					Convey("End", func() {
						err := encoder.EncodeTo(buf)
						So(err, ShouldBeNil)
						So(buf.String(), ShouldEqual, test.output)
					})
				})
			})

			Convey("Create decoder", func() {
				decoder := newPayloadDecoder(buf)

				Convey("Decode", func() {
					for i := 0; ; i++ {
						d, err := decoder.Next()
						if err == io.EOF {
							break
						}
						So(err, ShouldBeNil)
						So(d.Type(), ShouldEqual, test.packets[i].Type)

						if l := len(test.packets[i].Data); l > 0 {
							buf := make([]byte, len(test.packets[i].Data)+1)
							n, err := d.Read(buf)
							if n > 0 {
								So(err, ShouldBeNil)
								So(buf[:n], ShouldResemble, test.packets[i].Data)
							}
							_, err = d.Read(buf)
							So(err, ShouldEqual, io.EOF)
						}
						err = d.Close()
						So(err, ShouldBeNil)
					}
				})
			})
		})
	}
}

func TestLimitReaderDecoder(t *testing.T) {
	Convey("Test decoder with limit reader", t, func() {
		buf := bytes.NewBufferString("\x34\xe6\xb5\x8b\xe8\xaf\x95123")
		reader := newLimitReader(buf, 7)
		decoder, err := newDecoder(reader)
		So(err, ShouldBeNil)
		So(decoder.Type(), ShouldEqual, MESSAGE)
		err = decoder.Close()
		So(err, ShouldBeNil)
		So(buf.String(), ShouldEqual, "123")
	})
}

func TestParalletEncode(t *testing.T) {
	prev := runtime.GOMAXPROCS(10)
	defer runtime.GOMAXPROCS(prev)

	Convey("Test parallet encode", t, func() {
		c := make(chan int)
		max := 1000
		buf1 := bytes.NewBuffer(nil)
		buf2 := bytes.NewBuffer(nil)
		encoder := newStringPayloadEncoder()
		for i := 0; i < max; i++ {
			go func() {
				e, err := encoder.NextString(MESSAGE)
				So(err, ShouldBeNil)
				_, err = e.Write([]byte("1234"))
				So(err, ShouldBeNil)
				e.Close()
				c <- 1
			}()
		}
		for i := 0; i < max/2; i++ {
			<-c
		}
		err := encoder.EncodeTo(buf1)
		So(err, ShouldBeNil)
		for i := 0; i < max/2; i++ {
			<-c
		}
		err = encoder.EncodeTo(buf2)
		So(err, ShouldBeNil)

		for s := buf1.String(); len(s) > 0; {
			So(s, ShouldStartWith, "5:41234")
			s = s[len("5:41234"):]
		}
		for s := buf2.String(); len(s) > 0; {
			So(s, ShouldStartWith, "5:41234")
			s = s[len("5:41234"):]
		}
	})
}
