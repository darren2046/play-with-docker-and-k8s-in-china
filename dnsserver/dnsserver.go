package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type handler struct{}

var blacklist []string
var upstreamDNS *string
var ipAddress *string
var domainListPath *string

func matchDomain(domain string, blacklist []string) bool {
	for _, b := range blacklist {
		if strings.Contains(domain, b) {
			return true
		}
	}
	return false
}

func readDomains(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var domains []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		domains = append(domains, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return domains, nil
}

// udpSendReceive 发送数据到指定的UDP端口，并接收响应
func udpSendReceive(remoteAddr string, data []byte) ([]byte, error) {
	// 解析远端地址
	rAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return nil, err
	}

	// 建立UDP连接
	conn, err := net.DialUDP("udp", nil, rAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 设置超时时间
	deadline := time.Now().Add(5 * time.Second)
	conn.SetDeadline(deadline)

	// 发送数据
	_, err = conn.Write(data)
	if err != nil {
		return nil, err
	}

	// 接收响应
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, err
	}

	return buffer[:n], nil
}

func (this *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)

	log.Printf("%d ==> %s", r.Question[0].Qtype, msg.Question[0].Name)

	if r.Question[0].Qtype == dns.TypeA {
		if matchDomain(msg.Question[0].Name, blacklist) {
			msg.Authoritative = true
			domain := msg.Question[0].Name
			var rttl uint32 = 60

			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: rttl},
				A:   net.ParseIP(*ipAddress),
			})
		} else {
			log.Println("IP not found, query from upstream")
			m, err := r.Pack()
			if err != nil {
				log.Println("Error when pack dns request")
			}

			response, err := udpSendReceive(*upstreamDNS+":53", m)
			if err == nil {
				w.Write(response)
			} else {
				log.Printf("Error while query to upstream dns server: %v", err)
			}
		}

	}

	w.WriteMsg(&msg)
}

func main() {
	upstreamDNS = flag.String("dns", "8.8.8.8", "上游DNS服务器地址")
	ipAddress = flag.String("ip", "127.0.0.1", "需要返回的IP地址")
	domainListPath = flag.String("file", "", "域名列表文件的路径")

	flag.Parse()

	if *domainListPath == "" {
		fmt.Println("请提供域名列表文件的路径")
		os.Exit(1)
	}

	var err error
	blacklist, err = readDomains(*domainListPath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	srv := &dns.Server{Addr: ":53", Net: "udp"}
	srv.Handler = &handler{}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}
