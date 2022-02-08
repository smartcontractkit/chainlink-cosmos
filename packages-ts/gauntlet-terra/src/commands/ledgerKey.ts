import { Key, SimplePublicKey, SignatureV2, SignDoc, SignerInfo, ModeInfo } from '@terra-money/terra.js'
import { SignMode } from '@terra-money/terra.proto/cosmos/tx/signing/v1beta1/signing'
import LedgerTerraConnector, {ERROR_CODE} from '@terra-money/ledger-terra-js'
import TransportNodeHid from "@ledgerhq/hw-transport-node-hid"
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { signatureImport } from 'secp256k1';


export class LedgerKey extends Key {
    private path: Array<number>

    private constructor(path: Array<number>) {
        super()
        this.path = path
    }

    public async initialize() {
        const {ledgerConnector, terminateConnection} = await this.connectToLedger()

        const response = await ledgerConnector.getPublicKey(this.path)
        if (response.return_code !== ERROR_CODE.NoError) {
            logger.error(`LedgerKey init failed: ${response.error_message}. Is Ledger unlocked and Terra app open?`)
            throw new Error(response.error_message)
        }

        this.publicKey = new SimplePublicKey(Buffer.from(response.compressed_pk.data).toString('base64'))
        await terminateConnection()
    }

    public static async create(path: string): Promise<LedgerKey> {
        const pathArr = this.pathStringToArray(path)
        const ledgerKey = new LedgerKey(pathArr)
        await ledgerKey.initialize()

        return ledgerKey
    }

    private static pathStringToArray(path: string): Array<number> {
        return path.split('\'/').map(item => parseInt(item))
    }

    public async createSignature(signDoc: SignDoc): Promise<SignatureV2> {
        if (!this.publicKey) {
          throw new Error(
            'Signature could not be created: Key instance missing publicKey.'
          );
        }
    
        // backup for restore
        const signerInfos = signDoc.auth_info.signer_infos;
        signDoc.auth_info.signer_infos = [
          new SignerInfo(
            this.publicKey,
            signDoc.sequence,
            new ModeInfo(new ModeInfo.Single(SignMode.SIGN_MODE_DIRECT))
          ),
        ];
    
        const signDocBuffer = Buffer.from(signDoc.toAminoJSON())
        const sigBytes = (await this.sign(signDocBuffer)).toString('base64');
    
        // restore signDoc to origin
        signDoc.auth_info.signer_infos = signerInfos;
    
        return new SignatureV2(
          this.publicKey,
          new SignatureV2.Descriptor(
            new SignatureV2.Descriptor.Single(SignMode.SIGN_MODE_DIRECT, sigBytes)
          ),
          signDoc.sequence
        );
      }

    public async sign(payload: Buffer): Promise<Buffer> {
        const {ledgerConnector, terminateConnection} = await this.connectToLedger()
        try { 
            logger.info('Approve tx on your Ledger device.')
            const response = await ledgerConnector.sign(this.path, payload)
            if (response.return_code !== ERROR_CODE.NoError) {
                throw new Error(response.error_message)
            }
   
            const signature = signatureImport(Buffer.from(response.signature as any))
            return Buffer.from(signature)
        } catch (e) {
            logger.error(`LedgerKey sign failed: ${e.message}. Is Ledger unlocked and Terra app open?`)
            throw e
        } finally {
            await terminateConnection()
        }
    }

    private async connectToLedger(){
        const transport = await TransportNodeHid.create()
        const ledgerConnector = new LedgerTerraConnector(transport)
        await ledgerConnector.initialize()

        return {
            ledgerConnector, 
            terminateConnection: transport.close.bind(transport) 
        }
    }
}