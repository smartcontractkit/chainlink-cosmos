import { WebSocketClient } from '@terra-money/terra.js';
import { TxLog, Int } from '@terra-money/terra.js';

export interface Round {
    answer: Int;
    round: number;
    epoch: number;
    aggregatorRoundId: number;
    observationsTS: Date;
}

export class OCR2Feed {
    private _wsClient: WebSocketClient;

    constructor(readonly client: WebSocketClient){
        this._wsClient = client;
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
                OCR2Feed.parseLog(txRes.log).forEach(callback);
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
            if (!vals || vals.length != 1) {
                return null;
            }
            return vals[0];
        }
        let tryInt = (s?:string): number => {
            return parseInt(s) || null;
        }
        let tryBig = (s?:string): Int => {
            if (!s) return null;
            return new Int(s);
        }
        let tryUnixDate = (s?:string): Date => {
            let unixTS = tryInt(s);
            if (!unixTS) return null;
            return new Date(unixTS * 1000);
        }
        return {
            answer: tryBig(onlyAttr("answer")),
            round: tryInt(onlyAttr("round")),
            epoch: tryInt(onlyAttr("epoch")),
            aggregatorRoundId: tryInt(onlyAttr("aggregator_round_id")),
            observationsTS: tryUnixDate(onlyAttr("observations_timestamp")),
        }
    }
}
