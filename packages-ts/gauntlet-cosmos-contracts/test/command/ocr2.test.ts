import { GasPrice } from '@cosmjs/stargate'
import { CosmWasmClient, ExecuteResult, SigningCosmWasmClient } from '@cosmjs/cosmwasm-stargate'
import { Ocr2QueryClient } from '../../codegen/Ocr2.client'
import { CW20BaseQueryClient, CW20BaseClient } from '../../codegen/CW20Base.client'
import {
  endWasmd,
  CMD_FLAGS,
  initWasmd,
  NODE_URL,
  TIMEOUT,
  deployAC,
  deployLink,
  toAddr,
  DEFAULT_GAS_PRICE,
  deployOCR2,
} from '../utils'
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing'

import SetBilling from '../../src/commands/contracts/ocr2/setBilling'
import BeginProposal from '../../src/commands/contracts/ocr2/proposal/beginProposal'
import ProposeConfig from '../../src/commands/contracts/ocr2/proposeConfig'
import ProposeOffchainConfig from '../../src/commands/contracts/ocr2/proposeOffchainConfig'
import AcceptProposal from '../../src/commands/contracts/ocr2/proposal/acceptProposal'
import FinalizeProposal from '../../src/commands/contracts/ocr2/proposal/finalizeProposal'
import WithdrawFunds from '../../src/commands/contracts/ocr2/withdrawFunds'
import WithdrawPayment from '../../src/commands/contracts/ocr2/withdrawPayment'
import ProposeConfigClose from '../../src/commands/contracts/ocr2/close/proposeConfig'
import ProposeOffchainConfigClose from '../../src/commands/contracts/ocr2/close/proposeOffchainConfig'
import AcceptProposalClose from '../../src/commands/contracts/ocr2/close/acceptProposal'

// Note: signers are not bech32 addresses
const SIGNERS = [
  'b90e50daf82024624549e7708199dd05b6de8e10d6df62cd27581c65e5096b24',
  '5f786249d2b5018ce084442fb3dce6180087ad5ca6ae3fd0487402200cb4177f',
  '1da3c8ac817762f6f36efdbc14c66be357ed9f4bcfa7f27eca9f0aaa2618fa46',
  '967f37d14afeb8d0e5935e173265f59b55afcbc04b9feba3fcb2e0aa3c964fa7',
]

const TRANSMITTERS = [
  'wasm1jcamds37x233wtpzj5k3f5f7gszaahjghqvdxe',
  'wasm1a2dr8rjv2ft2z6g4spe3534hxn9hu50sutxqw5',
  'wasm1p2udqrgwtyh2t284829cwzzfqc2kjaw4gvhqza',
  'wasm1sya395n2l8z74efxmfw7fxfxseke3wpypqd68w',
]

const PAYEES = [
  // 'wasm1vz2r63kdkcdj25nexg43axk2hrvvrq55rlzj03',
  // first payee is the deployer
  'wasm1lsagfzrm4gz28he4wunt63sts5xzmczwda8vl6',
  'wasm1jv7p2hml6gsu2k0ynx0uwtqrcyjqv8xv6k3ua6',
  'wasm1a6vse48zvfge8yw53m5u62rw7xw3qmf2x6jgcu',
  'wasm18vga7wnhks6jaqhzjf4hlnqdmmk6x9sgf2ztzv',
]

const OFF_CHAIN_CONFIG = {
  deltaProgressNanoseconds: 8000000000,
  deltaResendNanoseconds: 30000000000,
  deltaRoundNanoseconds: 3000000000,
  deltaGraceNanoseconds: 500000000,
  deltaStageNanoseconds: 20000000000,
  rMax: 5,
  s: [1, 2],
  offchainPublicKeys: [
    'af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852090',
    'af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852091',
    'af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852092',
    'af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852093',
  ],
  peerIds: SIGNERS,
  reportingPluginConfig: {
    alphaReportInfinite: false,
    alphaReportPpb: 0,
    alphaAcceptInfinite: false,
    alphaAcceptPpb: 0,
    deltaCNanoseconds: 0,
  },
  maxDurationQueryNanoseconds: 2000000000,
  maxDurationObservationNanoseconds: 1000000000,
  maxDurationReportNanoseconds: 200000000,
  maxDurationShouldAcceptFinalizedReportNanoseconds: 200000000,
  maxDurationShouldTransmitAcceptedReportNanoseconds: 200000000,
  configPublicKeys: [
    'af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852094',
    'af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852095',
    'af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852096',
    'af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852097',
  ],
}

const EXPECTED_OFF_CHAIN_CONFIG_DIGEST =
  'CICg2eYdEIDYjuFvGIC8wZYLIIDKte4BKICQ38BKMAU6AgECQiCvQAAE+l0CzVFwtSYQMucfKEfq02FZz43uaK/8PIUgkEIgr0AABPpdAs1RcLUmEDLnHyhH6tNhWc+N7miv/DyFIJFCIK9AAAT6XQLNUXC1JhAy5x8oR+rTYVnPje5or/w8hSCSQiCvQAAE+l0CzVFwtSYQMucfKEfq02FZz43uaK/8PIUgk0pAYjkwZTUwZGFmODIwMjQ2MjQ1NDllNzcwODE5OWRkMDViNmRlOGUxMGQ2ZGY2MmNkMjc1ODFjNjVlNTA5NmIyNEpANWY3ODYyNDlkMmI1MDE4Y2UwODQ0NDJmYjNkY2U2MTgwMDg3YWQ1Y2E2YWUzZmQwNDg3NDAyMjAwY2I0MTc3ZkpAMWRhM2M4YWM4MTc3NjJmNmYzNmVmZGJjMTRjNjZiZTM1N2VkOWY0YmNmYTdmMjdlY2E5ZjBhYWEyNjE4ZmE0NkpAOTY3ZjM3ZDE0YWZlYjhkMGU1OTM1ZTE3MzI2NWY1OWI1NWFmY2JjMDRiOWZlYmEzZmNiMmUwYWEzYzk2NGZhN1IAWICo1rkHYICU69wDaICEr19wgISvX3iAhK9fggGMAQog0hE5M5sxG47EI185H10kNCHQhkpmwspXZqsPV7+qXnUSIOj0ItmcmAv232ne4Nlqjc9GgagueO6ok2daXRsq/2C2GhB/vzvH+Aj0jKKWqAsDxqbGGhCQDRvnURzFURcHXAdpvyDMGhDrk1kDNKBqRnCNDCmhK8RkGhB6RC+kAddU9FU+tDNrLDMl'

describe('OCR2 Execution', () => {
  let cosmClient: CosmWasmClient
  let OCR2: Ocr2QueryClient
  let CW20Query: CW20BaseQueryClient
  let CW20Exec: CW20BaseClient
  let ocr2Addr: string
  let deployerWallet: DirectSecp256k1HdWallet
  let deployerAddr: string
  let aliceAddr: string
  let linkAddr: string
  let billingACAddr: string
  let requesterACAddr: string
  const revertMsg = /[\S\s]*query wasm contract failed: unknown request[\S\s]*/

  afterAll(async () => {
    await endWasmd()
  })

  const verifyDeployment = async () => {
    expect(await OCR2.owner()).toEqual(deployerAddr)
    expect(await OCR2.latestConfigDetails()).toEqual({
      config_count: 0,
      block_number: 0,
      config_digest: new Array(32).fill(0),
    })
    expect(OCR2.latestTransmissionDetails()).rejects.toThrow(revertMsg)
    expect(await OCR2.latestConfigDigestAndEpoch()).toEqual({
      config_digest: new Array(32).fill(0),
      epoch: 0,
      scan_logs: false,
    })
    expect(await OCR2.requesterAccessController()).toEqual(requesterACAddr)
    expect(await OCR2.billingAccessController()).toEqual(billingACAddr)
    expect(await OCR2.description()).toEqual('yessir it is ocr2!')
    expect(await OCR2.decimals()).toEqual(18)
    expect(OCR2.latestRoundData()).rejects.toThrow(revertMsg)
    expect(await OCR2.linkToken()).toEqual(linkAddr)
    expect(await OCR2.billing()).toEqual({
      gas_adjustment: null,
      gas_base: null,
      gas_per_signature: null,
      observation_payment_gjuels: 0,
      recommended_gas_price_micro: '0',
      transmission_payment_gjuels: 0,
    })
    expect(OCR2.owedPayment({ transmitter: deployerAddr })).rejects.toThrow(revertMsg)
    expect(await OCR2.linkAvailableForPayment()).toEqual({
      amount: '0',
    })
    expect(OCR2.oracleObservationCount({ transmitter: deployerAddr })).rejects.toThrow(revertMsg)
  }

  beforeAll(async () => {
    // Ideally, we'd start wasmd beforeEach() but it takes too long
    const [deployer, alice] = await initWasmd()
    deployerWallet = deployer
    aliceAddr = await toAddr(alice)

    deployerAddr = await toAddr(deployer)

    // deploy access controller and link tokens
    linkAddr = await deployLink()
    billingACAddr = await deployAC()
    requesterACAddr = await deployAC()

    // just give two non-contract addresses
    ocr2Addr = await deployOCR2({
      minSubmissionValue: '1',
      maxSubmissionValue: '3',
      decimals: '18',
      name: 'yessir it is ocr2!',
      billingAccessController: billingACAddr,
      requesterAccessController: requesterACAddr,
      link: linkAddr,
    })
    const deployerSignedClient = await SigningCosmWasmClient.connectWithSigner(NODE_URL, deployer, {
      gasPrice: GasPrice.fromString(DEFAULT_GAS_PRICE),
    })
    cosmClient = await CosmWasmClient.connect(NODE_URL)
    OCR2 = new Ocr2QueryClient(cosmClient, ocr2Addr)
    CW20Query = new CW20BaseQueryClient(cosmClient, linkAddr)
    CW20Exec = new CW20BaseClient(deployerSignedClient, deployerAddr, linkAddr)

    // test in pre-setup instead to avoid state changes from another test running before it
    await verifyDeployment()
  }, TIMEOUT)

  it(
    'Set Billing',
    async () => {
      await new SetBilling(
        {
          ...CMD_FLAGS,
          recommendedGasPriceMicro: '2',
          observationPaymentGjuels: 7,
          transmissionPaymentGjuels: 10,
        },
        [ocr2Addr],
      ).run()

      expect(await OCR2.billing()).toEqual({
        gas_adjustment: null,
        gas_base: null,
        gas_per_signature: null,
        observation_payment_gjuels: 7,
        recommended_gas_price_micro: '2',
        transmission_payment_gjuels: 10,
      })
    },
    TIMEOUT,
  )

  // Ideally, we'd split into multiple tests, but redeploying the
  // contract in a beforeEach hook each time would be tedious
  it(
    'Config Management',
    async () => {
      // initialize proposal
      const beginResp = await new BeginProposal(
        {
          ...CMD_FLAGS,
        },
        [ocr2Addr],
      ).run()
      const beginEvents = (beginResp['responses'][0]['tx'] as ExecuteResult).events
      const beginEvent = beginEvents.find((e) => e.type === 'wasm')!
      const proposalId = beginEvent.attributes.find((a) => a.key === 'proposal_id')!.value

      const proposal = await OCR2.proposal({ id: proposalId })
      expect(proposal).toEqual({
        f: 0,
        finalized: false,
        offchain_config: '',
        offchain_config_version: 0,
        oracles: [],
        owner: deployerAddr,
      })

      // configure proposal
      await new ProposeConfig(
        {
          ...CMD_FLAGS,
          f: 1,
          proposalId,
          signers: SIGNERS,
          transmitters: TRANSMITTERS,
          payees: PAYEES,
        },
        [ocr2Addr],
      ).run()

      const updatedProposal = await OCR2.proposal({ id: proposalId })
      expect(updatedProposal).toEqual({
        f: 1,
        finalized: false,
        offchain_config: '',
        offchain_config_version: 0,
        oracles: [
          [
            'uQ5Q2vggJGJFSedwgZndBbbejhDW32LNJ1gcZeUJayQ=',
            'wasm1jcamds37x233wtpzj5k3f5f7gszaahjghqvdxe',
            'wasm1lsagfzrm4gz28he4wunt63sts5xzmczwda8vl6',
          ],
          [
            'X3hiSdK1AYzghEQvs9zmGACHrVymrj/QSHQCIAy0F38=',
            'wasm1a2dr8rjv2ft2z6g4spe3534hxn9hu50sutxqw5',
            'wasm1jv7p2hml6gsu2k0ynx0uwtqrcyjqv8xv6k3ua6',
          ],
          [
            'HaPIrIF3Yvbzbv28FMZr41ftn0vPp/J+yp8KqiYY+kY=',
            'wasm1p2udqrgwtyh2t284829cwzzfqc2kjaw4gvhqza',
            'wasm1a6vse48zvfge8yw53m5u62rw7xw3qmf2x6jgcu',
          ],
          [
            'ln830Ur+uNDlk14XMmX1m1Wvy8BLn+uj/LLgqjyWT6c=',
            'wasm1sya395n2l8z74efxmfw7fxfxseke3wpypqd68w',
            'wasm18vga7wnhks6jaqhzjf4hlnqdmmk6x9sgf2ztzv',
          ],
        ],
        owner: deployerAddr,
      })

      // propose offchain config
      await new ProposeOffchainConfig(
        {
          ...CMD_FLAGS,
          input: {
            offchainConfig: OFF_CHAIN_CONFIG,
            proposalId,
            f: 1,
            signers: SIGNERS,
            transmitters: TRANSMITTERS,
            onchainConfig: '',
            offchainConfigVersion: 1,
            secret: 'awe accuse polygon tonic depart acuity onyx inform bound gilbert expire',
            randomSecret: 'random chaos',
          },
        },
        [ocr2Addr],
      ).run()

      const proposalWithOffchainData = await OCR2.proposal({ id: proposalId })
      expect(proposalWithOffchainData.offchain_config).toEqual(EXPECTED_OFF_CHAIN_CONFIG_DIGEST)
      expect(proposalWithOffchainData.offchain_config_version).toEqual(1)

      // finalize proposal
      const res = await new FinalizeProposal(
        {
          ...CMD_FLAGS,
          proposalId,
        },
        [ocr2Addr],
      ).run()

      // retrieve the proposal digest from the 'wasm' event attributes
      const events = (res['responses'][0]['tx'] as ExecuteResult).events
      const wasmEventAttrs = events[events.length - 1].attributes
      const proposalDigest = wasmEventAttrs[wasmEventAttrs.length - 1].value

      const finalizedProposal = await OCR2.proposal({ id: proposalId })
      expect(finalizedProposal.finalized).toEqual(true)

      // approve proposal
      const approveRes = await new AcceptProposal(
        {
          ...CMD_FLAGS,
          input: {
            proposalId,
            // proposal digeset is not to be confused with the offchain digest
            digest: proposalDigest,
            offchainConfig: OFF_CHAIN_CONFIG,
            secret: 'awe accuse polygon tonic depart acuity onyx inform bound gilbert expire',
            randomSecret: 'random chaos',
          },
        },
        [ocr2Addr],
      ).run()

      // retrieve updates from 'set_config' event
      const approveEvents = (approveRes['responses'][0]['tx'] as ExecuteResult).events
      const setConfigEvent = approveEvents.filter((e) => e.type === 'wasm-set_config')[0]

      const latest = await OCR2.latestConfigDigestAndEpoch()

      expect(setConfigEvent.attributes).toEqual([
        {
          key: '_contract_address',
          value: ocr2Addr,
        },
        { key: 'previous_config_block_number', value: '0' },
        {
          key: 'latest_config_digest',
          // convert array of decimals to hex string representation
          value: latest.config_digest.reduce((acc, num) => acc + num.toString(16).padStart(2, '0'), ''),
        },
        { key: 'config_count', value: '1' },
        {
          key: 'signers',
          value: 'b90e50daf82024624549e7708199dd05b6de8e10d6df62cd27581c65e5096b24',
        },
        {
          key: 'signers',
          value: '5f786249d2b5018ce084442fb3dce6180087ad5ca6ae3fd0487402200cb4177f',
        },
        {
          key: 'signers',
          value: '1da3c8ac817762f6f36efdbc14c66be357ed9f4bcfa7f27eca9f0aaa2618fa46',
        },
        {
          key: 'signers',
          value: '967f37d14afeb8d0e5935e173265f59b55afcbc04b9feba3fcb2e0aa3c964fa7',
        },
        {
          key: 'transmitters',
          value: 'wasm1jcamds37x233wtpzj5k3f5f7gszaahjghqvdxe',
        },
        {
          key: 'transmitters',
          value: 'wasm1a2dr8rjv2ft2z6g4spe3534hxn9hu50sutxqw5',
        },
        {
          key: 'transmitters',
          value: 'wasm1p2udqrgwtyh2t284829cwzzfqc2kjaw4gvhqza',
        },
        {
          key: 'transmitters',
          value: 'wasm1sya395n2l8z74efxmfw7fxfxseke3wpypqd68w',
        },
        {
          key: 'payees',
          value: 'wasm1lsagfzrm4gz28he4wunt63sts5xzmczwda8vl6',
        },
        {
          key: 'payees',
          value: 'wasm1jv7p2hml6gsu2k0ynx0uwtqrcyjqv8xv6k3ua6',
        },
        {
          key: 'payees',
          value: 'wasm1a6vse48zvfge8yw53m5u62rw7xw3qmf2x6jgcu',
        },
        {
          key: 'payees',
          value: 'wasm18vga7wnhks6jaqhzjf4hlnqdmmk6x9sgf2ztzv',
        },
        { key: 'f', value: '1' },
        {
          key: 'onchain_config',
          value: 'AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAw==',
        },
        { key: 'offchain_config_version', value: '1' },
        {
          key: 'offchain_config',
          value: EXPECTED_OFF_CHAIN_CONFIG_DIGEST,
        },
      ])

      // test payment is 0
      await new WithdrawPayment(
        {
          ...CMD_FLAGS,
          // transmitter associated with the deployer payee
          transmitter: 'wasm1jcamds37x233wtpzj5k3f5f7gszaahjghqvdxe',
        },
        [ocr2Addr],
      ).run()
    },
    TIMEOUT,
  )

  it(
    'Withdraw Remaining Funds To Chosen Receipient',
    async () => {
      // send some link to ocr2 contract
      await CW20Exec.transfer({ amount: '10', recipient: ocr2Addr })
      expect(await CW20Query.balance({ address: ocr2Addr })).toEqual({ balance: '10' })

      await new WithdrawFunds(
        {
          ...CMD_FLAGS,
          amount: '5',
          recipient: aliceAddr,
        },
        [ocr2Addr],
      ).run()
      expect(await CW20Query.balance({ address: aliceAddr })).toEqual({ balance: '5' })

      await new WithdrawFunds(
        {
          ...CMD_FLAGS,
          all: true,
          recipient: aliceAddr,
        },
        [ocr2Addr],
      ).run()
      expect(await CW20Query.balance({ address: aliceAddr })).toEqual({ balance: '10' })
    },
    TIMEOUT,
  )

  it(
    'Retire OCR2 Contract',
    async () => {
      const beginResp = await new BeginProposal(
        {
          ...CMD_FLAGS,
        },
        [ocr2Addr],
      ).run()
      const beginEvents = (beginResp['responses'][0]['tx'] as ExecuteResult).events
      const beginEvent = beginEvents.find((e) => e.type === 'wasm')!
      const proposalId = beginEvent.attributes.find((a) => a.key === 'proposal_id')!.value

      await new ProposeConfigClose(
        {
          ...CMD_FLAGS,
          configProposal: proposalId,
        },
        [ocr2Addr],
      ).run()

      const closedProposal = await OCR2.proposal({ id: proposalId })

      expect(closedProposal).toEqual({
        f: 1,
        finalized: false,
        offchain_config: '',
        offchain_config_version: 0,
        oracles: [
          [
            'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=',
            'wasm1hft9sxhx7d7furw9y0rjxu4hfsm76ehkman78g',
            'wasm1hft9sxhx7d7furw9y0rjxu4hfsm76ehkman78g',
          ],
          [
            'ERERERERERERERERERERERERERERERERERERERERERE=',
            'wasm10f0wy3fs6ex395ylturr0hv03m3cjcjpy4ux6x',
            'wasm10f0wy3fs6ex395ylturr0hv03m3cjcjpy4ux6x',
          ],
          [
            'IiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiI=',
            'wasm1jv45uny4kuyeecgzw5xftkr7nssdj5e56ajchs',
            'wasm1jv45uny4kuyeecgzw5xftkr7nssdj5e56ajchs',
          ],
          [
            'MzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzM=',
            'wasm1n947av78pavcs9tpp79me30gk6rfqhnam9gsls',
            'wasm1n947av78pavcs9tpp79me30gk6rfqhnam9gsls',
          ],
        ],
        owner: deployerAddr,
      })

      await new ProposeOffchainConfigClose(
        {
          ...CMD_FLAGS,
          configProposal: proposalId,
        },
        [ocr2Addr],
      ).run()
      const proposalWithOffchainData = await OCR2.proposal({ id: proposalId })
      expect(proposalWithOffchainData.offchain_config).toEqual(
        'UgCCAUQKIE9e1js7V6huFZm3L+xN069zJi9nK1O7unSzsCoJ5KQ/EiBzBdfpzmkffyiUqgkafLFunVBq+Nrzl5wGqEZgr0YLIg==',
      )
      expect(proposalWithOffchainData.offchain_config_version).toEqual(0)

      const finalizeResp = await new FinalizeProposal(
        {
          ...CMD_FLAGS,
          proposalId,
        },
        [ocr2Addr],
      ).run()
      const finalizedProposal = await OCR2.proposal({ id: proposalId })
      expect(finalizedProposal.finalized).toEqual(true)

      // retrieve the proposal digest from the 'wasm' event attributes
      const events = (finalizeResp['responses'][0]['tx'] as ExecuteResult).events
      const wasmEventAttrs = events[events.length - 1].attributes
      const proposalDigest = wasmEventAttrs[wasmEventAttrs.length - 1].value

      // just make sure it passes (we test accept proposal logic elsewhere)
      await new AcceptProposalClose(
        {
          ...CMD_FLAGS,
          input: {
            proposalId,
            // proposal digeset is not to be confused with the offchain digest
            digest: proposalDigest,
            secret: 'awe accuse polygon tonic depart acuity onyx inform bound gilbert expire',
          },
        },
        [ocr2Addr],
      ).run()
    },
    TIMEOUT,
  )
})
