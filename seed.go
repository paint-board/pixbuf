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
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
)

type Token string

func (t1 Token) Equal(t Token) bool {
	return t1 == t
}

const RandTokenGenBufSize = 128

type RandTokenGen struct{}

func (_ RandTokenGen) Generate() Token {
	buf := make([]byte, RandTokenGenBufSize)

	_, _ = rand.Read(buf)

	sha := sha1.Sum(buf)

	return Token(hex.EncodeToString(sha[0 : len(sha)-1]))
}
