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

package dnscontroller

import (
	"bytes"
	"sort"
	"strings"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	tmplutil "github.com/kubeclipper/kubeclipper/pkg/utils/template"
)

const corednsCorefile = `.:53 {
	errors
	health {
		lameduck 5s
	}
	ready
	kubernetes {{ .DNSDomain }} in-addr.arpa ip6.arpa {
	   pods insecure
	   fallthrough in-addr.arpa ip6.arpa
	   ttl 30
	}
	prometheus :9153
	forward . /etc/resolv.conf
	cache 10
	loop
	reload 10s
	loadbalance
}
{{- range $domain := .Domains}}
    {{- $n := len $domain.A}}
    {{- if gt $n 0}}
{{/* hosts use 53 port with default*/}}
{{$domain.Name}}:53 {
    errors
    cache 10
	loadbalance
    {{- $n := len $domain.A}}
    {{- if gt $n 0}}
    hosts {
        {{- range $record := $domain.A}}
        {{$record.IP}} {{$record.RR}}
        {{- end}}
        fallthrough
    }
	{{- /* forward to 5300 resolved by template */}}
	forward . {{ $.CoreDNSVIP }}:5300
    {{- end}}
}
{{- end}}
{{- $n := len $domain.Extensive}}
    {{- if gt $n 0}}
{{$domain.Name}}:5300 {
    errors
    cache 10
	loadbalance
 {{- range $record := $domain.Extensive}}
    template IN A {{$record.Name}} {
        match .{{ $record.RR | replace "." "\\." }}
        {{- range $RIP := $record.IPs}}
        answer "{{ "e3sgLk5hbWUgfX0=" | b64dec }} 60 IN A {{$RIP}}"
        {{- end}}
        fallthrough
    }
    {{- end}}
}
    {{- end}}
{{- end}}
`

type DNS struct {
	Domains    []Domain `json:"domains"`
	DNSDomain  string   `json:"dnsDomain"`
	CoreDNSVIP string   `json:"coreDNSVIP"` // coredns service ip
}

type Domain struct {
	Name      string            `json:"name"`
	A         []Record          `json:"a"`
	Extensive []ExtensiveRecord `json:"extensive"`
}

type ExtensiveRecord struct {
	ExRecord `json:",inline"`
	Name     string `json:"name"`
}

type Record struct {
	RR string `json:"rr"`
	IP string `json:"ip"`
}

type ExRecord struct {
	RR  string   `json:"rr"`
	IPs []string `json:"ip"`
}

type exList []ExtensiveRecord

func (l exList) Len() int      { return len(l) }
func (l exList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l exList) Less(i, j int) bool {
	return len(l[i].RR) > len(l[j].RR)
}

func renderCorefile(data []*v1.Domain, vip, dnsDomain string) (string, error) {
	dns := transformer(data)
	dns.DNSDomain = dnsDomain
	dns.CoreDNSVIP = vip
	buffer := bytes.NewBuffer(nil)
	at := tmplutil.New()
	_, err := at.RenderTo(buffer, corednsCorefile, dns)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func transformer(data []*v1.Domain) DNS {
	var (
		domainList = make([]Domain, 0, len(data))
	)
	for _, domain := range data {
		a := make([]Record, 0)
		ex := make([]ExtensiveRecord, 0)
		for _, record := range domain.Spec.Records {
			key := record.RR + "." + record.Domain
			if record.RR == "@" { // @ is special record
				key = record.Domain
			}
			if isGenericRecord(record.RR) {
				item := ExtensiveRecord{
					ExRecord: ExRecord{
						RR: key,
					},
					Name: record.Domain,
				}
				for _, parseRecord := range record.ParseRecord {
					item.IPs = append(item.IPs, parseRecord.IP)
				}
				ex = append(ex, item)
			} else {
				for _, parseRecord := range record.ParseRecord {
					a = append(a, Record{
						RR: key,
						IP: parseRecord.IP,
					})
				}
			}
		}
		sort.Sort(exList(ex)) // sort by length of RR to change record priority
		d := Domain{
			Name:      domain.Name,
			A:         a,
			Extensive: ex,
		}
		domainList = append(domainList, d)
	}
	return DNS{
		Domains: domainList,
	}
}

// 判断是否为泛解析 *.x 或者 * 格式
func isGenericRecord(rr string) bool {
	if rr == "*" {
		return true
	}
	split := strings.Split(rr, ".")
	return len(split) > 1 && split[0] == "*"
}
