package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/ryho/go-robinhood"
)

const (
	HMNY = "HMNY"
)

func main() {
	robinhood.DebugMode = true
	err := sliceSalami(HMNY, 1000, false)
	if err != nil {
		fmt.Println(err)
	} /*
		err = getQuote(HMNY)
		if err != nil {
			fmt.Println(err)
		}
		err = getQuote("S")
		if err != nil {
			fmt.Println(err)
		}
		err = getQuote("SQ")
		if err != nil {
			fmt.Println(err)
		}*/
}

// fetch all stocks and sort by price
func findPennyStocks() error {
	return nil
}

func getClient() (*robinhood.Client, error) {
	myCreds := robinhood.NewCredsWithMFA("theryanhollis@gmail.com", "asdf", "123")
	catcher := &robinhood.CredsCacher{
		Creds: myCreds,
		Path:  "creds.txt",
	}
	client, err := robinhood.Dial(catcher)
	if err != nil {
		return nil, fmt.Errorf("error loging in: %v", err)
	}
	return client, nil
}

func getClientAndAccount() (*robinhood.Client, *robinhood.Account, error) {
	client, err := getClient()
	if err != nil {
		return nil, nil, err
	}
	accounts, err := client.GetAccounts()
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching accounts: %v", err)
	}
	if len(accounts) != 1 {
		return nil, nil, fmt.Errorf("expected 1 account, got %v", len(accounts))
	}
	return client, &accounts[0], nil
}

// Fetches all orders for this symbol and cancels ones that are queued
func cancelAllOpenOrders(symbol string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	instrument, err := client.GetInstrumentForSymbol(symbol)
	if err != nil {
		return fmt.Errorf("error getting instrument: %v", err)
	}
	orders, err := client.GetRecentOrders(instrument)
	if err != nil {
		return err
	}
	fmt.Printf("Orders %v\n", PrettyPrint(orders))
	for _, order := range orders {
		if order.State == robinhood.OrderState_Queued {
			err = client.CancelOrder(order.Id)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getQuote(symbol string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	instrument, err := client.GetInstrumentForSymbol(symbol)
	if err != nil {
		return fmt.Errorf("error getting instrument: %v", err)
	}
	fmt.Printf("instrument %v", PrettyPrint(instrument))

	quotes, err := client.GetQuote(symbol)
	if err != nil {
		return fmt.Errorf("error getting quote: %v", err)
	}
	fmt.Printf("quote %v", PrettyPrint(quotes[0]))
	return nil
}

func buy1000Stock(symbol string) error {
	client, account, err := getClientAndAccount()
	if err != nil {
		return err
	}
	instrument, err := client.GetInstrumentForSymbol(symbol)
	if err != nil {
		return fmt.Errorf("error getting instrument: %v", err)
	}

	var i int
	ticker := time.NewTicker(200 * time.Millisecond)
	for range ticker.C {
		if i > 1000 {
			break
		}
		randNum := rand.Intn(10)
		i++
		orderReq := &robinhood.OrderRequest{
			Account:       account.URL,
			Instrument:    instrument.URL,
			Symbol:        instrument.Symbol,
			Type:          robinhood.OrderType_Limit,
			TimeInForce:   robinhood.TimeInForce_GoodForDay,
			Trigger:       robinhood.Trigger_Imediate,
			Price:         0.06 + 0.001*float64(randNum),
			Quantity:      1,
			Side:          robinhood.Side_Buy,
			ExtendedHours: true,
		}
		orderResponse, err := client.SendOrder(orderReq)
		if err != nil {
			return fmt.Errorf("error sending order: %v", err)
		}
		fmt.Printf("Order Response %v\n", PrettyPrint(orderResponse))
	}
	return nil
}

func PrettyPrint(v interface{}) string {
	str, _ := json.MarshalIndent(v, "", "	")
	return string(str)
}
