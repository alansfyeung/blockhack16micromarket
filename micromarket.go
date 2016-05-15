package main

import (
	"errors"
	"fmt"
	"strconv"
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

const   STATE_PROPOSED      =  0
const   STATE_MANAGED       =  1
const   STATE_RECLAIMED     =  2


//==============================================================================================================================
//	 Structure Definitions 
//==============================================================================================================================
//	Chaincode - A blank struct for use with Shim (A HyperLedger included go file used for get/put state
//				and other HyperLedger functions)
//==============================================================================================================================
type  Chaincode struct {
}

//==============================================================================================================================
//	Property - Defines the structure for a property object. JSON on right tells it what JSON fields to map to
//			  that element when reading a JSON object into the struct e.g. JSON propetyId -> Struct ID.
//==============================================================================================================================
type Property struct {
	ID           string `json:"propertyId"`
	AddressLine1 string `json:"addressLine1"`
	Suburb       string `json:"suburb"`
	State        string `json:"state"`					
	PostCode     string `json:"postcode"`
	Status       int    `json:"status"`
	Rent         int    `json:"rent"`
	Rented       bool   `json:"rented"`
	Shares       int    `json:"shares"`
}

//==============================================================================================================================
//	Share - Defines the structure for a share in a property.
//==============================================================================================================================
type Share struct {
	ID           string `json:"shareId"`
	Balance      string `json:"propertyId"`
}

//==============================================================================================================================
//	Dollar - Defines the structure for an Australian dollar.
//==============================================================================================================================
type Dollar struct {
}

//==============================================================================================================================
//	Cent - Defines the structure for an Australian cent.
//==============================================================================================================================
type Cent struct {
}

//==============================================================================================================================
//	Micron - Defines the structure for an 1/1000 of an Australian cent.
//==============================================================================================================================
type Micron struct {
}

//==============================================================================================================================
//	ECertResponse - Struct for storing the JSON response of retrieving an ECert. JSON OK -> Struct OK
//==============================================================================================================================
type ECertResponse struct {
	OK string `json:"OK"`
}

//==============================================================================================================================
//	 Chaincode Lifecycle Functions
//=================================================================================================================================
//	 Main - main - Starts up the chaincode
//=================================================================================================================================
func main() {
	err := shim.Start(new(Chaincode))
	if err != nil {
        fmt.Printf("Error starting Chaincode: %s", err)
    }
}

//==============================================================================================================================
//	Init Function - Called when the user deploys the chaincode																	
//==============================================================================================================================
func (t *Chaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	return nil, nil
}

//=================================================================================================================================	
//	Query - Called on chaincode query. Takes a function name passed and calls that function. Passes the
//  		initial arguments passed are passed on to the called function.
//=================================================================================================================================	
func (t *Chaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	return nil, nil
}

//==============================================================================================================================
//	Invoke - Called on chaincode invoke. Takes a function name passed and calls that function. Converts some
//		     initial arguments passed to other things for use in the called function e.g. name -> ecert
//==============================================================================================================================
func (t *Chaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	return nil, nil
}

//==============================================================================================================================
//	 Security Subroutines
//==============================================================================================================================
//	 get_user_data - Calls the get_ecert and check_role functions and returns the ecert and role for the
//					 name passed.
//==============================================================================================================================
func (t *Chaincode) get_user_data(stub *shim.ChaincodeStub, name string) ([]byte, int64, error){
    ecert, err := t.get_ecert(stub, name);
    if err != nil {
        return nil, -1, errors.New("Could not find ecert for user: "+name)
    }

    role, err := t.check_role(stub,[]string{string(ecert)});
    if err != nil {
        return nil, -1, err
    }

    return ecert, role, nil
}

//==============================================================================================================================
//	 check_role - Takes an ecert, decodes it to remove html encoding then parses it and checks the
// 				  certificates extensions containing the role before returning the role interger. Returns -1 if it errors
//==============================================================================================================================
func (t *Chaincode) check_role(stub *shim.ChaincodeStub, args []string) (int64, error) {																							
	ECertSubjectRole := asn1.ObjectIdentifier{2, 1, 3, 4, 5, 6, 7}																														
	
	decodedCert, err := url.QueryUnescape(args[0]);    		// make % etc normal //
	
															if err != nil { return -1, errors.New("Could not decode certificate") }
	
	pem, _ := pem.Decode([]byte(decodedCert))           	// Make Plain text   //

	x509Cert, err := x509.ParseCertificate(pem.Bytes);		// Extract Certificate from argument //
														
															if err != nil { return -1, errors.New("Couldn't parse certificate")	}

	var role int64
	for _, ext := range x509Cert.Extensions {				// Get Role out of Certificate and return it //
		if reflect.DeepEqual(ext.Id, ECertSubjectRole) {
			role, err = strconv.ParseInt(string(ext.Value), 10, len(ext.Value)*8)   
                                            			
															if err != nil { return -1, errors.New("Failed parsing role: " + err.Error())	}
			break
		}
	}
	
	return role, nil
}
//==============================================================================================================================
//	 get_user - Takes an ecert, decodes it to remove html encoding then parses it and gets the
// 				common name and returns it
//==============================================================================================================================
func (t *Chaincode) get_user(stub *shim.ChaincodeStub, encodedCert string) (string, error) {
	
	decodedCert, err := url.QueryUnescape(encodedCert);    		// make % etc normal //
	
															if err != nil { return "", errors.New("Could not decode certificate") }
	
	pem, _ := pem.Decode([]byte(decodedCert))           	// Make Plain text   //

	x509Cert, err := x509.ParseCertificate(pem.Bytes);
	
															if err != nil { return "", errors.New("Couldn't parse certificate")	}

	return x509Cert.Subject.CommonName, nil

}

//==============================================================================================================================
//	 get_ecert - Takes the name passed and calls out to the REST API for HyperLedger to retrieve the ecert
//				 for that user. Returns the ecert as retrived including html encoding.
//==============================================================================================================================
func (t *Chaincode) get_ecert(stub *shim.ChaincodeStub, name string) ([]byte, error) {
	
	var cert ECertResponse
	
	response, err := http.Get("BLC_API_URL/registrar/"+name+"/ecert") // Calls out to the HyperLedger REST API to get the ecert of the user with that name
    
															if err != nil { return nil, errors.New("Could not get ecert") }
	
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)			// Read the response from the http callout into the variable contents
	
															if err != nil { return nil, errors.New("Could not read body") }
	
	err = json.Unmarshal(contents, &cert)
	
															if err != nil { return nil, errors.New("ECert not found for user: "+name) }
															
	return []byte(string(cert.OK)), nil
}