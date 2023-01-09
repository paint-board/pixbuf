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
	"encoding/json"
	ws "golang.org/x/net/websocket"
	"net/netip"
	"net/url"
	"sync/atomic"
)

func serveWS(conn *ws.Conn) {
	var (
		err error
	)

	defer conn.Close()

	r := json.NewDecoder(conn)
	w := json.NewEncoder(conn)

	remoteAddr := netip.MustParseAddrPort(conn.RemoteAddr().String()).Addr().String()

	stat, ok := lookupIPStatistic(remoteAddr)
	if !ok {
		stat = addIPStatistic(remoteAddr)
	} else {
		if atomic.LoadInt32(&stat.Challenges) <= 0 {
			return
		}
	}

	for {
		var (
			req  url.Values
			resp url.Values
		)

		err = r.Decode(&req)
		if err != nil {
			return
		}

		if !req.Has("op") {
			resp.Set("error", "missing $op")
			return
		}

		switch req.Get("op") {
		case "draw":
		case "admin":
		case "":
		}

		err = w.Encode(resp)
		if err != nil {
			return
		}
	}
}
