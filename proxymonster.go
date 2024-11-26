package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var asciiArt = `
 ||  ||  
 \\()// 
//(__)\\
||    || 
ProxyMonster - by iseeu
`

func main() {
	cyan := color.New(color.FgHiCyan).SprintFunc()
	yellow := color.New(color.FgHiYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Println(green(asciiArt))

	fmt.Println(yellow("seçmek istediğiniz proxy türünü girin:"))
	fmt.Println(cyan("[1] http\n[2] https\n[3] socks4\n[4] socks5"))
	var choice int
	fmt.Print("seçiminiz: ")
	fmt.Scanln(&choice)

	var proxyType string
	switch choice {
	case 1:
		proxyType = "http"
	case 2:
		proxyType = "https"
	case 3:
		proxyType = "socks4"
	case 4:
		proxyType = "socks5"
	default:
		color.Red("geçersiz seçim!")
		return
	}

	fmt.Print("proxy listesinin dosya adını girin (örnek: proxies.txt): ")
	var fileName string
	fmt.Scanln(&fileName)

	proxies, err := loadProxies(fileName)
	if err != nil {
		color.Red("hata: proxy listesi yüklenemedi: %v", err)
		return
	}

	color.Cyan("\nproxyler kontrol ediliyor...\n")
	results := checkProxiesConcurrent(proxies, proxyType)

	color.Green("\n\ntoplam proxy: %d | çalışan proxy: %d\n", len(proxies), len(results))
	color.Yellow("çalışan proxyler:")
	for _, proxy := range results {
		fmt.Println(proxy)
	}
	fmt.Println("\ndevam etmek için enter tuşuna bas.")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func loadProxies(fileName string) ([]string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// satır satır proxyleri oku
	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			proxies = append(proxies, line)
		}
	}

	return proxies, scanner.Err()
}

func checkProxiesConcurrent(proxies []string, proxyType string) []string {
	var workingProxies []string
	var wg sync.WaitGroup
	var mu sync.Mutex

	// dinamik sonuçlar için kanal oluştur
	results := make(chan string)

	// her proxy için bir goroutine başlat
	for i, proxy := range proxies {
		wg.Add(1)
		go func(i int, proxy string) {
			defer wg.Done()
			if checkProxy(proxy, proxyType) {
				mu.Lock()
				workingProxies = append(workingProxies, proxy)
				mu.Unlock()
				results <- fmt.Sprintf("[%d/%d] %s - %s", i+1, len(proxies), proxy, color.GreenString("ok"))
			} else {
				results <- fmt.Sprintf("[%d/%d] %s - %s", i+1, len(proxies), proxy, color.RedString("fail"))
			}
		}(i, proxy)
	}

	// tüm goroutine'ler tamamlanınca kanalı kapatıyoruz
	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		fmt.Println(result)
	}

	return workingProxies
}

// proxyyi kontrol eden temel fonksiyon
func checkProxy(proxy, proxyType string) bool {
	switch proxyType {
	case "http", "https":
		client := &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(&url.URL{
					Scheme: proxyType,
					Host:   proxy,
				}),
			},
		}
		resp, err := client.Get("http://example.com")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == 200
	case "socks4", "socks5":
		conn, err := net.DialTimeout("tcp", proxy, 5*time.Second)
		if err != nil {
			return false
		}
		defer conn.Close()
		return true
	default:
		return false
	}
}
