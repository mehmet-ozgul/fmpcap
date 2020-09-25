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
	conseqLossProbability = flag.Float64("conseq", 0.25, "Loss probability of each *lost* packet")
	delayAtLoss           = flag.Int("delay", 200, "Delay to introduce before sending the first packet after loss, in ms")
	delayVar              = flag.Float64("delayVar", 0.1, "Max variance in delay at loss, proportinal to the delay value.")
	input                 = flag.String("input", "", "Input file")
	output                = flag.String("output", "", "Output file")
)

// Config has the user configuration.
type Config struct {
	LossProbability       float64
	ConseqLossProbability float64
	DelayAtLoss           int
	DelayVar              float64
}

func TranslateFile(input, output string, config Config) error {
	fmt.Printf("Processing file [%v]\n", input)
	fin, err := os.Open(input)
	if err != nil {
		return err
	}
	defer fin.Close()
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
	defer w.Flush()

	var packets int
	var size int
	var previousTs time.Time
	var deltaTotal time.Duration
	lost := false
	delayOffset := time.Duration(0)

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
		if lost {
			p = config.ConseqLossProbability
		}

		addDelay := false
		if !previousTs.IsZero() && rand.Float64() < p {
			lost = true
		} else {
			if lost {
				addDelay = true
			}
			lost = false
		}

		if !previousTs.IsZero() {
			deltaTotal += ci.Timestamp.Sub(previousTs)
		}
		previousTs = ci.Timestamp

		if !lost {
			if addDelay {
				delayOffset += time.Duration(config.DelayAtLoss) * time.Millisecond
			}
			ci.Timestamp = ci.Timestamp.Add(delayOffset)
			if err := w.WritePacket(ci, data); err != nil {
				return err
			}
		}
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
		DelayVar:              *delayVar,
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
