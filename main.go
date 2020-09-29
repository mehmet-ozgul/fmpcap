package main

import (
	"flag"
	"fmt"
	"github.com/google/gopacket/pcapgo"
	"io"
	"math/rand"
	"os"
	"time"
)

var (
	lossProbability       = flag.Float64("loss", 0.02, "Loss probability of each packet")
	conseqLossProbability = flag.Float64("conseq", 0.05, "Loss probability of each *lost* packet")
	delayAtLoss           = flag.Int64("delay", 180, "Delay to introduce before sending the first packet after loss, in ms")
	delayErr              = flag.Int64("delayErr", 40, "Total Delay = DelayAtLoss +- Err/2")
	input                 = flag.String("input", "", "Input file")
	output                = flag.String("output", "", "Output file")
)

// Config has the user configuration.
type Config struct {
	LossProbability       float64
	ConseqLossProbability float64
	DelayAtLoss           int64
	DelayErr              int64
}

func saneTime(t time.Time) int64 {
	return (t.UnixNano() / 1000000) - 1600965000000
}

//nolint:gocognit
func TranslateFile(input, output string, config Config) error {
	fmt.Printf("Processing file [%v]\n", input)
	fin, err := os.Open(input)
	if err != nil {
		return err
	}
	defer func() {
		if err := fin.Close(); err != nil {
			fmt.Println("Failed to close the input put")
		}
	}()
	r, err := pcapgo.NewNgReader(fin, pcapgo.DefaultNgReaderOptions)
	if err != nil {
		return err
	}

	fout, err := os.Create(output)
	if err != nil {
		return err
	}

	w, err := pcapgo.NewNgWriter(fout, r.LinkType())
	if err != nil {
		return err
	}
	defer func() {
		if err := w.Flush(); err != nil {
			fmt.Println("Failed to flush the output file")
		}
	}()

	var packets int
	var size int
	var previousTs time.Time
	var deltaTotal time.Duration
	lost := false
	delayOffset := time.Duration(0)

	fmt.Println("pTS\tLost\tDOff\tDelta\tCom\tTS\tTS'\tpTS'")
	//nolint:gosec
	for {
		data, ci, err := r.ReadPacketData()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		packets++
		size += len(data)

		p := config.LossProbability
		if delayOffset > 0 {
			p = config.ConseqLossProbability
		}

		lost = !previousTs.IsZero() && rand.Float64() < p

		if previousTs.IsZero() {
			previousTs = ci.Timestamp
		}

		fmt.Printf("%v\t%v\t%v\t", saneTime(previousTs), lost, delayOffset)

		if !lost {
			if delayOffset > 0 {
				delta := ci.Timestamp.Sub(previousTs)
				fmt.Printf("D.%d\t", delta)
				if delta > 0 {
					compression := delta - (time.Duration(500+rand.Intn(1000)) * time.Microsecond)
					fmt.Printf("c.%v\t", compression)
					delayOffset -= compression
				} else {
					fmt.Printf("%v\t", 0)
				}
			} else {
				fmt.Printf("*%d\t%v\t", 0, 0)
				delayOffset = 0
			}
			fmt.Printf("%v\t", saneTime(ci.Timestamp))
			previousTs = ci.Timestamp
			ci.Timestamp = ci.Timestamp.Add(delayOffset)
			fmt.Printf("%v\t", saneTime(ci.Timestamp))
			if err := w.WritePacket(ci, data); err != nil {
				return err
			}
		} else {
			dv := config.DelayAtLoss - config.DelayErr/2 + (rand.Int63() % config.DelayErr)
			delayOffset += time.Duration(dv) * time.Millisecond
			fmt.Printf("%v\t%v\t%v\t%v\t", "LOST", 0, 0, 0)
		}

		fmt.Printf("%v\n", saneTime(previousTs))
	}
	sec := int(deltaTotal.Seconds())
	if sec == 0 {
		sec = 1
	}
	fmt.Printf("Avg packet rate %d/s\n", packets/sec)
	fmt.Printf("Packets: %v, Size: %v\n", packets, size)

	return nil
}

func _main() int {
	flag.Parse()
	config := Config{
		LossProbability:       *lossProbability,
		ConseqLossProbability: *conseqLossProbability,
		DelayAtLoss:           *delayAtLoss,
		DelayErr:              *delayErr,
	}
	if len(*input) == 0 || len(*output) == 0 || *input == *output {
		fmt.Printf("Improper input/output [%v]/[%v]\n", *input, *output)
		flag.PrintDefaults()
		return 2
	}

	fmt.Printf("Config: %#v\nin: %s\nout: %s\n", config, *input, *output)

	if err := TranslateFile(*input, *output, config); err != nil {
		fmt.Printf("Unable to translate file %v: %v", *input, err)
		return 2
	}

	return 0
}

func main() {
	os.Exit(_main())
}
