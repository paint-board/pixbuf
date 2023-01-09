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
	"sync/atomic"
	"time"
)

type Tick uint64

var (
	sysTick Tick
)

func Now() Tick {
	return Tick(atomic.LoadUint64((*uint64)(&sysTick)))
}

func (t Tick) Add(u Tick) Tick {
	return t + u
}

func (t Tick) Before(u Tick) bool {
	if t < u {
		return true
	}
	return false
}

func (t Tick) After(u Tick) bool {
	if t > u {
		return true
	}
	return false
}

func doTick(timestamp time.Duration) {
	c := time.Tick(timestamp)
	for {
		select {
		case <-c:
			atomic.AddUint64((*uint64)(&sysTick), 1)
		}
	}
}
