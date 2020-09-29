# FMPCAP
A simple tool, designed to be used inconjunction with sipp, it introduces randomized packet loss and delay to rtp streams. 
`fmpcap` works on pcapng files and randomly removes packets, optionally adding packets delay after packet loss.

```text
Usage of fmpcap
  -conseq float
    	Loss probability of each *lost* packet (default 0.25)
  -delay int
    	Delay to introduce before sending the first packet after loss, in ms (default 200)
  -delayErr int
    	Total Delay = DelayAtLoss +- Err/2 (default 40)
  -input string
    	Input file
  -loss float
    	Loss probability of each packet (default 0.02)
  -output string
    	Output file
```

## Shortcomings

- gopacket does not support writing to `pcap` files (Or I haven't found a way to write). 
The output is `pcapng`, use wireshark (or tshark etc.) to convert output to `pcap`. 
`sipp` does not play `pcapng`.
 