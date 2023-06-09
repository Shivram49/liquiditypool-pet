/*
 * Copyright IBM Corp. All Rights Reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

'use strict';

const { Gateway, Wallets } = require('fabric-network');
const path = require('path');
const { buildCCPOrg1, buildCCPOrg2, buildWallet} = require('../../test-application/javascript/AppUtil.js');

const myChannel = 'mychannel';
const myChaincodeName = 'auction';

async function createCommonWallet(ccp,wallet,user,walletId,energyUnits,price) {
	try {

		const gateway = new Gateway();

		// Connect using Discovery enabled
		await gateway.connect(ccp,
			{ wallet: wallet, identity: user, discovery: { enabled: true, asLocalhost: true } });

		const network = await gateway.getNetwork(myChannel);
		const contract = network.getContract(myChaincodeName);

		let statefulTxn = contract.createTransaction('CreateCommonWallet');

		console.log('\n--> Submit Transaction: Propose a new auction');
		await statefulTxn.submit(walletId,energyUnits,price);
		console.log('*** Result: committed');

		console.log('\n--> Evaluate Transaction: query the auction that was just created');
		let result = await contract.evaluateTransaction('QueryWalletById',walletId);
		console.log('*** Result: Wallet: ' + result);

		gateway.disconnect();
	} catch (error) {
		console.error(`******** FAILED to submit the updated wallet: ${error}`);
	}
}

async function main() {
	try {

		if (process.argv[2] === undefined || process.argv[3] === undefined ||
            process.argv[4] === undefined || process.argv[5] === undefined) {
			console.log('Usage: node createCommonWallet.js org userID walletId item');
			process.exit(1);
		}

		const org = process.argv[2];
		const user = process.argv[3];
		const walletId = process.argv[4];
		const energyUnits = process.argv[5];
		const price = process.argv[6];

		if (org === 'Org1' || org === 'org1') {
			const ccp = buildCCPOrg1();
			const walletPath = path.join(__dirname, 'wallet/org1');
			const wallet = await buildWallet(Wallets, walletPath);
			await createCommonWallet(ccp,wallet,user,walletId,energyUnits,price);
		}
		else if (org === 'Org2' || org === 'org2') {
			const ccp = buildCCPOrg2();
			const walletPath = path.join(__dirname, 'wallet/org2');
			const wallet = await buildWallet(Wallets, walletPath);
			await createCommonWallet(ccp,wallet,user,walletId,energyUnits,price);
		}  else {
			console.log('Usage: node createCommonWallet.js org userID walletId item');
			console.log('Org must be Org1 or Org2');
		}
	} catch (error) {
		console.error(`******** FAILED to run the application: ${error}`);
	}
}


main();


