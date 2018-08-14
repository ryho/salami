package main

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ryho/go-robinhood"
)

const (
	MaxPennyStockPrice = .50
	MaxFractionalCents = .50
)

func sliceSalami(stockSymbol string, numberToBuy int, useStop bool) error {
	if !robinhood.IsRobinhoodExtendedTradingTime() {
		return errors.New("it is outside of trading hours")
	}
	client, account, err := getClientAndAccount()
	if err != nil {
		return err
	}

	instrument, err := client.GetInstrumentForSymbol(stockSymbol)
	if err != nil {
		return fmt.Errorf("error getting instrument: %v", err)
	}
	if !instrument.Tradeable {
		return errors.New("stock is not tradeable on Robin Hood")
	}

	quotes, err := client.GetQuote(stockSymbol)
	if err != nil {
		return fmt.Errorf("error getting quote: %v", err)
	}
	if quotes[0].TradingHalted {
		return errors.New("trading is halted for this stock")
	}
	if quotes[0].LastTradePrice > MaxPennyStockPrice {
		return errors.New("stock is too expensive to make money")
	}
	// The stock has to be priced at $0.0500 to $0.055 in order to make money from rounding down.
	priceCents := quotes[0].LastTradePrice * 100
	floorCents := math.Floor(priceCents)
	if priceCents-floorCents > MaxFractionalCents {
		return errors.New("can't make money on this stock")
	}

	fmt.Printf("Attempting to buy %v (%v)\n", instrument.Name, instrument.Symbol)
	var i int
	var orderIds []string
	ticker := time.NewTicker(1000 * time.Millisecond)
	for range ticker.C {
		i++
		if i > numberToBuy {
			break
		}
		//randNum := rand.Intn(10)
		// Price:floorCents/100+0.001*float64(randNum),
		orderReq := &robinhood.OrderRequest{
			Side:          robinhood.Side_Buy,
			Account:       account.URL,
			Instrument:    instrument.URL,
			Symbol:        instrument.Symbol,
			Type:          robinhood.OrderType_Limit,
			TimeInForce:   robinhood.TimeInForce_GoodForDay,
			Trigger:       robinhood.Trigger_Imediate,
			Price:         floorCents/100 + 0.005,
			Quantity:      1,
			ExtendedHours: true,
		}
		if useStop {
			// Prevent purchases of stock that is too low.
			orderReq.Trigger = robinhood.Trigger_Stop
			orderReq.StopPrice = floorCents/100 + .001
		}
		order, err := client.SendOrder(orderReq)
		if err != nil {
			return fmt.Errorf("error sending order: %v", err)
		}
		fmt.Printf("Order Response %v\n", PrettyPrint(order))
		switch order.State {
		case robinhood.OrderState_Failed:
			return errors.New("order failed")
		case robinhood.OrderState_Rejected:
			return fmt.Errorf("order rejected: %v", order.RejectReason)
		case robinhood.OrderState_Canceled:
			return errors.New("order canceled")
		case robinhood.OrderState_Filled:
		default:
			// Save them for later so that we can check them.
			// Don't bother saving ones that were filled instantly.
			orderIds = append(orderIds, order.Id)
		}
	}
	// Wait for orders to be filled
	for len(orderIds) > 0 {
		var newOrderIds []string
		for _, orderId := range orderIds {
			order, err := client.GetOrder(orderId)
			if err != nil {
				return err
			}
			switch order.State {
			case robinhood.OrderState_Failed:
				return errors.New("order failed")
			case robinhood.OrderState_Rejected:
				return fmt.Errorf("order rejected: %v", order.RejectReason)
			case robinhood.OrderState_Canceled:
				return errors.New("order canceled")
			case robinhood.OrderState_Filled:
			default:
				newOrderIds = append(newOrderIds, order.Id)
			}
			time.Sleep(time.Second)
		}
		orderIds = newOrderIds
	}

	// Sell all the stocks at once to reap the rounding benefits
	orderReq := &robinhood.OrderRequest{
		Side:          robinhood.Side_Sell,
		Account:       account.URL,
		Instrument:    instrument.URL,
		Symbol:        instrument.Symbol,
		Type:          robinhood.OrderType_Limit,
		TimeInForce:   robinhood.TimeInForce_GoodForDay,
		Trigger:       robinhood.Trigger_Imediate,
		Price:         floorCents/100 + 0.001,
		Quantity:      numberToBuy,
		ExtendedHours: true,
	}
	fmt.Println("Sending sell")
	order, err := client.SendOrder(orderReq)
	if err != nil {
		return fmt.Errorf("error sending order: %v", err)
	}
	for {
		switch order.State {
		case robinhood.OrderState_Failed:
			return errors.New("order failed")
		case robinhood.OrderState_Rejected:
			return fmt.Errorf("order rejected: %v", order.RejectReason)
		case robinhood.OrderState_Canceled:
			return errors.New("order failed")
		case robinhood.OrderState_Filled:
			fmt.Printf("Successfully sold for an average price of %v\n", order.AveragePrice)
			return nil
		}
		order, err = client.GetOrder(order.Id)
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}
