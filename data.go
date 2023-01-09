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
	"errors"
	"image"
	"image/color"
	"sync"
	"sync/atomic"
)

type TokenStatistic struct {
	Lock              sync.Mutex
	NextOperationTime Tick
}

func (stat *TokenStatistic) Init() {
	stat.NextOperationTime = Now()
}

type Color struct {
	R, G, B, A uint8
}

func (c Color) RGBA() *color.RGBA {
	return &color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

type Point image.Point

func (d1 Point) Greater(d Point) bool {
	if d1.X > d.X || d1.Y > d.Y {
		return true
	}
	return false
}

type Zone struct {
	closed bool

	Size, Range Point
	Map         [][]Color
	Title       string

	TokenStats      map[Token]*TokenStatistic
	TokenStatsLock  sync.RWMutex
	PrivilegedToken Token
	ReqChan         chan DrawRequest
	//IllegalOpChan   chan Token
	//BroadcastChan   chan string
	Freeze Tick
}

var (
	zones     []*Zone
	zonesLock sync.RWMutex
)

func LookupZone(zoneId int) *Zone {
	zonesLock.RLock()
	defer zonesLock.RUnlock()

	if zoneId < len(zones) || zoneId >= 0 {
		return zones[zoneId]
	}

	return nil
}

func (z *Zone) AddToken(token Token) {
	stat := new(TokenStatistic)
	stat.Init()

	z.TokenStatsLock.Lock()
	defer z.TokenStatsLock.Unlock()

	z.TokenStats[token] = stat
}

func (z *Zone) DeleteToken(token Token) {
	z.TokenStatsLock.Lock()
	defer z.TokenStatsLock.Unlock()

	delete(z.TokenStats, token)
}

func (z *Zone) UpdatePrivilegedToken(t Token) {
	z.TokenStatsLock.Lock()
	defer z.TokenStatsLock.Unlock()

	z.PrivilegedToken = t
}

func (z *Zone) Init(size Point) error {
	if size.X <= 0 || size.Y <= 0 {
		return errors.New("zone size cannot be zero")
	}

	z.Map = make([][]Color, size.X)
	for i := range z.Map {
		z.Map[i] = make([]Color, size.Y)
	}

	z.Size = size
	z.Range.X = size.X - 1
	z.Range.Y = size.Y - 1

	z.closed = false

	z.TokenStats = make(map[Token]*TokenStatistic, 10)
	//z.IllegalOpChan = make(chan Token, 10)
	z.ReqChan = make(chan DrawRequest, 1000)

	return nil
}

func (z *Zone) Close() {
	z.closed = true
	//close(z.IllegalOpChan)
	close(z.ReqChan)
}

func (z *Zone) GenImage() *image.RGBA {
	img := image.NewRGBA(image.Rectangle{Min: image.Point{}, Max: image.Point{X: z.Size.X, Y: z.Size.Y}})

	for x := 0; x <= z.Range.X; x++ {
		for y := 0; y <= z.Range.Y; y++ {
			img.Set(x, y, z.Map[x][y].RGBA())
		}
	}
	return img
}

func (z *Zone) LoadImage(img image.Image) error {
	if img.ColorModel() != color.RGBAModel {
		return errors.New("RGBA required")
	}

	size := Point(img.Bounds().Size())
	if size.Greater(z.Size) {
		return errors.New("too large")
	}

	for x := 0; x <= z.Range.X; x++ {
		for y := 0; y <= z.Range.Y; y++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			z.Map[x][y] = Color{uint8(r), uint8(g), uint8(b), uint8(a)}
		}
	}
	return nil
}

type IPStatistic struct {
	Lock       sync.RWMutex
	Challenges int32
}

func (s *IPStatistic) Init() {
	s.Challenges = maxChallenges
}

func (s *IPStatistic) GetChallenges() int32 {
	return atomic.LoadInt32(&s.Challenges)
}

func (s *IPStatistic) DecreaseChallenges() {
	atomic.AddInt32(&s.Challenges, -1)
}

var (
	ipStatistics     map[string]*IPStatistic
	ipStatisticsLock sync.RWMutex
)

func lookupIPStatistic(remoteAddr string) (*IPStatistic, bool) {
	ipStatisticsLock.RLock()
	stat, ok := ipStatistics[remoteAddr]
	ipStatisticsLock.RUnlock()

	return stat, ok
}

func addIPStatistic(remoteAddr string) *IPStatistic {
	stat := new(IPStatistic) // No competition.
	stat.Init()

	ipStatisticsLock.Lock()
	ipStatistics[remoteAddr] = stat // Pointer modification competition may cause lacks of allocation, but GC will solve it.
	ipStatisticsLock.Unlock()

	return stat
}

func recordIPStatistic(remoteAddr string) *IPStatistic {
	stat, ok := lookupIPStatistic(remoteAddr)
	if ok {
		return stat
	}

	return addIPStatistic(remoteAddr)
}
