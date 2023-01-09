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
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

type MissingArg struct {
	Args []string
}

func (e *MissingArg) Error() string {
	return fmt.Sprintln("missing arguments:", e.Args)
}

type RestfulHandler func(w http.ResponseWriter, r *http.Request)

var handlers map[string]RestfulHandler

func init() {
	handlers = make(map[string]RestfulHandler, 0)
	handlers["/"] = handleRoot
	handlers["/draw"] = handleDraw
	handlers["/create_zone"] = handleCreateZone
	handlers["/info"] = handleInfo
	handlers["/stop"] = handleStop
	handlers["/info_zone"] = nil

	// TODO
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	prepare(w, r)
	// TODO
}

func handleDraw(w http.ResponseWriter, r *http.Request) {
	form, err := prepare(w, r, "token:s", "zone:i", "x:i", "y:i", "r:u8", "g:u8", "b:u8", "a:u8")
	if err != nil {
		returnError(w, err)
		return
	}

	zoneId := form["zone"].(int)

	zone := LookupZone(zoneId)
	if zone == nil {
		returnError(w, err)
		return
	}

	zone.ReqChan <- DrawRequest{
		Point: Point{X: form["x"].(int), Y: form["y"].(int)},
		Color: Color{R: form["r"].(uint8), G: form["g"].(uint8), B: form["b"].(uint8), A: form["a"].(uint8)},
	}
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	form, err := prepare(w, r, "token:s")
	if err != nil {
		returnError(w, err)
		return
	}

	if MasterAuth(Token(form["token"].(string))) {
		stopChan <- struct{}{}
	}
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	prepare(w, r)
	// TODO
}

func handleCreateZone(w http.ResponseWriter, r *http.Request) {
	form, err := prepare(w, r, "token:s", "x:i", "y:i", "freeze:i")
	if err != nil {
		returnError(w, err)
		return
	}

	if !MasterAuth(Token(form["token"].(string))) {
		// TODO Error
		return
	}

	// TODO
}

func prepare(w http.ResponseWriter, r *http.Request, template ...string) (map[string]interface{}, error) {
	remoteAddr := netip.MustParseAddrPort(r.RemoteAddr).String()
	stat := recordIPStatistic(remoteAddr)
	if stat.GetChallenges() > maxChallenges {
		return nil, errors.New("max challenges")
	}

	err := r.ParseForm()
	if err != nil {
		returnError(w, err)
		return nil, err
	}

	return parseForm(r.Form, template...)
}

func returnError(w http.ResponseWriter, err error) {
	w.Header().Set("error", err.Error())
	io.WriteString(w, err.Error())
}

func parseForm(form url.Values, template ...string) (map[string]interface{}, error) {
	lMap := make(map[string]string, len(template))

	for _, t := range template {
		name := strings.Split(t, ":")[0]

		if !form.Has(name) {
			return nil, &MissingArg{}
		}

		lMap[name] = form.Get(name)
	}

	return parseLiteralMap(lMap, template...)
}

func parseLiteralMap(lMap map[string]string, template ...string) (map[string]interface{}, error) {
	var (
		vMap map[string]interface{}
		err  error
	)

	vMap = make(map[string]interface{}, len(template))

	for _, t := range template {
		v := strings.Split(t, ":")
		name, typ := v[0], v[1]

		a, ok := lMap[name]
		if !ok {
			return nil, &MissingArg{}
		}

		switch typ {
		case "s":
			vMap[name] = a
		case "i":
			vMap[name], err = strconv.Atoi(a)
			if err != nil {
				return nil, err
			}
		case "u8":
			val, err := strconv.Atoi(a)
			if err != nil {
				return nil, err
			}
			vMap[name] = uint8(val)
		}
	}

	return vMap, nil
}

func MasterAuth(token Token) bool {
	return token == masterToken
}
