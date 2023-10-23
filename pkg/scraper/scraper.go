package scraper

import (
	"fmt"
	"log"
	"time"

	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

type Scraper struct {
	DriverPath   string
	Url          string
	DriverOpts   []string
	Timeout      <-chan time.Time
	Isrunning    bool
	Service      *selenium.Service
	Capabilities selenium.Capabilities
}

func NewScraper(driverpath, url string, opts []string, timeoutsec int64) *Scraper {
	service, err := selenium.NewChromeDriverService(driverpath, 4444)
	if err != nil {
		log.Fatal("Error:", err)
	}
	caps := selenium.Capabilities{}
	caps.AddChrome(chrome.Capabilities{Args: opts})

	return &Scraper{
		DriverPath:   driverpath,
		Url:          url,
		DriverOpts:   opts,
		Timeout:      time.After(time.Second * time.Duration(timeoutsec)),
		Isrunning:    false,
		Service:      service,
		Capabilities: caps,
	}
}

func (sc *Scraper) Get() string {
	if !sc.Isrunning {
		sc.Service, _ = selenium.NewChromeDriverService(sc.DriverPath, 4444)
		fmt.Println(sc.Service)
		sc.Capabilities = selenium.Capabilities{}
		sc.Capabilities.AddChrome(chrome.Capabilities{Args: sc.DriverOpts})
		sc.Isrunning = true
		go func() {
			// <-sc.Timeout
			time.Sleep(time.Second * 1800)
			defer sc.Service.Stop()
			sc.Isrunning = false
		}()
	}
	driver, err := selenium.NewRemote(sc.Capabilities, "")
	if err != nil {
		log.Fatal("Error:", err)
	}
	err = driver.Get("https://scrapingclub.com/exercise/list_infinite_scroll/")
	if err != nil {
		log.Fatal("Error:", err)
	}

	return ""
}
