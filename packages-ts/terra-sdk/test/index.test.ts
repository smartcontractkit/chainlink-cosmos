import { OCR2Feed, Round } from '../src'
import { Int, WebSocketClient } from '@terra-money/terra.js'
import exp = require('constants')

describe('OCR2Feed', () => {
  it('parseLog', () => {
    let got = OCR2Feed.parseLog(
      '[{"events":[{"type":"execute_contract","attributes":[{"key":"sender","value":"terra167h3sh8c4pgs8grxz24pam2x764flydv3h9pd8"},{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"}]},{"type":"from_contract","attributes":[{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"},{"key":"method","value":"transmit"},{"key":"method","value":"transmit"}]},{"type":"message","attributes":[{"key":"action","value":"/terra.wasm.v1beta1.MsgExecuteContract"},{"key":"module","value":"wasm"},{"key":"sender","value":"terra167h3sh8c4pgs8grxz24pam2x764flydv3h9pd8"}]},{"type":"wasm","attributes":[{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"},{"key":"method","value":"transmit"},{"key":"method","value":"transmit"}]},{"type":"wasm-new_transmission","attributes":[{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"},{"key":"aggregator_round_id","value":"25016"},{"key":"answer","value":"9099065386"},{"key":"transmitter","value":"terra167h3sh8c4pgs8grxz24pam2x764flydv3h9pd8"},{"key":"observations_timestamp","value":"1646415623"},{"key":"observers","value":"0d0c020805010409060b0e0003070000000000000000000000000000000000"},{"key":"juels_per_fee_coin","value":"156135087"},{"key":"config_digest","value":"000283aa7440f0e70a44e5e122a6a4112b4670c9a72f8746eeda1cc93c296b1a"},{"key":"epoch","value":"26787"},{"key":"round","value":"6"},{"key":"reimbursement","value":"460650"},{"key":"observations","value":"9094516578"},{"key":"observations","value":"9094516578"},{"key":"observations","value":"9094516578"},{"key":"observations","value":"9096974971"},{"key":"observations","value":"9097065371"},{"key":"observations","value":"9098065379"},{"key":"observations","value":"9098065379"},{"key":"observations","value":"9099065386"},{"key":"observations","value":"9099065386"},{"key":"observations","value":"9099975292"},{"key":"observations","value":"9100065393"},{"key":"observations","value":"9100791011"},{"key":"observations","value":"9100823064"},{"key":"observations","value":"9102065407"}]},{"type":"wasm-transmitted","attributes":[{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"},{"key":"config_digest","value":"000283aa7440f0e70a44e5e122a6a4112b4670c9a72f8746eeda1cc93c296b1a"},{"key":"epoch","value":"26787"}]}]}]',
    )
    expect(got).toEqual([
      {
        contract: 'terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c',
        answer: new Int(9099065386),
        roundId: 6,
        epoch: 26787,
        aggregatorRoundId: 25016,
        observationsTS: new Date('2022-03-04T17:40:23.000Z'),
      },
    ])
  })
  it('parseLog multi big and negative', () => {
    let got = OCR2Feed.parseLog(
      '[' +
        '{"events":[{"type":"wasm-new_transmission","attributes":[{"key":"contract_address","value":"foo"},{"key":"answer","value":"99999999999999999999999999999999"},{"key":"epoch","value":"13"},{"key":"round","value":"42"},{"key":"observations_timestamp","value":"1646417623"},{"key":"aggregator_round_id","value":"6688"}]}]},' +
        '{"events":[{"type":"wasm-new_transmission","attributes":[{"key":"answer","value":"-1234567890"},{"key":"epoch","value":"42"},{"key":"round","value":"13"},{"key":"aggregator_round_id","value":"99"},{"key":"observations_timestamp","value":"1646425623"},{"key":"contract_address","value":"bar"}]}]}' +
        ']',
    )
    expect(got).toEqual([
      {
        contract: 'foo',
        answer: new Int('99999999999999999999999999999999'),
        roundId: 42,
        epoch: 13,
        aggregatorRoundId: 6688,
        observationsTS: new Date('2022-03-04T18:13:43.000Z'),
      },
      {
        contract: 'bar',
        answer: new Int(-1234567890),
        roundId: 13,
        epoch: 42,
        aggregatorRoundId: 99,
        observationsTS: new Date('2022-03-04T20:27:03.000Z'),
      },
    ])
  })
  it('parseLog missing fields', () => {
    let got = OCR2Feed.parseLog(
      '[{"events":[{"type":"execute_contract","attributes":[{"key":"sender","value":"terra167h3sh8c4pgs8grxz24pam2x764flydv3h9pd8"},{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"}]},{"type":"from_contract","attributes":[{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"},{"key":"method","value":"transmit"},{"key":"method","value":"transmit"}]},{"type":"message","attributes":[{"key":"action","value":"/terra.wasm.v1beta1.MsgExecuteContract"},{"key":"module","value":"wasm"},{"key":"sender","value":"terra167h3sh8c4pgs8grxz24pam2x764flydv3h9pd8"}]},{"type":"wasm","attributes":[{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"},{"key":"method","value":"transmit"},{"key":"method","value":"transmit"}]},{"type":"wasm-new_transmission","attributes":[{"key":"transmitter","value":"terra167h3sh8c4pgs8grxz24pam2x764flydv3h9pd8"},{"key":"observers","value":"0d0c020805010409060b0e0003070000000000000000000000000000000000"},{"key":"juels_per_fee_coin","value":"156135087"},{"key":"config_digest","value":"000283aa7440f0e70a44e5e122a6a4112b4670c9a72f8746eeda1cc93c296b1a"},{"key":"reimbursement","value":"460650"},{"key":"observations","value":"9094516578"},{"key":"observations","value":"9094516578"},{"key":"observations","value":"9094516578"}]},{"type":"wasm-transmitted","attributes":[{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"},{"key":"config_digest","value":"000283aa7440f0e70a44e5e122a6a4112b4670c9a72f8746eeda1cc93c296b1a"}]}]}]',
    )
    expect(got).toEqual([
      {
        contract: null,
        answer: null,
        roundId: null,
        epoch: null,
        aggregatorRoundId: null,
        observationsTS: null,
      },
    ])
  })
  it('parseLog missing event', () => {
    let got = OCR2Feed.parseLog(
      '[{"events":[{"type":"execute_contract","attributes":[{"key":"sender","value":"terra167h3sh8c4pgs8grxz24pam2x764flydv3h9pd8"},{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"}]},{"type":"from_contract","attributes":[{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"},{"key":"method","value":"transmit"},{"key":"method","value":"transmit"}]},{"type":"message","attributes":[{"key":"action","value":"/terra.wasm.v1beta1.MsgExecuteContract"},{"key":"module","value":"wasm"},{"key":"sender","value":"terra167h3sh8c4pgs8grxz24pam2x764flydv3h9pd8"}]},{"type":"wasm","attributes":[{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"},{"key":"method","value":"transmit"},{"key":"method","value":"transmit"}]},{"type":"wasm-transmitted","attributes":[{"key":"contract_address","value":"terra1fwhxcdlm7cefu585pv54vgcpacq9daxppa823c"},{"key":"config_digest","value":"000283aa7440f0e70a44e5e122a6a4112b4670c9a72f8746eeda1cc93c296b1a"}]}]}]',
    )
    expect(got).toEqual([])
  })
  let itWS = process.env.WS_URL ? it : it.skip
  itWS(
    'onRound',
    async () => {
      let contract = process.env.OCR2_CONTRACT || 'terra154secslh0htss73hjv0zdmdev0jy7s0a00ehcp' // Columbus-5
      let cl = new OCR2Feed(new WebSocketClient(process.env.WS_URL))
      let resolve
      let promise = new Promise<Round>((r) => (resolve = r))
      cl.onRound(contract, (round) => {
        resolve(round)
        cl.destroy()
      })
      cl.start()
      let got = await promise
      expect(got).toBeDefined()
      expect(got.contract).toEqual(contract)
      expect(got.answer).toBeDefined()
      expect(got.roundId).toBeDefined()
      expect(got.epoch).toBeDefined()
      expect(got.aggregatorRoundId).toBeDefined()
      expect(got.observationsTS).toBeDefined()
    },
    120_000,
  )
})
