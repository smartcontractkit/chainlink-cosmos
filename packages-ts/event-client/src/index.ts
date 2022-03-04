import { WebSocketClient } from '@terra-money/terra.js';
import { TxLog, Int } from '@terra-money/terra.js';

export class Chainlink {
    private _wsClient: WebSocketClient;

    constructor(
        readonly url: string, //TODO or pass own client?
    ){
        this._wsClient = new WebSocketClient(url);
    }

    public start() {
        this._wsClient.start()
    }

    public destroy() {
        this._wsClient.destroy()
    }

    public onRound(contract: string, callback: (round: Round) => void) {
        this._wsClient.subscribeTx({
            'wasm-new_transmission.contract_address': contract,
            },
            async data => {
                const txRes = data.value.TxResult.result;
                Chainlink.parseLog(txRes.log).forEach(callback);
            }
        );
    }

    public static parseLog(log: string): Round[] {
        let logs: TxLog[] = JSON.parse(log).map(TxLog.fromData);
        return logs.map(l=>l.eventsByType['wasm-new_transmission']).map(this.roundFromAttributes);
    }

    private static roundFromAttributes(attrs: {[k:string]:string[]}): Round {
        let onlyAttr = (key:string): string => {
            let vals = attrs[key];
            if (vals.length != 1) {
                return null;
            }
            return vals[0];
        }
        return {
            answer: new Int(onlyAttr("answer")),
            round: parseInt(onlyAttr("round")),
            epoch: parseInt(onlyAttr("epoch")),
        }
    }
}

export interface Round {
    answer: Int;
    round: number;
    epoch: number;
}