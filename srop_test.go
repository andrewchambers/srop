package srop

import (
	"bytes"
	"context"
	"io"
	"log"
	"reflect"
	"testing"
	"time"
)

const (
	TYPE_TESTFOO = 0xfaeaaba3127e15c0
	TYPE_TESTBAR = 0xd669132bfbb9724c
)

type Foo struct {
	X byte
}

func (f *Foo) SropType() uint64              { return TYPE_TESTFOO }
func (f *Foo) SropMarshal() []byte           { return []byte{f.X} }
func (f *Foo) SropUnmarshal(buf []byte) bool { f.X = buf[0]; return true }

type Bar struct {
}

func (b *Bar) SropType() uint64              { return TYPE_TESTBAR }
func (b *Bar) SropMarshal() []byte           { return []byte{} }
func (b *Bar) SropUnmarshal(buf []byte) bool { return true }

type TestObject struct {
	clunked    bool
	GotUnknown bool
}

func (to *TestObject) Message(ctx context.Context, s *ConnServer, m Message, respond RespondFunc) {

	log.Printf("handling request %#v", m)

	switch m := m.(type) {
	case *Bar:
		s.Go(func() {
			time.Sleep(1*time.Second + 50*time.Millisecond)
			respond(m)
		})
	case *Foo:
		s.Go(func() {
			m.X += 1
			respond(m)
			log.Printf("clunking root object")
			s.Clunk(0)
			time.Sleep(100 * time.Millisecond)
		})
	default:
		panic(m)
	}

}

func (to *TestObject) UnknownMessage(ctx context.Context, s *ConnServer, t uint64, buf []byte, respond RespondFunc) {
	to.GotUnknown = true
	respond(&UnexpectedMessage{})
}

func (to *TestObject) Clunk(s *ConnServer) {
	log.Printf("root object clunk called")
	to.clunked = true
}

func TestBasicServerRequestResponse(t *testing.T) {

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	a, b := ConnectedPipes()

	reg := NewRegistry()
	RegisterStandardMessagesAndErrors(reg)
	reg.RegisterMessage(TYPE_TESTFOO, func() Message { return &Foo{} })

	to := &TestObject{}
	s := NewConnServer(a, ConnServerOptions{
		Registry:      reg,
		BootstrapFunc: func(c io.ReadWriteCloser) Object { return to },
	})

	go s.Serve(ctx, a)

	foo := &Foo{
		X: 3,
	}

	client := NewClient(b, ClientOptions{
		Registry: reg,
	})

	m, _ := client.RawSendParsedReply(reg, BOOTSTRAP_OBJECT_ID, 12345, []byte{})
	_ = m.(*UnexpectedMessage)

	if to.GotUnknown != true {
		t.Fatal("expected unknown message...")
	}

	m, err := client.SendWithReg(reg, BOOTSTRAP_OBJECT_ID, foo)
	if err != nil {
		t.Fatal(err)
	}

	replyFoo := m.(*Foo)

	if replyFoo.X != 4 {
		t.Fatal(replyFoo.X)
	}

	client.Close()

	cancelCtx()
	s.Wait()
}

func TestConcurrencyLimits(t *testing.T) {

	startT := time.Now()

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	a, b := ConnectedPipes()

	reg := NewRegistry()
	RegisterStandardMessagesAndErrors(reg)
	reg.RegisterMessage(TYPE_TESTBAR, func() Message { return &Bar{} })

	to := &TestObject{}
	s := NewConnServer(a, ConnServerOptions{
		Registry:               reg,
		MaxOutstandingRequests: 1,
		BootstrapFunc:          func(c io.ReadWriteCloser) Object { return to },
	})

	go s.Serve(ctx, a)

	client := NewClient(b, ClientOptions{
		Registry: reg,
	})

	errs := make(chan error, 2)

	// Testing max outstanding.
	go func() {
		_, err := client.SendWithReg(reg, BOOTSTRAP_OBJECT_ID, &Bar{})
		errs <- err
		t.Log("Call1 complete.")
	}()

	// Testing max outstanding.
	go func() {
		_, err := client.SendWithReg(reg, BOOTSTRAP_OBJECT_ID, &Bar{})
		errs <- err
		t.Log("Call2 complete.")
	}()

	err := <-errs
	if err != nil {
		t.Fatal(err)
	}
	err = <-errs
	if err != nil {
		t.Fatal(err)
	}

	client.Close()

	cancelCtx()
	s.Wait()

	endT := time.Now()
	testDuration := endT.Sub(startT)

	seconds := testDuration.Seconds()
	if int(seconds) != 2 {
		t.Fatalf("The test should be between 2 seconds if concurrency limits are being obeyed. t=%f", seconds)
	}

}

func TestResponseReadWrite(t *testing.T) {
	var b bytes.Buffer

	r1 := Response{
		RequestId:    123,
		ResponseType: 456,
		ResponseData: []byte{7, 8, 9},
	}

	err := WriteResponse(&b, r1)
	if err != nil {
		t.Fatal(err)
	}

	r2, err := ReadResponse(&b, 9999)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(r1, r2) {
		t.Fatalf("%v != %v", r1, r2)
	}
}

func TestRequestReadWrite(t *testing.T) {
	var b bytes.Buffer

	r1 := Request{
		ObjectId:    123,
		RequestId:   456,
		MessageType: 789,
		MessageData: []byte{10, 11, 12},
	}

	err := WriteRequest(&b, r1)
	if err != nil {
		t.Fatal(err)
	}

	r2, err := ReadRequest(&b, 9999)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(r1, r2) {
		t.Fatalf("%v != %v", r1, r2)
	}
}

// Some helpers, manually vendored

type MergedReadWriteCloser struct {
	RC io.ReadCloser
	WC io.WriteCloser
}

func (m *MergedReadWriteCloser) Read(buf []byte) (int, error) {
	return m.RC.Read(buf)
}

func (m *MergedReadWriteCloser) Write(buf []byte) (int, error) {
	return m.WC.Write(buf)
}

func (m *MergedReadWriteCloser) Close() error {
	_ = m.RC.Close()
	_ = m.WC.Close()
	return nil
}

func ConnectedPipes() (*MergedReadWriteCloser, *MergedReadWriteCloser) {
	a, b := io.Pipe()
	x, y := io.Pipe()

	return &MergedReadWriteCloser{
			RC: a,
			WC: y,
		}, &MergedReadWriteCloser{
			RC: x,
			WC: b,
		}
}
