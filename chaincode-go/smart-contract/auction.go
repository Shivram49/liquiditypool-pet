/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

import (
	"crypto/sha256"
	"encoding/base64"
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

type CommonWallet struct {
	Type         string  `json:"objectType"`
	Amount       float32 `json:"amount"`
	EnergyTokens float32 `json:"EnergyTokens"`
	Identity     string  `json:"Identity"`
	WalletId     string  `json:"WalletId"`
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

func (s *SmartContract) ProvideLiquidity(ctx contractapi.TransactionContextInterface, auctionID string, energyUnits float32, price float32, walletId string) error {

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
	//based on the CommonWallet, find out if the client has enough tokens and energy unites to provide liquidity
	//get the wallet of the client
	//TODO : Modularzie this code
	data := walletId + clientID
	hash := sha256.Sum256([]byte(data))
	hashedData := base64.StdEncoding.EncodeToString(hash[:])
	walletAsBytes, err := ctx.GetStub().GetState(hashedData)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if walletAsBytes == nil {
		return fmt.Errorf("the wallet %s does not exist", hashedData)
	}
	var wallet CommonWallet
	err = json.Unmarshal(walletAsBytes, &wallet)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	//check if the client has enough tokens and energy units
	if wallet.Amount < price {
		return fmt.Errorf("the client does not have enough tokens to provide liquidity")
	}
	if wallet.EnergyTokens < energyUnits {
		return fmt.Errorf("the client does not have enough energy units to provide liquidity")
	}
	//update the wallet
	wallet.Amount = wallet.Amount - price
	wallet.EnergyTokens = wallet.EnergyTokens - energyUnits
	walletJson, err := json.Marshal(wallet)
	if err != nil {
		return err
	}
	// put wallet into state
	err = ctx.GetStub().PutState(hashedData, walletJson)
	if err != nil {
		return fmt.Errorf("failed to put auction in public data: %v", err)
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

//Creating a common wallet for the client
func (s *SmartContract) CreateCommonWallet(ctx contractapi.TransactionContextInterface, walletId string, energyTokens float32, amount float32) error {
	clientId, err := s.GetSubmittingClientIdentity(ctx)
	if err != nil {
		fmt.Errorf("failed to get client identity %v", err)
	}
	commonWallet := CommonWallet{
		Type:         "commonWallet",
		Amount:       amount,
		EnergyTokens: energyTokens,
		Identity:     clientId,
		WalletId:     walletId,
	}
	commonWalletJson, err := json.Marshal(commonWallet)
	if err != nil {
		return err
	}
	//hash concatenate walletId & clientId together
	data := walletId + clientId
	hash := sha256.Sum256([]byte(data))
	hashedWalletId := base64.StdEncoding.EncodeToString(hash[:])
	// put wallet into state
	err = ctx.GetStub().PutState(hashedWalletId, commonWalletJson)
	if err != nil {
		return fmt.Errorf("failed to put auction in public data: %v", err)
	}
	return nil
}

func round(f float32) float32 {
	return float32(math.Round(float64(f)*100) / 100)
}
func (s *SmartContract) BuyEnergy(ctx contractapi.TransactionContextInterface, auctionID string, buyingPrice float32, walletId string) error {

	// get ID of submitting client
	clientID, err := s.GetSubmittingClientIdentity(ctx)
	// get the MSP ID of the bidder's org
	clientOrgID, err := ctx.GetClientIdentity().GetMSPID()

	fmt.Print(clientOrgID)
	//based on the CommonWallet, find out if the client has enough tokens unites to buy energy
	//get the wallet of the client
	//TODO: Make sure to modularize this code
	data := walletId + clientID
	hash := sha256.Sum256([]byte(data))
	walletIdHash := base64.StdEncoding.EncodeToString(hash[:])
	walletAsBytes, err := ctx.GetStub().GetState(walletIdHash)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if walletAsBytes == nil {
		return fmt.Errorf("the wallet %s does not exist", walletIdHash)
	}

	var wallet CommonWallet
	err = json.Unmarshal(walletAsBytes, &wallet)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	//check if the client has enough tokens
	if wallet.Amount < buyingPrice {
		return fmt.Errorf("the client does not have enough tokens to buy energy")
	}
	// get the auction from public state
	pool, err := s.QueryPool(ctx, auctionID)
	if err != nil {
		return fmt.Errorf("failed to get auction from public state %v", err)
	}

	currPrice := pool.TokensPerUnit
	totalBought := round(buyingPrice / currPrice)
	pool.EnergyUnits = round(pool.EnergyUnits - totalBought)
	//update the wallet
	wallet.Amount = wallet.Amount - buyingPrice
	wallet.EnergyTokens += totalBought
	//adding the money to the pool
	pool.TotalTokens = round(pool.TotalTokens + buyingPrice)

	//the price needs to be updated
	pool.TokensPerUnit = round(pool.TotalTokens / pool.EnergyUnits)

	if err != nil {
		return fmt.Errorf("failed to get client MSP ID: %v", err)
	}

	// Add the bidding organization to the list of participating organizations if it is not already

	newPoolJson, _ := json.Marshal(pool)
	walletJson, err := json.Marshal(wallet)
	if err != nil {
		return err
	}
	// Update values of the state
	err = ctx.GetStub().PutState(auctionID, newPoolJson)
	err = ctx.GetStub().PutState(walletIdHash, walletJson)
	if err != nil {
		return fmt.Errorf("failed to update auction: %v", err)
	}

	return nil
}
