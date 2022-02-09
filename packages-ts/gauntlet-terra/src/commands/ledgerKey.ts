import { Key, SimplePublicKey, SignatureV2, SignDoc, SignerInfo, ModeInfo } from '@terra-money/terra.js'
import LedgerTerraConnector, {ERROR_CODE, CommonResponse} from '@terra-money/ledger-terra-js'
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
        this.checkForErrors(response)

        this.publicKey = new SimplePublicKey(Buffer.from(response.compressed_pk.data).toString('base64'))
        await terminateConnection()
    }

    public static async create(path: string): Promise<LedgerKey> {
        const pathArr = this.pathStringToArray(path)
        const ledgerKey = new LedgerKey(pathArr)
        await ledgerKey.initialize()

        return ledgerKey
    }

    public async sign(payload: Buffer): Promise<Buffer> {
        const {ledgerConnector, terminateConnection} = await this.connectToLedger()
        try { 
            logger.info('Approve tx on your Ledger device.')
            const response = await ledgerConnector.sign(this.path, payload)
            this.checkForErrors(response)
   
            const signature = signatureImport(Buffer.from(response.signature.data))
            return Buffer.from(signature)
        } catch (e) {
            logger.error(`LedgerKey sign failed: ${e.message}`)
            throw e
        } finally {
            await terminateConnection()
        }
    }

    private async connectToLedger(){
        const transport = await TransportNodeHid.create()
        const ledgerConnector = new LedgerTerraConnector(transport)
        const response = await ledgerConnector.initialize()
        this.checkForErrors(response)

        return {
            ledgerConnector, 
            terminateConnection: transport.close.bind(transport) 
        }
    }

    private static pathStringToArray(path: string): Array<number> {
        return path.split('\'/').map(item => parseInt(item))
    }

    private checkForErrors (response: CommonResponse) {
        if (!response)
            return

        const { 
            error_message: ledgerError, 
            return_code: returnCode,
            device_locked: isLocked
        } = response

        if (returnCode === ERROR_CODE.NoError)
            return

        if (isLocked) {
            throw new Error('Is Ledger unlocked and Terra app open?')
        }
        throw new Error(ledgerError)
    }
}