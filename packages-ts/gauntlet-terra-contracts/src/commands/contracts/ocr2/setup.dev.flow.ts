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

    this.flow = [
      {
        name: 'Upload Contracts',
        command: UploadContractCode,
      },
      {
        name: 'Deploy LINK',
        command: DeployLink,
        id: this.stepIds.LINK,
      },
      {
        name: 'Deploy Billing Access Controller',
        command: 'access_controller:deploy',
        id: this.stepIds.BILLING_ACCESS_CONTROLLER,
      },
      {
        name: 'Deploy Request Access Controller',
        command: 'access_controller:deploy',
        id: this.stepIds.REQUEST_ACCESS_CONTROLLER,
      },
      {
        name: 'Set environment',
        exec: this.setEnvironment,
      },
      {
        name: 'Deploy OCR 2',
        command: DeployOCR2,
        id: this.stepIds.OCR_2,
      },
      {
        name: 'Set Billing',
        command: SetBilling,
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
      },
      {
        name: 'Set Payees',
        command: SetPayees,
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
      },
      {
        name: 'Set Config',
        command: SetConfig,
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
