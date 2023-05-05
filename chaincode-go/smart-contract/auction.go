/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

// Auction data
type Auction struct {
	Type         string             `json:"objectType"`
	ItemSold     string             `json:"item"`
	Seller       string             `json:"seller"`
	Orgs         []string           `json:"organizations"`
	PrivateBids  map[string]BidHash `json:"privateBids"`
	RevealedBids map[string]FullBid `json:"revealedBids"`
	Winner       string             `json:"winner"`
	Price        int                `json:"price"`
	Status       string             `json:"status"`
}

type Pool struct {
	Type          string  `json:"objectType"`
	EnergyUnits   float32 `json:"energyUnits"`
	TotalTokens   float32 `json:"totalTokens"`
	TokensPerUnit float32 `json:"tokensPerUnit"`
}

type commonWallet struct {
	Type     string  `json:"objectType"`
	Amount   float32 `json:"amount"`
	identity string  `json:"identity"`
}

// FullBid is the structure of a revealed bid
type FullBid struct {
	Type   string `json:"objectType"`
	Price  int    `json:"price"`
	Org    string `json:"org"`
	Bidder string `json:"bidder"`
}

// BidHash is the structure of a private bid
type BidHash struct {
	Org  string `json:"org"`
	Hash string `json:"hash"`
}

const bidKeyType = "bid"

func (s *SmartContract) ProvideLiquidity(ctx contractapi.TransactionContextInterface, auctionID string, energyUnits float32, price float32) error {

	// get ID of submitting client
	clientID, err := s.GetSubmittingClientIdentity(ctx)

	//print clientId
	fmt.Println(clientID)

	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	// get org of submitting client
	clientOrgID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	// Create pool
	pool := Pool{
		Type:          "pool",
		EnergyUnits:   energyUnits,
		TotalTokens:   price,
		TokensPerUnit: price / energyUnits,
	}

	poolJson, err := json.Marshal(pool)
	if err != nil {
		return err
	}

	// put auction into state
	err = ctx.GetStub().PutState(auctionID, poolJson)
	if err != nil {
		return fmt.Errorf("failed to put auction in public data: %v", err)
	}

	// set the seller of the auction as an endorser
	err = setAssetStateBasedEndorsement(ctx, auctionID, clientOrgID)
	if err != nil {
		return fmt.Errorf("failed setting state based endorsement for new organization: %v", err)
	}

	return nil
}

func round(f float32) float32 {
	return float32(math.Round(float64(f)*100) / 100)
}
func (s *SmartContract) BuyEnergy(ctx contractapi.TransactionContextInterface, auctionID string, buyingPrice float32) error {

	// get the MSP ID of the bidder's org
	clientOrgID, err := ctx.GetClientIdentity().GetMSPID()

	fmt.Print(clientOrgID)
	// get the auction from public state
	pool, err := s.QueryPool(ctx, auctionID)
	if err != nil {
		return fmt.Errorf("failed to get auction from public state %v", err)
	}

	currPrice := pool.TokensPerUnit
	totalBought := round(buyingPrice / currPrice)
	pool.EnergyUnits = round(pool.EnergyUnits - totalBought)

	//adding the money to the pool
	pool.TotalTokens = round(pool.TotalTokens + buyingPrice)

	//the price needs to be updated
	pool.TokensPerUnit = round(pool.TotalTokens / pool.EnergyUnits)

	if err != nil {
		return fmt.Errorf("failed to get client MSP ID: %v", err)
	}

	// Add the bidding organization to the list of participating organizations if it is not already

	newPoolJson, _ := json.Marshal(pool)

	err = ctx.GetStub().PutState(auctionID, newPoolJson)
	if err != nil {
		return fmt.Errorf("failed to update auction: %v", err)
	}

	return nil
}
