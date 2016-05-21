package main

import (
    "errors"
    "fmt"
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

const   LOG_DEBUG           =  0
const   LOG_INFO            =  1
const   LOG_WARN            =  2
const   LOG_ERROR           =  3

const   CURRENT_LOG_LEVEL   = LOG_DEBUG

const   PROPERTY_PREFIX     = "property:"
const   ACCOUNT_PREFIX      = "account:"


//==============================================================================================================================
//     Structure Definitions 
//==============================================================================================================================
//    Chaincode - A blank struct for use with Shim (A HyperLedger included go file used for get/put state
//                and other HyperLedger functions)
//==============================================================================================================================
type Chaincode struct {
}

//==============================================================================================================================
//    Log - A blank struct to use as simple stateless logger
//==============================================================================================================================
type Log struct {
}
var log Log

//==============================================================================================================================
//    Property - Defines the structure for a property object. JSON on right tells it what JSON fields to map to
//              that element when reading a JSON object into the struct e.g. JSON propetyId -> Struct ID.
//==============================================================================================================================
type Property struct {
  //details
    ID              string      `json:"propertyId"`
    AddressLine     string      `json:"addressLine"`
    Suburb          string      `json:"suburb"`
    State           string      `json:"state"`

  //comparison
    PostCode        string      `json:"postcode"`
    Bedrooms        int         `json:"bedrooms"`
    Bathrooms       int         `json:"bathrooms"`
    Squares         int         `json:"squares"`
    Size            int         `json:"size"`
    Zoning          int         `json:"zoning"`

  //financials
    Rented          bool        `json:"rented"`
    Rent            int         `json:"rent"`
    LastPayment     int         `json:"lastPaymentDate"`
    Valuation       int         `json:"valution"`
    ValuationDate   int         `json:"valuationDate"`
    
  //info
    ManagedBy       string      `json:"managedBy"`
    Issuer          string      `json:"issuer"`
    Units           int         `json:"units"`
    Status          int         `json:"status"`
}

//==============================================================================================================================
//    Account - Defines the structure for a share in a property.
//==============================================================================================================================
type Account struct {
    ID              string      `json:"accountId"`
    Cash            float64     `json:"cash"`
    Status          int         `json:"status"`
    Holdings        []Holding   `json:"holdings"`
}

//==============================================================================================================================
//    Holding - Defines the structure for a share in a property.
//==============================================================================================================================
type Holding struct {
    Entity          string      `json:"entity"`
    Units           int         `json:"units"`
}

//==============================================================================================================================
//    ECertResponse - Struct for storing the JSON response of retrieving an ECert. JSON OK -> Struct OK
//==============================================================================================================================
type ECertResponse struct {
    OK string `json:"OK"`
}

//==============================================================================================================================
//     Chaincode Lifecycle Functions
//=================================================================================================================================
//     Main - main - Starts up the chaincode
//=================================================================================================================================
func main() {
    err := shim.Start(new(Chaincode))
    if checkErrors(err){fmt.Printf("Error starting Chaincode: %s", err)}
}

//==============================================================================================================================
//    Init Function - Called when the user deploys the chaincode                                                                    
//==============================================================================================================================
func (t *Chaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    //authenticate the user
    //caller_ecert, caller_role, err := t.get_user_data(stub, args[0])
    //if checkErrors(err){return nil, err}
    
    log.debug("Create an issuers account for Cardy and Co first up.")
    var issuerAccount Account
    issuerAccount.ID = "cardy"
    issuerAccount.Cash = 100000
    issuerAccount.Status = ACCOUNT_STATE_ACTIVE
    err := issuerAccount.save(stub)
    if checkErrors(err){fmt.Printf("Error starting Chaincode: %s", err)}

    return nil, err
}

//=================================================================================================================================    
//    Query - Called on chaincode query. Takes a function name passed and calls that function. Passes the
//          initial arguments passed are passed on to the called function.
//=================================================================================================================================    
func (t *Chaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    //authenticate the user
    //caller_ecert, caller_role, err := t.get_user_data(stub, args[0])
    //if checkErrors(err){return nil, err}

    A := args[0]

	// Get the state from the ledger
	Avalbytes, err := stub.GetState(A)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + A + "\"}"
		return nil, errors.New(jsonResp)
	}

    if Avalbytes == nil {
		jsonResp := "{\"Error\":\"Nil amount for " + A + "\"}"
		return nil, errors.New(jsonResp)
	}

	return Avalbytes, nil
}

//==============================================================================================================================
//    Invoke - Called on chaincode invoke. Takes a function name passed and calls that function. Converts some
//             initial arguments passed to other things for use in the called function e.g. name -> ecert
//==============================================================================================================================
func (t *Chaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
    //authenticate the user
    //caller_ecert, caller_role, err := t.get_user_data(stub, args[0])
    //if checkErrors(err){return nil, err}

    return nil, nil
}

//==============================================================================================================================
//     Business Logic Methods
//==============================================================================================================================
//     issueProperty - Issue a property for trading on the block chain. The property's units will automatically be assigned
//                     to the account of the issuer
//==============================================================================================================================
func (t *Chaincode) issueProperty(stub *shim.ChaincodeStub, args []string) error {

    //    0
    //    json
    //    {
    //        id:"",
    //        issuer:""
    //    }
    //

    log.debug("check issueProperty args")
    if len(args) != 1 {return errors.New("Incorrect number of arguments. Expecting property json")}

    log.debug("unmarshalling " + args[0])
    property, err := unmarshalProperty([]byte(args[0]))
    if checkErrors(err){return err}
    
    log.debug("creating the property in the blockchain")
    err = property.create(stub)
    if checkErrors(err){return err}
    
    log.debug("get the account for the issuer " + property.Issuer)
    issuerAccount, err := getAccount(stub, property.Issuer)
    if checkErrors(err){return err}

    log.debug("Set the issuer to be the owner of all units")
    var issuerHolding Holding
    issuerHolding.Entity = property.ID
    issuerHolding.Units = property.Units
    issuerAccount.Holdings = append(issuerAccount.Holdings, issuerHolding)

    log.debug("save the issuer's account")
    err = issuerAccount.save(stub)
    if checkErrors(err){return err}

    log.debug("now create an account for the property with an initial view of the holdings")
    var propertyAccount Account
    propertyAccount.ID = property.ID
    propertyAccount.Cash = 0

    var propertyHolding Holding
    propertyHolding.Entity = property.Issuer
    propertyHolding.Units = property.Units
    propertyAccount.Holdings = append(propertyAccount.Holdings, propertyHolding)
    
    err = propertyAccount.create(stub)
    if checkErrors(err){return err}
    
    log.info("Issued property " + property.ID)

    return nil
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

func (object *Property) create(stub *shim.ChaincodeStub) error {
    err := object.validate()
    if checkErrors(err){return err}

    if object.ID != "" {return errors.New("Can't create property with ID already assigned")}

    existing, err := getProperty(stub, object.ID)
    if checkErrors(err){return err}
    if existing.ID != "" {return errors.New("A property with this ID already exists")}

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

func (object *Account) create(stub *shim.ChaincodeStub) error {
    err := object.validate()
    if checkErrors(err){return err}

    if object.ID == "" {return errors.New("An account needs to be assigned to an owner")}

    existing, err := getAccount(stub, object.ID)
    if checkErrors(err){return err}
    if existing.ID != "" {return errors.New("This account already exists")}

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

func (object *Account) validate() error {
    return nil
}

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

func (object *Property) marshal() ([]byte, error) {
    bytes, err := json.Marshal(object)
    if checkErrors(err){return nil, errors.New("Error marshalling property")}
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

func (object *Account) marshal() ([]byte, error) {
    bytes, err := json.Marshal(object)
    if checkErrors(err){return nil, errors.New("Error marshalling account")}
    return bytes, nil
}

//==============================================================================================================================
//     Utility Subroutines
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
    return logLevel >= CURRENT_LOG_LEVEL
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
func (t *Chaincode) get_user_data(stub *shim.ChaincodeStub, name string) ([]byte, int64, error){
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
func (t *Chaincode) check_role(stub *shim.ChaincodeStub, args []string) (int64, error) {                                                                                            
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
func (t *Chaincode) get_user(stub *shim.ChaincodeStub, encodedCert string) (string, error) {
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
func (t *Chaincode) get_ecert(stub *shim.ChaincodeStub, name string) ([]byte, error) {    
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
