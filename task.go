/*
   Copyright (C) 2023  Holiday Paintboard Authors

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published
   by the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	_ "embed"
	"flag"
	"fmt"
	ws "golang.org/x/net/websocket"
	"image/png"
	"log"
	"math"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

var isDebug bool

func fatal(a ...interface{}) {
	_, _ = fmt.Fprintln(os.Stderr, a...)
	os.Exit(1)
}

var (
	memoryLimit   uint64
	maxChallenges int32
	masterToken   Token
	masterLock    sync.Mutex
	wg            sync.WaitGroup
	stopChan      chan struct{}
	tokenGen      RandTokenGen
)

//go:embed LICENSE
var LICENSE string

func main() {
	var (
		workDir, keyPath, certPath string
		restfulPort, wsPort        string
		tickDuration               int
		err                        error
	)

	fmt.Println("Copyright (C) 2023 Holiday Paintboard Authors @github.com/paint-board\nThis program is licensed under GNU AGPLv3. Use option -L to view.")

	showLicense := flag.Bool("L", false, "Print license and exit.")
	flag.BoolVar(&isDebug, "D", false, "Enable debug mode.")
	flag.StringVar(&workDir, "W", "", "Declare working directory.")

	maxChallengesP := flag.Int("c", 3, "Declare max challenges.")
	flag.IntVar(&tickDuration, "d", 1000, "Declare tick duration (millisecond) .")
	flag.Uint64Var(&memoryLimit, "m", 1024, "Declare memory limit (mebibyte) .")
	flag.StringVar(&restfulPort, "p", "19198", "Declare RESTful API service port.")
	flag.StringVar(&wsPort, "T", "19199", "Declare WebSocket API service port.")
	certKeyPair := flag.String("k", ":", "Certificate:key certKeyPair.")
	flag.Parse()

	if *showLicense {
		fmt.Println(LICENSE)
		return
	}

	if *maxChallengesP > math.MaxInt32 || *maxChallengesP < math.MinInt32 {
		fmt.Println("Max challenges: illegal value.")
		return
	}

	if workDir == "" {
		workDir, err = os.MkdirTemp("", "paint-board-")
		if err != nil {
			fatal(err)
		}
	} else {
		err = os.Mkdir(workDir, 0750)
		if err != nil {
			if os.IsExist(err) {
				err := os.RemoveAll(workDir)
				if err != nil {
					fatal(err)
				}
			} else {
				fatal(err)
			}
		}
	}

	split := strings.Split(*certKeyPair, ":")
	if len(split) != 2 {
		flag.Usage()
		return
	}
	certPath, keyPath = split[0], split[1]

	err = os.Chdir(workDir)
	if err != nil {
		fatal(err)
	}

	fmt.Println()

	fmt.Println("Debug mode:", isDebug)
	fmt.Println("Working directory:", workDir)
	fmt.Println("Memory limit:", memoryLimit, "MiB")
	fmt.Println("Tick duration:", tickDuration, "ms")
	fmt.Println("Max challenges:", *maxChallengesP)
	fmt.Print("Port: http/", restfulPort, " ws/", wsPort)
	fmt.Println()

	maxChallenges = int32(*maxChallengesP) // TODO Unsafe!

	memoryLimit *= 1024 * 1024

	restfulPort = ":" + restfulPort
	wsPort = ":" + wsPort

	// Initialize.

	zones = make([]*Zone, 0)
	ipStatistics = make(map[string]*IPStatistic, 100)
	stopChan = make(chan struct{}, 1)

	// Start global ticker.
	go doTick((time.Duration)(tickDuration) * time.Millisecond)

	masterToken = tokenGen.Generate()
	if err != nil {
		log.Fatalln(err)
	}
	if isDebug {
		masterToken = "M"
	}

	log.Println("Master token generated:", masterToken)

	go func() {
		var m runtime.MemStats
		for {
			time.Sleep(1 * time.Second)
			runtime.ReadMemStats(&m)
			if m.TotalAlloc > memoryLimit {
				log.Println("Out of memory limit: total", m.TotalAlloc, "bytes")
				stopChan <- struct{}{}
			}
		}
	}()

	// RESTful service
	go func() {
		var (
			err error
		)

		for pattern, handleFunc := range handlers {
			http.HandleFunc(pattern, handleFunc)
		}
		if *certKeyPair == ":" {
			log.Println("Listening (insecure):", restfulPort)
			err = http.ListenAndServe(restfulPort, nil)
		} else {
			log.Println("Listening (TLS):", restfulPort)
			err = http.ListenAndServeTLS(restfulPort, certPath, keyPath, nil)
		}
		if err != nil {
			log.Fatalln("RESTful API service failure:", err)
		}
	}()

	// WebSocket service
	go func() {
		var (
			err error
		)
		if *certKeyPair == ":" {
			err = http.ListenAndServe(wsPort, ws.Handler(serveWS))
		} else {
			err = http.ListenAndServeTLS(wsPort, certPath, keyPath, ws.Handler(serveWS))
		}
		if err != nil {
			log.Fatalln("WebSocket API service failure:", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-stopChan:
		}
		// TODO Notify.
	}()

	wg.Wait()

	for _, z := range zones {
		z.Close()
	}

	log.Println("Exporting zones.")

	for i, z := range zones {
		err = z.export()

		if err != nil {
			log.Println("Error occurred while exporting zone", i, err)
		}
	}

	log.Println("Exiting.")
}

func (z *Zone) export() error {
	fileImage, err := os.Create(string(z.PrivilegedToken + ".png"))
	if err != nil {
		return err
	}

	err = png.Encode(fileImage, z.GenImage())
	if err != nil {
		return err
	}

	fileToken, err := os.Create(string(z.PrivilegedToken + ".tok"))
	if err != nil {
		return err
	}

	for t := range z.TokenStats {
		_, err = fmt.Fprintln(fileToken, t)
		if err != nil {
			return err
		}
	}
	return nil
}

type DrawRequest struct {
	Point Point
	Color Color
	Token Token
}

func (z *Zone) doDrawPoints() {
	for {
		if z.closed == true {
			return
		}
		select {
		case r := <-z.ReqChan:
			if r.Point.Greater(z.Range) {
				continue
			}
			z.Map[r.Point.X][r.Point.Y] = r.Color
		}
	}
}

func createZone(size Point, freezeTime Tick) (*Zone, error) {
	zone := new(Zone)
	err := zone.Init(size)
	if err != nil {
		return nil, err
	}

	zone.Freeze = freezeTime

	zonesLock.Lock()
	defer zonesLock.Unlock()
	zones = append(zones, zone)
	zoneId := len(zones) - 1

	t := tokenGen.Generate()
	if isDebug {
		t = "A"
	}

	zone.UpdatePrivilegedToken(t)
	log.Println("Created", size.X, "X", size.Y, "Y zone", zoneId, "with privileged token:", zone.PrivilegedToken)

	go zone.doDrawPoints()

	return zone, nil
}
