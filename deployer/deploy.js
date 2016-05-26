'use strict';
(function(){
     // Step 1 ==================================
    var Ibc1 = require('ibm-blockchain-js');
    var ibc = new Ibc1(/*logger*/);             //you can pass a logger such as winston here - optional
    var chaincode = {};

    // ==================================
    // configure ibc-js sdk
    // ==================================
    var options =   {
        network:{
            peers:   [{
	            "api_host": "f6b6484c-f69e-4bb0-9bfa-368ebe1ed70f_vp1-api.blockchain.ibm.com",
	            "api_port_tls": 443,
	            "api_port": 80,
            	"id": "f6b6484c-f69e-4bb0-9bfa-368ebe1ed70f_vp1"
            }],
            users:  [{
				"enrollId": "dashboarduser_type0_3341babe6c",
            	"enrollSecret": "d84f2ede5d"
            }],
            options: {                          //this is optional
                quiet: true, 
                timeout: 60000
            }
        },
        chaincode:{
            zip_url: 'https://github.com/alansfyeung/blockhack16micromarket/archive/master.zip',
            unzip_dir: 'blockhack16micromarket-master/',
            git_url: 'https://github.com/alansfyeung/blockhack16micromarket/'
        }
    };

    // Step 2 ==================================
    ibc.load(options, cb_ready);

    // Step 3 ==================================
    function cb_ready(err, cc){                             //response has chaincode functions
        // app1.setup(ibc, cc);
        // app2.setup(ibc, cc);

    // Step 4 ==================================
        if(cc.details.deployed_name === ""){                //decide if I need to deploy or not
            cc.deploy('init', ['99'], null, cb_deployed);
        }
        else{
            console.log('chaincode summary file indicates chaincode has been previously deployed');
            cb_deployed();
        }
    }

    // Step 5 ==================================
    function cb_deployed(err){
        console.log('sdk has deployed code and waited');
        //chaincode.query.read(['a']);
    }


}());
