import { FlowCommand } from '@chainlink/gauntlet-core'
import { CATEGORIES } from '../../../lib/constants'
import { waitExecute, TransactionResponse } from '@chainlink/gauntlet-terra'
import { makeAbstractCommand } from '../../abstract'
import UploadContractCode from '../../tooling/upload'
import DeployLink from '../link/deploy'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import DeployOCR2 from './deploy'
import SetBilling from './setBilling'
import SetConfig from './setConfig'
import SetPayees from './setPayees'

export default class DeployOCR2Flow extends FlowCommand<TransactionResponse> {
  static id = 'ocr2:setup:flow'
  static category = CATEGORIES.OCR

  constructor(flags, args) {
    super(flags, args, waitExecute, makeAbstractCommand)

    this.stepIds = {
      BILLING_ACCESS_CONTROLLER: 1,
      REQUEST_ACCESS_CONTROLLER: 2,
      LINK: 3,
      OCR_2: 4,
    }

    const billingInput = {
      recommendedGasPrice: 1,
      observationPaymentGjuels: 1,
      transmissionPaymentGjuels: 1,
    }

    const offchainConfigInput = {
      f: 1,
      deltaProgressNanoseconds: 300000000,
      deltaResendNanoseconds: 300000000,
      deltaRoundNanoseconds: 30,
      deltaGraceNanoseconds: 30,
      deltaStageNanoseconds: 30,
      rMax: 30,
      s: [1, 1, 1, 1],
      offchainPublicKeys: [
        '5cd10bf991c8b0db7bee3ec371c7795a69297b6bccf7b4d738e0920b56131772',
        'd58a9b179d5ac550376734ce1da5ee4572718fd6d315e0541b1da1d1671d0d71',
        'ef104fe8812c2c73d4c1b57dc82a15f8dd5a23149bd91917abad295f305ed21a',
      ],
      peerIds: [
        'DxRwKpwNBuMzKf5YEG1vLpnRbWeKo1Z4tKHfFGt8vUkj',
        '8sdUrh9LQdAXhrgFEBDxnUauJTTLfEq5PNsJbn9Pw19K',
        '9n1sSGA5rhfsQyaX3tHz3ZU1ffR6V8KffvWtFPBcFrJw',
      ],
      reportingPluginConfig: {
        alphaReportInfinite: false,
        alphaReportPpb: 0,
        alphaAcceptInfinite: false,
        alphaAcceptPpb: 30,
        deltaCNanoseconds: 30,
      },
      maxDurationQueryNanoseconds: 30,
      maxDurationObservationNanoseconds: 30,
      maxDurationReportNanoseconds: 30,
      maxDurationShouldAcceptFinalizedReportNanoseconds: 30,
      maxDurationShouldTransmitAcceptedReportNanoseconds: 30,
      configPublicKeys: [
        '5cd10bf991c8b0db7bee3ec371c7795a69297b6bccf7b4d738e0920b56131772',
        'd58a9b179d5ac550376734ce1da5ee4572718fd6d315e0541b1da1d1671d0d71',
        'ef104fe8812c2c73d4c1b57dc82a15f8dd5a23149bd91917abad295f305ed21a',
      ],
    }

    const transmitters = [
      'terra1fcksmfjncl6m7apvpalvhwv5jxd9djv5lwyu82',
      'terra1trcufj64y53hxk7g8cra33xw3jkyvlr9lu99eu',
      'terra1s38kfu4qp0ttwxkka9zupaysefl5qruhv5rc0z',
      'terra19ty8cgqjvl26aj809xgd3kksj4kdqu0gkssxca',
    ]
    const configInput = {
      signers: [
        '0cAFF71b6Dbb4f9Ebc862F8E9C124E737C917e80',
        '6b211EdeF015C9931eA7D65CD326472891ecf501',
        'C6CD7e27Ea7653362906A7C9923c15602dC04F41',
        '1b7c57E22a4D4B6c94365A73AD5FF743DBE9c55E',
      ],
      transmitters,
      offchainConfig: offchainConfigInput,
      offchainConfigVersion: 2,
      onchainConfig: [],
    }

    const payeesInput = {
      payees: [
        'terra18lq43mfarxmyuvpyj0wu40selmpgfmss69vj2d',
        'terra1rd37efmhvscdjcpakqxym68zv9da7uvt5ld62y',
        'terra1c66g2zcd7ch0rpmgkpmnqkxma49rwtt74wgzex',
        'terra1404zfsh35k383akcs9hg4z6r27cgy3h96tq4tx',
      ],
      transmitters,
    }

    this.flow = [
      // {
      //   name: 'Upload Contracts',
      //   command: UploadContractCode,
      // },
      // {
      //   name: 'Deploy LINK',
      //   command: DeployLink,
      //   id: this.stepIds.LINK,
      // },
      // {
      //   name: 'Deploy Billing Access Controller',
      //   command: 'access_controller:deploy',
      //   id: this.stepIds.BILLING_ACCESS_CONTROLLER,
      // },
      // {
      //   name: 'Deploy Request Access Controller',
      //   command: 'access_controller:deploy',
      //   id: this.stepIds.REQUEST_ACCESS_CONTROLLER,
      // },
      // {
      //   name: 'Set environment',
      //   exec: this.setEnvironment,
      // },
      {
        name: 'Deploy OCR 2',
        command: DeployOCR2,
        flags: {
          input: {
            billingAccessController: process.env.BILLING_ACCESS_CONTROLLER,
            requesterAccessController: process.env.REQUESTER_ACCESS_CONTROLLER,
            linkToken: process.env.LINK,
            decimals: 18,
            description: 'TEST',
            maxAnswer: '100000000000',
            minAnswer: '1',
          },
        },
        id: this.stepIds.OCR_2,
      },
      {
        name: 'Set Billing',
        command: SetBilling,
        flags: {
          input: billingInput,
        },
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
      },
      {
        name: 'Set Payees',
        command: SetPayees,
        flags: {
          input: payeesInput,
        },
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
      },
      {
        name: 'Set Config',
        command: SetConfig,
        flags: {
          input: configInput,
        },
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
      },
    ]
  }

  setEnvironment = async () => {
    const linkAddress = this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.LINK))
    const billingAC = this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.BILLING_ACCESS_CONTROLLER))
    const requesterAC = this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.REQUEST_ACCESS_CONTROLLER))
    logger.info(`
      Setting the following env variables. Include them into .env.${this.flags.network} for future runs
        LINK=${linkAddress}
        BILLING_ACCESS_CONTROLLER=${billingAC}
        REQUESTER_ACCESS_CONTROLLER=${requesterAC}
      `)
    process.env.LINK = linkAddress
    process.env.BILLING_ACCESS_CONTROLLER = billingAC
    process.env.REQUESTER_ACCESS_CONTROLLER = requesterAC
    await prompt('Continue?')
  }
}
