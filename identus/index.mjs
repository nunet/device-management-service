import { Apollo, Domain } from '@hyperledger/identus-sdk';
import crypto from 'node:crypto';

async function run() {
    console.log('--- Identus SDK Demo (ESM) ---');

    try {
        const apollo = new Apollo();

        // Generate a random seed (64 bytes)
        const seed = new Int8Array(crypto.randomBytes(64));
        console.log('Generated random seed (Int8Array). Length:', seed.length);

        // Generate a new private key (secp256k1)
        console.log('Generating private key...');
        const privateKey = apollo.createPrivateKey({
            type: 'EC',
            curve: 'secp256k1',
            seed: seed
        });

        console.log('Key generated successfully.');

        // Export as JWK
        console.log('Exporting key to JWK...');


        if (privateKey.to && typeof privateKey.to.JWK === 'function') {
            const jwk = privateKey.to.JWK();
            console.log('JWK:', JSON.stringify(jwk, null, 2));
            console.log('Private Key (JSON):', JSON.stringify(privateKey, (key, value) => {
                if (value instanceof Map) {
                    return Object.fromEntries(value);
                }
                if (value instanceof Uint8Array || value instanceof Int8Array) {
                    return Buffer.from(value).toString('hex');
                }
                if (typeof value === 'bigint') {
                    return value.toString();
                }
                return value;
            }, 2));
        } else {
            console.log('Could not find JWK export method on privateKey.');
            console.log('PrivateKey object:', privateKey);
        }

    } catch (error) {
        console.error('Error running demo:', error);
        if (error.stack) console.error(error.stack);
    }
}

run();
