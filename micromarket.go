package main

import (
    "errors"
    "fmt"
    "sort"
    "strconv"
    "crypto/md5"
    "encoding/hex"
    // "strings"
    "github.com/hyperledger/fabric/core/chaincode/shim"
    "encoding/json"
    "crypto/x509"
    "reflect"
    "encoding/asn1"
    "encoding/pem"
    "net/http"
    "net/url"
    "io/ioutil"
    // "regexp"
)

const   ROLE_MARKET_MAKER   =  0
const   ROLE_MANAGER        =  1
const   ROLE_PRIVATE_ENTITY =  2
const   ROLE_EXCHANGE       =  3

const   PROPERTY_STATE_PROPOSED      =  0
const   PROPERTY_STATE_MANAGED       =  1
const   PROPERTY_STATE_RECLAIMED     =  2

const   ACCOUNT_STATE_ACTIVE       =  0
const   ACCOUNT_STATE_INACTIVE     =  1

const   LOG_DEBUG           =  1
const   LOG_INFO            =  2
const   LOG_WARN            =  3
const   LOG_ERROR           =  4

const   TRADE_BUY           =  "B"
const   TRADE_SELL          =  "S"

const   PROPERTY_PREFIX     = "property:"
const   ACCOUNT_PREFIX      = "account:"
const   TRDING_PRPTY_PREFIX = "trdprpty:"
const   OFFER_PREFIX        = "offer:"
const   ACCT_TRADES_PREFIX  = "accttrades:"
const   PRPTY_TRADES_PREFIX = "prptytrades:"


//==============================================================================================================================
//     Structure Definitions 
//==============================================================================================================================
//    Chaincode
//==============================================================================================================================
type SimpleChaincode struct {
}

//==============================================================================================================================
//    Log
//==============================================================================================================================
type Log struct {
}

//==============================================================================================================================
//    Configuration
//==============================================================================================================================
type Configuration struct {
    logLevel        int         `json:"logLevel"`
}

//==============================================================================================================================
//    Property
//==============================================================================================================================
type Property struct {
  //details
    ID              string      `json:"propertyID"`
    AddressLine     string      `json:"addressLine"`
    Suburb          string      `json:"suburb"`
    State           string      `json:"state"`
    PostCode        string      `json:"postcode"`
    
  //info
    ManagedBy       string      `json:"managedBy"`
    Issuer          string      `json:"issuer"`
    Units           int         `json:"units"`
    Status          int         `json:"status"`
    
/*
  //comparison
    Bedrooms        int         `json:"bedrooms,omitempty"`
    Bathrooms       int         `json:"bathrooms,omitempty"`
    Squares         int         `json:"squares,omitempty"`
    Size            int         `json:"size,omitempty"`
    Zoning          int         `json:"zoning,omitempty"`

  //financials
    Rented          bool        `json:"rented,omitempty"`
    Rent            int         `json:"rent,omitempty"`
    LastPayment     int         `json:"lastPaymentDate,omitempty"`
    Valuation       int         `json:"valution,omitempty"`
    ValuationDate   int         `json:"valuationDate,omitempty"`
  */
}

//==============================================================================================================================
//    Account
//==============================================================================================================================
type Account struct {
    ID              string      `json:"accountID"`
    Cash            float64     `json:"cash"`
    Status          int         `json:"status"`
    Holdings        []Holding   `json:"holdings"`
}

//==============================================================================================================================
//    Holding
//==============================================================================================================================
type Holding struct {
    Entity          string      `json:"entity"`
    Units           int         `json:"units"`
}

//==============================================================================================================================
//    TradeMap
//==============================================================================================================================
type TradeMap struct {
    Trades      map[string]Trade  `json:"trades"`
}

//==============================================================================================================================
//    Trade
//==============================================================================================================================
type Trade struct {
    ID              string      `json:"tradeID"`
    AccountID       string      `json:"accountID"`
    PropertyID      string      `json:"propertyID"`
    Direction       string      `json:"direction"`
    Price           float64     `json:"price"`
    Units           int         `json:"units"`
    Escrow          float64     `json:"escrow"`
}

//==============================================================================================================================
//    ReturnTrade
//==============================================================================================================================
type ReturnTrade struct {
    PropertyID      string      `json:"propertyID"`
    Direction       string      `json:"direction"`
    Price           float64     `json:"price"`
    Units           int         `json:"units"`
}

//==============================================================================================================================
//    TradingProperties
//==============================================================================================================================
type TradingProperties struct {
    PropertyIDs  map[string]string  `json:"propertyIDs"`
}

//==============================================================================================================================
//    Offer
//==============================================================================================================================
type Offer struct {
    ID              string      `json:"offerID"`
    PropertyID      string      `json:"propertyID"`
    Direction       string      `json:"direction"`
    Price           float64     `json:"price"`
    Units           int         `json:"units"`
}

//==============================================================================================================================
//    ECertResponse
//==============================================================================================================================
type ECertResponse struct {
    OK string `json:"OK"`
}

//==============================================================================================================================
//     SimpleChaincode Lifecycle Functions
//=================================================================================================================================
//     Main - main - Starts up the chaincode
//=================================================================================================================================

var log    Log
var config Configuration

func main() {
    err := shim.Start(new(SimpleChaincode))
    if checkErrors(err){fmt.Printf("Error starting Chaincode: %s", err)}
}

//==============================================================================================================================
//    Init Function - Called when the user deploys the chaincode                                                                    
//==============================================================================================================================
func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    //authenticate the user
    //caller_ecert, caller_role, err := t.get_user_data(stub, args[0])
    //if checkErrors(err){return nil, err}
    
    var err error
   /* 
    //make sure we have been configured up front
    if config.logLevel == 0 && function != "configure" {
        return nil, errors.New("Application hasn't been configured")
    }
    */

    //set the log level
    switch function {
        case "configure":
            configure(args)
        case "test":
            log.debug("*************************************")
            log.debug(" Running Tests")
            log.debug("*************************************")
            var output []string
            output = append(output, t.testAccountCreateSuccess(stub, "testaccount")...)

            sort.Strings(output)
            for i := 0; i<len(output); i++ {log.debug(output[i])}

        case "demo":
            config.logLevel = LOG_DEBUG

            //create the cardy account
            t.createAccount(stub, []string{"cardy"})
            t.depositCash(stub, []string{"cardy", "1000000"})
            t.issueProperty(stub, []string{`{addressLine: "30 Oakwood St", suburb: "Sutherland", state: "NSW", postcode: "2232", issuer: "cardy", units: 10000, valuation: 10000000}`})

            t.createAccount(stub, []string{"cripps"})
            t.depositCash(stub, []string{"cripps", "200000"})
            t.issueProperty(stub, []string{`{addressLine: "25a National Ave", suburb: "Loftus", state: "NSW", postcode: "2232", issuer: "cripps", units: 1400, valuation: 14000000}}`})
            t.issueProperty(stub, []string{`{addressLine: "43a Belmont St", suburb: "Sutherland", state: "NSW", postcode: "2232", issuer: "cripps", units: 800, valuation: 12000000}}`})

            
            t.createAccount(stub, []string{"m123456"})
            t.depositCash(stub, []string{"m123456", "200000"})

        default:
            err = errors.New("You must choose an initialisation mode")
    }
    config.logLevel = LOG_DEBUG

    return nil, err
 }

//=================================================================================================================================    
//    Query - Called on chaincode query. Takes a function name passed and calls that function. Passes the
//          initial arguments passed are passed on to the called function.
//=================================================================================================================================    
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    //authenticate the user
    //caller_ecert, caller_role, err := t.get_user_data(stub, args[0])
    //if checkErrors(err){return nil, err}

    // switch function {
    //     case "login":
    //         return t.login(stub, args)
    //     case "getAccount":
    //         return t.getAccount(stub, args)
    //     case "getProperties":
    //         return t.getProperties(stub, args)
    //     case "getOpenTradesByAccount":
    //         return t.getOpenTradesByAccount(stub, args)
    //     case "getAvailableTrades":
    //         return t.getAvailableTrades(stub, args)
    //     default:
    //         return nil, errors.New("Invalid function (" + function + ") called")
    // }
    if function == "login" {
        return t.login(stub, args)
    } else if function == "getAccount" {
        return t.getAccount(stub, args)
    } else if function == "getProperties" {
        return t.getProperties(stub, args)        
    } else if function == "getOpenTradesByAccount" {
        return t.getOpenTradesByAccount(stub, args)
    } else if function == "getAvailableTrades" {
        return t.getAvailableTrades(stub, args)
    } else {
        return nil, errors.New("Invalid function (" + function + ") called")
    }
}

//==============================================================================================================================
//    Invoke - Called on chaincode invoke. Takes a function name passed and calls that function. Converts some
//             initial arguments passed to other things for use in the called function e.g. name -> ecert
//==============================================================================================================================
func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    //authenticate the user
    //caller_ecert, caller_role, err := t.get_user_data(stub, args[0])
    //if checkErrors(err){return nil, err}

    // switch function {
    //     case "depositCash":
    //         return t.depositCash(stub, args)
    //     case "withdrawCash":
    //         return t.withdrawCash(stub, args)
    //     case "createTrade":
    //         return t.createTrade(stub, args)
    //     case "createAccount":
    //         return t.createAccount(stub, args)
    //     case "issueProperty":
    //         return t.issueProperty(stub, args)
    //     case "generateOffer":
    //         return t.generateOffer(stub, args)
    //     case "acceptOffer":
    //         return t.acceptOffer(stub, args)
    //     default:
    //         return nil, errors.New("Invalid function (" + function + ") called")
    // }
    if function == "depositCash" {
        return t.depositCash(stub, args)        
    } else if function == "withdrawCash" {
        return t.withdrawCash(stub, args)
    } else if function == "createTrade" {
        return t.createTrade(stub, args)    
    } else if function == "createAccount" {
        return t.createAccount(stub, args)        
    } else if function == "issueProperty" {
        return t.issueProperty(stub, args)
    } else if function == "generateOffer" {
        return t.generateOffer(stub, args) 
    } else if function == "acceptOffer" {
        return t.acceptOffer(stub, args)
    } else {
        return nil, errors.New("Invalid function (" + function + ") called")     
    }

}

//==============================================================================================================================
//     Query Logic Methods
//==============================================================================================================================
//     login
//==============================================================================================================================
func (t *SimpleChaincode) login(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
//login(accountID string)
    //currently just return the account
    return t.getAccount(stub, args)
}

//==============================================================================================================================
//     getAccount
//==============================================================================================================================
func (t *SimpleChaincode ) getAccount(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    //getAccount(accountID string)
    if len(args) != 1 {return nil, errors.New("Incorrect number of arguments passed")}
    accountID := args[0]

    account, err := getAccount(stub, accountID)
    if checkErrors(err) {return nil, err}
    
    return account.marshal()

}

//==============================================================================================================================
//     getProperties
//==============================================================================================================================
func (t *SimpleChaincode) getProperties(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    //getProperties(propertyIDs []string)
    var propertyIDs = args

    var properties []Property

    for i:=0;i<len(propertyIDs);i++ {
        property, err := getProperty(stub, propertyIDs[i])
        if checkErrors(err) {return nil, err}
        properties = append(properties, property)
    }

    return marshalProperties(properties)
}

//==============================================================================================================================
//     getOpenTradesByAccount
//==============================================================================================================================
func (t *SimpleChaincode ) getOpenTradesByAccount(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    //getOpenTradesByAccount(accountID string)
    if len(args) != 1 {return nil, errors.New("Incorrect number of arguments passed")}
    accountID := args[0]

    var account Account
    account.ID = accountID

    trades, err := account.getTrades(stub)
    if checkErrors(err) {return nil, err}

    return marshalTrades(trades)
}

//==============================================================================================================================
//     getAvailableTrades
//==============================================================================================================================
func (t *SimpleChaincode ) getAvailableTrades(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    //getAvailableTrades()
    if len(args) != 0 {return nil, errors.New("Incorrect number of arguments passed")}
    
    propertyIDs, err := getTradingProperties(stub)
    if checkErrors(err){return nil, err}

    var returnTrades []ReturnTrade

    for i:=0;i<len(propertyIDs);i++ {
        //for this property create a return trade
        var returnTrade ReturnTrade
        returnTrade.PropertyID = propertyIDs[i]

        var value float64
        trades, err := getPropertyTrades(stub, propertyIDs[i])
        if checkErrors(err){return nil, err}

        for j:=0;j<len(trades);j++ {
            returnTrade.Units += trades[j].Units
            tradeValue := float64(trades[j].Units) * trades[j].Price
            value += tradeValue
        }
        if returnTrade.Units == 0 {continue}
        
        returnTrade.Direction = trades[0].Direction
        returnTrade.Price = value / float64(returnTrade.Units)
        returnTrades = append(returnTrades, returnTrade)
    }

    return marshalReturnTrades(returnTrades)
}

//==============================================================================================================================
//     Invoke Logic Methods
//==============================================================================================================================
//     depositCash - Transfer cash into a blockchain account
//==============================================================================================================================
func (t *SimpleChaincode ) depositCash(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    //depositCash(accountID, value)
    if len(args) != 2 {return nil, errors.New("Incorrect number of arguments passed")}

    account, err := getAccount(stub, args[0])
    if checkErrors(err){return nil, err}

    cashValue, err := strconv.ParseFloat(args[1], 64)
    if checkErrors(err){return nil, errors.New("Could not parse "+args[1]+" to float")}

    account.Cash += cashValue
    account.save(stub)
    return nil, err
}

//==============================================================================================================================
//     withdrawCash - Transfer cash out of a blockchain account
//==============================================================================================================================
func (t *SimpleChaincode ) withdrawCash(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    //withdrawCash(accountID, value)
    if len(args) != 2 {return nil, errors.New("Incorrect number of arguments passed")}

    account, err := getAccount(stub, args[0])
    if checkErrors(err){return nil, err}

    cashValue, err := strconv.ParseFloat(args[1], 64)
    if checkErrors(err){return nil, errors.New("Could not parse "+args[1]+" to float")}

    if account.Cash < cashValue {
        return nil, errors.New("Not enough cash to withdraw")
    }
    account.Cash -= cashValue
    account.save(stub)
    return nil, err
}

//==============================================================================================================================
//     createTrade - Purchase units of a property
//==============================================================================================================================
func (t *SimpleChaincode ) createTrade(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    //createTrade(trade string) {accountID: "m123456", direction: "S", propertyID: "qwer1234", price: "100.00", units: "10"}
    if len(args) != 1 {return nil, errors.New("Incorrect number of arguments passed")}

    return nil, nil
}

//==============================================================================================================================
//     createAccount - Create an account for a user
//==============================================================================================================================
func (t *SimpleChaincode ) createAccount(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

    if len(args) != 1 {return nil, errors.New("Incorrect number of arguments passed")}
    accountID := args[0]

    if len(args) != 1 {return nil, errors.New("Incorrect number of arguments passed")}
    var account Account
    account.ID = accountID
    account.Status = ACCOUNT_STATE_ACTIVE
    if account.exists(stub) {return nil, errors.New("account already exists")}
    return nil, account.create(stub)
}

//==============================================================================================================================
//     generateOffer - buy into a new property issue
//==============================================================================================================================
func (t *SimpleChaincode ) generateOffer(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    //generateOffer(propertyID string, units int)
    if len(args) != 2 {return nil, errors.New("Incorrect number of arguments passed")}
    //propertyID := args[0]
    //units, err := strconv.Atoi(args[1])
    //if checkErrors(err) {return nil, err}

    return nil, nil
}

//==============================================================================================================================
//     acceptOffer - buy into a new property issue
//==============================================================================================================================
func (t *SimpleChaincode ) acceptOffer(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    //acceptOffer(offerID string, accountID string)
    if len(args) != 2 {return nil, errors.New("Incorrect number of arguments passed")}
    //offerID := args[0]
    //accountID := args[1]

    return nil, nil
}

//==============================================================================================================================
//     issueProperty - Issue a property for trading on the block chain. The property's units will automatically be assigned
//                     to the account of the issuer
//==============================================================================================================================
func (t *SimpleChaincode ) issueProperty(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

    log.debug("check issueProperty args")
    if len(args) != 1 {return nil, errors.New("Incorrect number of arguments. Expecting property json")}

    log.debug("unmarshalling " + args[0])
    property, err := unmarshalProperty([]byte(args[0]))
    if checkErrors(err){return nil, err}
    
    log.debug("creating the property in the blockchain")
    err = property.create(stub)
    if checkErrors(err){return nil, err}
    
    log.debug("get the account for the issuer " + property.Issuer)
    issuerAccount, err := getAccount(stub, property.Issuer)
    if checkErrors(err){return nil, err}

    log.debug("Set the issuer to be the owner of all units")
    issuerAccount.changeHolding(property.ID, property.Units)
    if checkErrors(err){return nil, err}

    log.debug("save the issuer's account")
    err = issuerAccount.save(stub)
    if checkErrors(err){return nil, err}

    log.debug("now create an account for the property with an initial view of the holdings")
    var propertyAccount Account
    propertyAccount.ID = property.ID
    propertyAccount.Cash = 0
    propertyAccount.changeHolding(property.Issuer, property.Units)
    if checkErrors(err){return nil, err}

    err = propertyAccount.create(stub)
    if checkErrors(err){return nil, err}    
    
    log.info("Issued property " + property.ID)

    return nil, nil
}

//==============================================================================================================================
//     CRUD Subroutines
//==============================================================================================================================
//     Property
//==============================================================================================================================
func getProperty(stub *shim.ChaincodeStub, id string) (Property, error) {
    var object Property
    bytes, err := stub.GetState(PROPERTY_PREFIX + id)
    if checkErrors(err){return object, errors.New("Couldn't retrieve property for " + id)}

    object, err = unmarshalProperty(bytes)
    if checkErrors(err){return object, err}

    return object, nil
}

func getTradingProperties(stub *shim.ChaincodeStub) ([]string, error) {
    var tradingProperties TradingProperties
    var propertyIDs []string

    bytes, err := stub.GetState(TRDING_PRPTY_PREFIX)
    if checkErrors(err){return nil, errors.New("Couldn't retrieve trading properties")}

    tradingProperties, err = unmarshalTradingProperties(bytes)
    if checkErrors(err){return nil, err}

    for _, value := range tradingProperties.PropertyIDs {
        propertyIDs = append(propertyIDs, value)
    }

    return propertyIDs, nil
}

func getPropertyTrades(stub *shim.ChaincodeStub, propertyID string) ([]Trade, error) {
    var object Property
    object.ID = propertyID
    return object.getTrades(stub)
}

func (object *Property) getTrades(stub *shim.ChaincodeStub) ([]Trade, error) {
    var trades []Trade
    var tradeMap map[string]Trade
    if object.ID == "" {return trades, errors.New("Need a property ID to search on")}

    bytes, err := stub.GetState(PRPTY_TRADES_PREFIX + object.ID)
    if checkErrors(err){return trades, errors.New("Couldn't retrieve trades for " + object.ID)}
    if bytes != nil && len(bytes) > 0 {
        tradeMapObj, err := unmarshalTradeMap(bytes)
        if checkErrors(err){return trades, err}
        tradeMap = tradeMapObj.Trades
    } else {
        tradeMap = map[string]Trade{}
    }

    for _, value := range tradeMap {
        trades = append(trades, value)
    }

    return trades, nil
}

func (object *Property) create(stub *shim.ChaincodeStub) error {
    err := object.validate()
    if checkErrors(err){return err}

    if object.ID != "" {return errors.New("Can't create property with ID already assigned")}
    object.ID = getMd5Hash(object.AddressLine + object.Suburb + object.State + object.PostCode)
    if object.exists(stub) {return errors.New("A property with this ID already exists")}

    return object.save(stub)
}

func (object *Property) save(stub *shim.ChaincodeStub) error {
    bytes, err := object.marshal()
    if checkErrors(err){return err}
    
    err = stub.PutState(PROPERTY_PREFIX + object.ID, bytes)
    if checkErrors(err){return errors.New("Couldn't save property for " + object.ID + " " + object.AddressLine)}

    return nil
}

func deleteProperty(stub *shim.ChaincodeStub, id string) error {
    object, err := getProperty(stub, id)
    if checkErrors(err){return err}
    
    return object.delete(stub)
}

func (object *Property) delete(stub *shim.ChaincodeStub) error {
    object.Status = PROPERTY_STATE_RECLAIMED
    err := object.save(stub)
    if checkErrors(err){return errors.New("Couldn't delete property for " + object.ID)}

    return nil
}

func (object *Property) exists(stub *shim.ChaincodeStub) bool {
    bytes, err := stub.GetState(PROPERTY_PREFIX + object.ID)
    return bytes != nil || err != nil
}

func (object *Property) validate() error {
    return nil
}

//==============================================================================================================================
//     Account
//==============================================================================================================================
func getAccount(stub *shim.ChaincodeStub, id string) (Account, error) {
    var object Account
    bytes, err := stub.GetState(ACCOUNT_PREFIX + id)
    if checkErrors(err){return object, errors.New("Couldn't retrieve account for " + id)}

    object, err = unmarshalAccount(bytes)
    if checkErrors(err){return object, err}

    return object, nil
}

func getAccountTrades(stub *shim.ChaincodeStub, accountID string) ([]Trade, error) {
    var object Account
    object.ID = accountID
    return object.getTrades(stub)
}

func (object *Account) getTrades(stub *shim.ChaincodeStub) ([]Trade, error) {
    var trades []Trade
    var tradeMap map[string]Trade
    if object.ID == "" {return trades, errors.New("Need an account ID to search on")}

    bytes, err := stub.GetState(ACCT_TRADES_PREFIX + object.ID)
    if checkErrors(err){return trades, errors.New("Couldn't retrieve trades for " + object.ID)}
    if bytes != nil && len(bytes) > 0 {
        tradeMapObj, err := unmarshalTradeMap(bytes)
        if checkErrors(err){return trades, err}
        tradeMap = tradeMapObj.Trades
    } else {
        tradeMap = map[string]Trade{}
    }

    for _, value := range tradeMap {
        trades = append(trades, value)
    }

    return trades, nil
}

func (object *Account) create(stub *shim.ChaincodeStub) error {
    err := object.validate()
    if checkErrors(err){return err}

    if object.ID == "" {return errors.New("An account needs to be assigned to an owner")}
    if object.exists(stub){return errors.New("This account already exists")}

    return object.save(stub)
}

func (object *Account) save(stub *shim.ChaincodeStub) error {
    bytes, err := object.marshal()
    if checkErrors(err){return err}
    
    err = stub.PutState(ACCOUNT_PREFIX + object.ID, bytes)
    if checkErrors(err){return errors.New("Couldn't save account for " + object.ID)}

    return nil
}

func deleteAccount(stub *shim.ChaincodeStub, id string) error {
    object, err := getAccount(stub, id)
    if checkErrors(err){return err}
    
    return object.delete(stub)
}

func (object *Account) delete(stub *shim.ChaincodeStub) error {
    object.Status = ACCOUNT_STATE_INACTIVE
    err := object.save(stub)
    if checkErrors(err){return errors.New("Couldn't delete account for " + object.ID)}

    return nil
}

func (object *Account) exists(stub *shim.ChaincodeStub) bool {
    bytes, err := stub.GetState(ACCOUNT_PREFIX + object.ID)
    return bytes != nil || err != nil
}

func (object *Account) validate() error {
    return nil
}

func (object *Account) changeHolding(entity string, unitsDelta int) error {
    var holding Holding
    var found bool
    for i := 0; i < len(object.Holdings) && !found; i++ {
		if object.Holdings[i].Entity == entity {
            holding = object.Holdings[i]
            found = true
        }
	}

    if !found {
        holding.Entity = entity
        object.Holdings = append(object.Holdings, holding)
    }
    
    var finalUnits int
    finalUnits = holding.Units + unitsDelta
    if (finalUnits < 0) {
        return errors.New("There are not enough units to make this trade")
    }

    holding.Units = finalUnits

    return nil
}


//==============================================================================================================================
//     Trade
//==============================================================================================================================


//==============================================================================================================================
//     Parsing Subroutines
//==============================================================================================================================
//     Property
//==============================================================================================================================
func unmarshalProperty(bytes []byte) (Property, error) {
    var object Property
    err := json.Unmarshal(bytes, &object)
    if checkErrors(err){return object, errors.New("Error unmarshalling property")}
    return object, nil
}

func marshalProperties(objects []Property) ([]byte, error) {
    bytes, err := json.Marshal(objects)
    if checkErrors(err){return nil, errors.New("Error marshalling property array")}
    return bytes, nil
}

func (object *Property) marshal() ([]byte, error) {
    bytes, err := json.Marshal(object)
    if checkErrors(err){return nil, errors.New("Error marshalling property")}
    return bytes, nil
}

func (object *TradingProperties) marshal() ([]byte, error) {
    bytes, err := json.Marshal(object)
    if checkErrors(err){return nil, errors.New("Error marshalling trading properties")}
    return bytes, nil
}

//==============================================================================================================================
//     Account
//==============================================================================================================================
func unmarshalAccount(bytes []byte) (Account, error) {
    var object Account
    err := json.Unmarshal(bytes, &object)
    if checkErrors(err){return object, errors.New("Error unmarshalling account")}
    return object, nil
}

func marshalAccounts(objects []Account) ([]byte, error) {
    bytes, err := json.Marshal(objects)
    if checkErrors(err){return nil, errors.New("Error marshalling account array")}
    return bytes, nil
}

func (object *Account) marshal() ([]byte, error) {
    bytes, err := json.Marshal(object)
    if checkErrors(err){return nil, errors.New("Error marshalling account")}
    return bytes, nil
}

//==============================================================================================================================
//     Trades
//==============================================================================================================================
func unmarshalTradeMap(bytes []byte) (TradeMap, error) {
    var object TradeMap
    err := json.Unmarshal(bytes, &object)
    if checkErrors(err){return object, errors.New("Error unmarshalling trade map")}
    return object, nil
}

func unmarshalTradingProperties(bytes []byte) (TradingProperties, error) {
    var object TradingProperties
    err := json.Unmarshal(bytes, &object)
    if checkErrors(err){return object, errors.New("Error unmarshalling trading properties")}
    return object, nil
}

func marshalTrades(objects []Trade) ([]byte, error) {
    bytes, err := json.Marshal(objects)
    if checkErrors(err){return nil, errors.New("Error marshalling trade array")}
    return bytes, nil
}

func marshalReturnTrades(objects []ReturnTrade) ([]byte, error) {
    bytes, err := json.Marshal(objects)
    if checkErrors(err){return nil, errors.New("Error marshalling return trade array")}
    return bytes, nil
}

//==============================================================================================================================
//     Generic
//==============================================================================================================================
func marshalStringArray(objects []string) ([]byte, error) {
    bytes, err := json.Marshal(objects)
    if checkErrors(err){return nil, errors.New("Error marshalling trade array")}
    return bytes, nil
}

//==============================================================================================================================
//     Utility Subroutines
//==============================================================================================================================
//     Configure
//==============================================================================================================================
func configure(args []string) error {
    err := json.Unmarshal([]byte(args[0]), &config)
    if checkErrors(err){return errors.New("Error unmarshalling configuration")}
    return nil
}

//==============================================================================================================================
//     Logging
//==============================================================================================================================
func (l *Log) debug(text string) {
    l.log(LOG_DEBUG, text)
}

func (l *Log) info(text string) {
    l.log(LOG_INFO, text)
}

func (l *Log) warn(text string) {
    l.log(LOG_WARN, text)
}

func (l *Log) error(text string) {
    l.log(LOG_ERROR, text)
}

func (l *Log) log(logLevel int, text string) {
    var prefix string
    switch logLevel {
        case LOG_DEBUG:
            prefix = "DEBUG: "
        case LOG_INFO:
            prefix = "INFO:  "
        case LOG_WARN:
            prefix = "WARN:  "
        case LOG_ERROR:
            prefix = "ERROR: "
    }
    if (l.shouldLog(logLevel)) {fmt.Println(prefix + text)}
}

func (l *Log) shouldLog(logLevel int) bool {
    return logLevel >= config.logLevel
}

//==============================================================================================================================
//     checkErrors - Standard error checking code
//==============================================================================================================================
func checkErrors(err error) bool {
    return err != nil
}

//==============================================================================================================================
//     getMd5Hash - Gets an MD5 hash of the text. This should be safe enough to produce unique deterministic ids
//                  provided our input text is unique.
//==============================================================================================================================
func getMd5Hash(text string) string {
    hasher := md5.New()
    hasher.Write([]byte(text))
    return hex.EncodeToString(hasher.Sum(nil))
}

//==============================================================================================================================
//     Security Subroutines
//==============================================================================================================================
//     get_user_data - Calls the get_ecert and check_role functions and returns the ecert and role for the
//                     name passed.
//==============================================================================================================================
func (t *SimpleChaincode ) get_user_data(stub *shim.ChaincodeStub, name string) ([]byte, int64, error){
    //get the ecert
    ecert, err := t.get_ecert(stub, name);
    if err != nil {
        return nil, -1, errors.New("Could not find ecert for user: "+name)
    }

    //get the role
    role, err := t.check_role(stub,[]string{string(ecert)});
    if err != nil {
        return nil, -1, err
    }

    return ecert, role, nil
}

//==============================================================================================================================
//     check_role - Takes an ecert, decodes it to remove html encoding then parses it and checks the
//                   certificates extensions containing the role before returning the role interger. Returns -1 if it errors
//==============================================================================================================================
func (t *SimpleChaincode ) check_role(stub *shim.ChaincodeStub, args []string) (int64, error) {                                                                                            
    ECertSubjectRole := asn1.ObjectIdentifier{2, 1, 3, 4, 5, 6, 7}                                                                                                                        

    //make % etc normal
    decodedCert, err := url.QueryUnescape(args[0]);
    if err != nil { return -1, errors.New("Could not decode certificate") }

    //make plain text
    pem, _ := pem.Decode([]byte(decodedCert))

    //extract certificate from argument
    x509Cert, err := x509.ParseCertificate(pem.Bytes);
    if err != nil {
        return -1, errors.New("Couldn't parse certificate")
    }

    //get role out of certificate and return it
    var role int64
    for _, ext := range x509Cert.Extensions {
        if reflect.DeepEqual(ext.Id, ECertSubjectRole) {
            role, err = strconv.ParseInt(string(ext.Value), 10, len(ext.Value)*8)   
            if err != nil {
                return -1, errors.New("Failed parsing role: " + err.Error())
            }
            break
        }
    }

    return role, nil
}

//==============================================================================================================================
//     get_user - Takes an ecert, decodes it to remove html encoding then parses it and gets the
//                 common name and returns it
//==============================================================================================================================
func (t *SimpleChaincode ) get_user(stub *shim.ChaincodeStub, encodedCert string) (string, error) {
    //make % etc normal 
    decodedCert, err := url.QueryUnescape(encodedCert);
    if err != nil {
        return "", errors.New("Could not decode certificate")
    }

    //make plain text
    pem, _ := pem.Decode([]byte(decodedCert))
    x509Cert, err := x509.ParseCertificate(pem.Bytes);
    if err != nil {
        return "", errors.New("Couldn't parse certificate")
    }

    //return the user from the certificate
    return x509Cert.Subject.CommonName, nil
}

//==============================================================================================================================
//     get_ecert - Takes the name passed and calls out to the REST API for HyperLedger to retrieve the ecert
//                 for that user. Returns the ecert as retrived including html encoding.
//==============================================================================================================================
func (t *SimpleChaincode ) get_ecert(stub *shim.ChaincodeStub, name string) ([]byte, error) {    
    var cert ECertResponse

    //call out to the hyperLedger rest api to get the ecert of the user with that name
    response, err := http.Get("BLC_API_URL/registrar/"+name+"/ecert")
    if err != nil {
        return nil, errors.New("Could not get ecert")
    }

    //use the defer construct to close the stream after the method completes
    defer response.Body.Close()

    //read the response from the http callout into the variable contents
    contents, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return nil, errors.New("Could not read body")
    }

    //unmarshall the contents of the certificate
    err = json.Unmarshal(contents, &cert)
    if err != nil {
        return nil, errors.New("ECert not found for user: "+name)
    }

    return []byte(string(cert.OK)), nil
}

//==============================================================================================================================
//     Unit Tests
//==============================================================================================================================

func (t *SimpleChaincode ) testAccountCreateSuccess(stub *shim.ChaincodeStub, accountID string) []string {
    var responses []string

    var account Account
    account.ID = accountID
    err := account.create(stub)
    if checkErrors(err) {
        responses = append(responses, "COMPLETE: Account created without errors")
    } else {
        responses = append(responses, "FAIL: call to create account failed")
    }
    return responses
}

