// Package worker provide a worker that updates a given Price data structure
// with current prices of supported currencies
// Prices are gotten from openexchangesrates API and coinmaketcap
// (Etherium price will be treated differently from the others)
package worker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/yagotome/challenge-bravo/config"
	"github.com/yagotome/challenge-bravo/currency"
)

// OpenExchangeRatesResponse ...
type OpenExchangeRatesResponse struct {
	Rates map[string]float64 `json:"rates"`
}

// CoinMarketCapResponse ...
type CoinMarketCapResponse struct {
	PriceUsd string `json:"price_usd"`
}

const ethSymbol = "ETH"

var (
	supportedCurrencies = []string{
		"USD",
		"BRL",
		"EUR",
		"BTC",
		ethSymbol,
	}
	wg = sync.WaitGroup{}
)

// Run is the function that runs worker task
func Run(p *currency.Price, c config.Config) {
	for {
		wg.Add(2)
		go treatError(updateFromOpenExchangeRates(p, &c))
		go treatError(updateFromCoinMarketCap(p, &c))
		wg.Wait()
		time.Sleep(time.Millisecond * time.Duration(c.WorkerUpdateInterval))
	}
}

func treatError(e error) {
	defer wg.Done()
	if e != nil {
		log.Println(e)
	}
}

func request(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}

func getJSONObject(url string) (map[string]interface{}, error) {
	buf, err := request(url)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	err = json.Unmarshal(buf, &data)
	return data, err
}

func getCoinOpenExchangeRates(url string) (*OpenExchangeRatesResponse, error) {
	buf, err := request(url)
	if err != nil {
		return nil, err
	}
	data := &OpenExchangeRatesResponse{}
	err = json.Unmarshal(buf, data)
	return data, err
}

// updateFromOpenExchangeRates updates all prices (expect ETH) from OpenExchangeRates.
// That API gives value of 1 USD in each supported currency
func updateFromOpenExchangeRates(p *currency.Price, conf *config.Config) error {
	url := fmt.Sprintf("%s?app_id=%s", conf.QuotesAPIURL.OpenExchangeRates, conf.APIKeys.OpenExchangeRates)
	fmt.Println("url", url)
	resp, err := getCoinOpenExchangeRates(url)
	if err != nil {
		return err
	}
	for _, c := range supportedCurrencies {
		if price, ok := resp.Rates[c]; ok {
			p.Save(c, price)
		}
	}
	return nil
}

func getCoinMarketCapResponse(url string) ([]CoinMarketCapResponse, error) {
	buf, err := request(url)
	if err != nil {
		return nil, err
	}
	var data []CoinMarketCapResponse
	err = json.Unmarshal(buf, &data)
	return data, err
}

// updateFromCoinMarketCap updates Etherium price from CoinMarketCap.
// That API gives value of 1 ETH in USD, so it needs to be inverted
func updateFromCoinMarketCap(p *currency.Price, conf *config.Config) error {
	resp, err := getCoinMarketCapResponse(conf.QuotesAPIURL.CoinMarketCap)
	if err != nil {
		return err
	}
	price, err := strconv.ParseFloat(resp[0].PriceUsd, 64)
	if err != nil {
		return err
	}
	p.Save(ethSymbol, currency.InvertPrice(price))
	return nil
}