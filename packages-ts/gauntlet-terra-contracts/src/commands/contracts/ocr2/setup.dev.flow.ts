import { FlowCommand } from '@chainlink/gauntlet-core'
import { CATEGORIES } from '../../../lib/constants'
import { waitExecute, TransactionResponse } from '@chainlink/gauntlet-terra'
import { makeAbstractCommand } from '../../abstract'
import UploadContractCode from '../../tooling/upload'
import DeployLink from '../link/deploy'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { makeOCR2DeployCommand } from './deploy'
import { makeOCR2SetConfigCommand } from './setConfig'
import { makeOCR2SetBillingCommand } from './setBilling'
import { makeOCR2SetPayeesCommand } from './setPayees'

export default class DeployOCR2Flow extends FlowCommand<TransactionResponse> {
  static id = 'ocr2:setup:flow'
  static category = CATEGORIES.OCR

  // yarn gauntlet ocr2:deploy:flow --network=bombay-testnet --decimals=8 --description="OCR2" --max_answer=10000000 --min_answer=0

  constructor(flags, args) {
    super(flags, args, waitExecute, makeAbstractCommand)

    this.stepIds = {
      BILLING_ACCESS_CONTROLLER: 1,
      REQUEST_ACCESS_CONTROLLER: 2,
      LINK: 3,
      OCR_2: 4,
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
        command: makeOCR2DeployCommand,
        id: this.stepIds.OCR_2,
        type: 'maker',
      },
      {
        name: 'Set Billing',
        command: makeOCR2SetBillingCommand,
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
        type: 'maker',
      },
      {
        name: 'Set Payees',
        command: makeOCR2SetPayeesCommand,
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
        type: 'maker',
      },
      {
        name: 'Set Config',
        command: makeOCR2SetConfigCommand,
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
        type: 'maker',
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
