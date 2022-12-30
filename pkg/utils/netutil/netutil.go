/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package netutil

import (
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"
)

func IsValidPort(port int) bool {
	return port > 0 && port < 65535
}

func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func GetRequestIP(req *http.Request) string {
	address := strings.Trim(req.Header.Get("X-Real-Ip"), " ")
	if address != "" {
		return address
	}

	address = strings.Trim(req.Header.Get("X-Forwarded-For"), " ")
	if address != "" {
		return address
	}

	address, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}

	return address
}

// InetAtoN convert str ip to int
// input: 192.168.1.1 output 3232235777
func InetAtoN(ip string) int64 {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To4())
	return ret.Int64()
}

// InetNtoA convert int ip to str
// input: 3232235777 output 192.168.1.1
func InetNtoA(ip int64) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

func Reachable(protocol string, addr string, timeout time.Duration) error {
	connection, err := net.DialTimeout(protocol, addr, timeout)
	if err == nil {
		connection.Close()
	}
	return err
}
