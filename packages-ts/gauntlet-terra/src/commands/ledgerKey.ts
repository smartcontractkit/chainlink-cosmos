import { Key, SimplePublicKey } from '@terra-money/terra.js'
import LedgerTerraConnector, {ERROR_CODE} from '@terra-money/ledger-terra-js'
import TransportNodeHid from "@ledgerhq/hw-transport-node-hid"
import { logger } from '@chainlink/gauntlet-core/dist/utils'


export class LedgerKey extends Key {
    private path: Array<number>
    private ledgerConnector: LedgerTerraConnector

    constructor(path: Array<number>) {
        super()
        this.path = path
    }

    public async initialize() {
        const transport = await TransportNodeHid.create()
        const ledgerConnector = new LedgerTerraConnector(transport)
        await ledgerConnector.initialize()

        const response = await ledgerConnector.getPublicKey(this.path)
        if (response.return_code !== ERROR_CODE.NoError) {
          logger.error(`LedgerKey init failed: ${response.error_message}. Is Ledger unlocked and Terra app open?`)
          throw new Error(response.error_message)
        }

        this.publicKey = new SimplePublicKey(Buffer.from(response.compressed_pk.data).toString('base64'))
        this.ledgerConnector = ledgerConnector
    }

    public static async create(path: string): Promise<LedgerKey> {
        const pathArr = this.pathStringToArray(path)
        const ledgerKey = new LedgerKey(pathArr)
        await ledgerKey.initialize()

        return ledgerKey
    }

    private static pathStringToArray(path: string): Array<number> {
        return path.split(',').map(item => parseInt(item))
    }

    public async sign(payload: Buffer): Promise<Buffer> {
        try { 
            if (!this.publicKey) {
                await this.initialize()
            }

            const response = await this.ledgerConnector.sign(this.path, payload)
            if (response.return_code !== ERROR_CODE.NoError) {
                throw new Error(response.error_message)
            }

            return Buffer.from(response.signature.data)
        } catch (e) {
            logger.error(`LedgerKey sign failed: ${e.message}. Is Ledger unlocked and Terra app open?`)
            throw e
        }
    }
}