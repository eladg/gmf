package gmf_test

import (
	"github.com/chenhengjie123/gmf"
	"log"
	"testing"
)

var CodecCtxTestData = struct {
	width    int
	height   int
	timebase gmf.AVR
	pixfmt   int32
	bitrate  int
}{
	width:    100,
	height:   200,
	timebase: gmf.AVR{Num: 1, Den: 25},
	pixfmt:   gmf.AV_PIX_FMT_YUV420P,
	bitrate:  400000,
}

func TestCodecCtx(t *testing.T) {
	td := CodecCtxTestData

	codec, err := gmf.FindEncoder("mpeg4")
	if err != nil {
		t.Fatal(err)
	}

	cc := gmf.NewCodecCtx(codec)
	if cc == nil {
		t.Fatal("Unable to allocate codec context")
	}

	cc.SetWidth(td.width).SetHeight(td.height).SetTimeBase(td.timebase).SetPixFmt(td.pixfmt).SetBitRate(td.bitrate)

	if cc.Width() != td.width {
		t.Fatalf("Expected width = %v, %v got.\n", td.width, cc.Width())
	}

	if cc.Height() != td.height {
		t.Fatalf("Expected height = %v, %v got.\n", td.height, cc.Height())
	}

	if cc.TimeBase().AVR().Num != td.timebase.Num || cc.TimeBase().AVR().Den != td.timebase.Den {
		t.Fatalf("Expected AVR = %v, %v got", cc.TimeBase().AVR(), td.timebase)
	}

	if cc.PixFmt() != td.pixfmt {
		t.Fatalf("Expected pixfmt = %v, %v got.\n", td.pixfmt, cc.PixFmt())
	}

	if err := cc.Open(nil); err != nil {
		t.Fatal(err)
	}

	log.Println("CodecCtx is successfully created and opened.")

	cc.Free()
}
