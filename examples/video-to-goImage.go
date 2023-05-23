package main

/* Valgrind report summary

==13531== LEAK SUMMARY:
==13531==    definitely lost: 0 bytes in 0 blocks
==13531==    indirectly lost: 0 bytes in 0 blocks
==13531==      possibly lost: 1,440 bytes in 5 blocks
==13531==    still reachable: 0 bytes in 0 blocks
==13531==         suppressed: 0 bytes in 0 blocks

*/

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"os"
	"time"

	"github.com/chenhengjie123/gmf"
)

var fileCount int = 0

func main() {
	var (
		srcFileName string
		swsctx      *gmf.SwsCtx
	)

	flag.StringVar(&srcFileName, "src", "tests-sample.mp4", "source video")
	flag.Parse()

	os.MkdirAll("./tmp", 0755)

	// 基于文件创建输入上下文
	inputCtx, err := gmf.NewInputCtx(srcFileName)
	if err != nil {
		log.Fatalf("Error creating context - %s\n", err)
	}
	// 函数执行完毕后，释放上下文
	defer inputCtx.Free()

	// 获取视频流
	srcVideoStream, err := inputCtx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		log.Printf("No video stream found in '%s'\n", srcFileName)
		return
	}

	// 获取编码器
	codec, err := gmf.FindEncoder(gmf.AV_CODEC_ID_RAWVIDEO)
	if err != nil {
		log.Fatalf("%s\n", err)
	}

	// 创建编码器上下文
	codecCtx := gmf.NewCodecCtx(codec)
	defer gmf.Release(codecCtx)

	codecCtx.SetTimeBase(gmf.AVR{Num: 1, Den: 1})

	// 设置编码器上下文的参数，包括颜色格式、宽、高。
	codecCtx.SetPixFmt(gmf.AV_PIX_FMT_RGBA).SetWidth(srcVideoStream.CodecCtx().Width()).SetHeight(srcVideoStream.CodecCtx().Height())
	// 如果是实验性的编码器，设置兼容性为实验性模式
	if codec.IsExperimental() {
		codecCtx.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
	}

	// 打开编码器·
	if err := codecCtx.Open(nil); err != nil {
		log.Fatal(err)
	}
	defer codecCtx.Free()

	// 从第一帧开始获取视频流
	inputStream, err := inputCtx.GetStream(srcVideoStream.Index())
	if err != nil {
		log.Fatalf("Error getting stream - %s\n", err)
	}
	defer inputStream.Free()

	// convert source pix_fmt into AV_PIX_FMT_RGBA
	// which is set up by codec context above
	inputCodecCtx := srcVideoStream.CodecCtx()
	if swsctx, err = gmf.NewSwsCtx(inputCodecCtx.Width(), inputCodecCtx.Height(), inputCodecCtx.PixFmt(), codecCtx.Width(), codecCtx.Height(), codecCtx.PixFmt(), gmf.SWS_BICUBIC); err != nil {
		panic(err)
	}
	defer swsctx.Free()

	start := time.Now()

	var (
		pkt        *gmf.Packet
		frames     []*gmf.Frame
		drain      int = -1
		frameCount int = 0
	)

	for {
		if drain >= 0 {
			break
		}

		pkt, err = inputCtx.GetNextPacket()
		if err != nil && err != io.EOF {
			if pkt != nil {
				pkt.Free()
			}
			log.Printf("error getting next packet - %s", err)
			break
		} else if err != nil && pkt == nil {
			drain = 0
		}

		// TODO: 待看明白
		if pkt != nil && pkt.StreamIndex() != srcVideoStream.Index() {
			continue
		}

		frames, err = inputStream.CodecCtx().Decode(pkt)
		if err != nil {
			log.Printf("Fatal error during decoding - %s\n", err)
			break
		}

		// Decode() method doesn't treat EAGAIN and EOF as errors
		// it returns empty frames slice instead. Countinue until
		// input EOF or frames received.
		if len(frames) == 0 && drain < 0 {
			continue
		}

		if frames, err = gmf.DefaultRescaler(swsctx, frames); err != nil {
			panic(err)
		}

		// 对视频数据进行重编码并保存到文件
		encode(codecCtx, frames, drain)

		for i, _ := range frames {
			frames[i].Free()
			frameCount++
		}

		if pkt != nil {
			pkt.Free()
			pkt = nil
		}
	}

	for i := 0; i < inputCtx.StreamsCnt(); i++ {
		st, _ := inputCtx.GetStream(i)
		st.CodecCtx().Free()
		st.Free()
	}

	since := time.Since(start)
	log.Printf("Finished in %v, avg %.2f fps", since, float64(frameCount)/since.Seconds())
}

func encode(cc *gmf.CodecCtx, frames []*gmf.Frame, drain int) {
	packets, err := cc.Encode(frames, drain)
	if err != nil {
		log.Fatalf("Error encoding - %s\n", err)
	}
	if len(packets) == 0 {
		return
	}

	for _, p := range packets {
		width, height := cc.Width(), cc.Height()

		img := new(image.RGBA)
		img.Pix = p.Data()
		img.Stride = 4 * width
		img.Rect = image.Rect(0, 0, width, height)

		writeFile(img)

		p.Free()
	}

	return
}

func writeFile(b image.Image) {
	name := fmt.Sprintf("tmp/%d.jpg", fileCount)
	fp, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening file '%s' - %s\n", name, err)
	}
	defer fp.Close()

	fileCount++

	log.Printf("Saving file %s\n", name)

	if err = jpeg.Encode(fp, b, &jpeg.Options{Quality: 80}); err != nil {
		log.Fatal(err)
	}
}
