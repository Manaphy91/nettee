package main

import (
	"fmt"
	"os"
	"strings"
	"strconv"
	"net"
	"github.com/Manaphy91/nettee/logger"
	"github.com/Manaphy91/nettee/logger/sink"
	"github.com/TreyBastian/colourize"
)


func main(){

	// start to parse command line arguments 
	if (len(os.Args) < 2) || (os.Args[1] == "-h") || (os.Args[1] == "--help") {
		fmt.Printf("usage: %s [options] --sink=(udp|tcp):<address>:<port>...\n", os.Args[0])
		fmt.Println("\n  options:\n")
		fmt.Println("    -h, --help                   exactly what you are reading now")
		fmt.Println("    --source=<filename>          watch a file like `less +F <filename>' do")
		fmt.Println("    --no-output                  do not print log entries to stdin")
		fmt.Println("    --show-date                  attach datetime to every log entry")
		os.Exit(0)
	} else {
		var source *string = new(string)
		var output bool = true
		var showDate bool = false
		i := 1
		for ; i < len(os.Args); i++ {
			if strings.HasPrefix(os.Args[i], "--source=") {
				*source = os.Args[i][len("--source="):]
				*source = strings.TrimSpace(*source)
				if _, err := os.Stat(*source); err != nil {
					fmt.Fprintf(os.Stderr, colourize.Colourize(fmt.Sprintf("Error reading source file %s stats: %s\n", *source, err), colourize.Red))
					os.Exit(1)
				}
			} else if strings.HasPrefix(os.Args[i], "--no-output") {
				output = false
			} else if strings.HasPrefix(os.Args[i], "--show-date") {
				showDate = true
			} else {
				break
			}
		}
		if len(*source) == 0 {
			source = nil
		}

		// logger object creation 
		log, err := logger.New(source)
		if err != nil {
			fmt.Fprintln(os.Stderr, colourize.Colourize(err))
		}
		if !output {
			log.SetPrintLogEntry(false)
		}

		// creation and set up of sinks
		var sinkSlice []sink.Sink
		for ; i < len(os.Args); i++ {
			if strings.HasPrefix(os.Args[i], "--sink=") {
				var protocol string
				var address string
				var port uint16
				if strings.HasPrefix(os.Args[i][len("--sink="):], "udp") {
					protocol = "udp"
				} else if strings.HasPrefix(os.Args[i][len("--sink="):], "tcp") {
					protocol = "tcp"
				} else {
					fmt.Fprintf(os.Stderr, colourize.Colourize(fmt.Sprintf("Error unknown protocol in sink param %s\n",
						os.Args[i][len("--sink="):]), colourize.Red))
					os.Exit(1)
				}
				startOffset := strings.Index(os.Args[i], ":")
				if startOffset == -1 {
					fmt.Fprintf(os.Stderr, colourize.Colourize(fmt.Sprintf("Error wrong syntax used to specify address in sink param %s\n",
						os.Args[i][len("--sink="):]), colourize.Red))
					os.Exit(1)
				}
				endOffset := strings.Index(os.Args[i][startOffset + 1:], ":")
				if endOffset == -1 {
					fmt.Fprintf(os.Stderr, colourize.Colourize(fmt.Sprintf("Error wrong syntax used to specify address in sink param %s\n",
						os.Args[i][len("--sink="):]), colourize.Red))
					os.Exit(1)
				}
				address = strings.TrimSpace(os.Args[i][startOffset + 1:startOffset + 1 + endOffset])
				if _, err := net.LookupIP(address); err != nil {
					fmt.Fprintf(os.Stderr, colourize.Colourize(fmt.Sprintf("Error during lookup of address in sink param %s: %s\n",
						os.Args[i][len("--sink="):], err), colourize.Red))
					os.Exit(1)
				}
				tmp, err := strconv.Atoi(strings.TrimSpace(os.Args[i][startOffset + 2 + endOffset:]))
				if err != nil || tmp <= 0 {
					fmt.Fprintf(os.Stderr, colourize.Colourize(fmt.Sprintf("Error wrong port number in sink param %s\n",
						os.Args[i][len("--sink="):]), colourize.Red))
					os.Exit(1)
				}
				port = uint16(tmp)
				var tmpSink sink.Sink
				if protocol == "tcp" {
					tmpSink = sink.NewRawSocketSink(address, port, true)
				} else if protocol == "udp" {
					tmpSink = sink.NewRawSocketSink(address, port, false)
				}
				tmpSink.(*sink.RawSocketSink).SetShowDate(showDate)
				log.AddSink(tmpSink)
				sinkSlice = append(sinkSlice, tmpSink)
			}
		}
		if len(sinkSlice) == 0 {
			fmt.Fprintf(os.Stderr, colourize.Colourize("Error no one sink specified\n", colourize.Red))
			os.Exit(1)
		}

		log.Start()
		// wait until everything was piped through one or multiple sinks, in other words
		// when an EOF was inserted in STDIN or the user kill the process
		<-log.StopChan()
	}

}
