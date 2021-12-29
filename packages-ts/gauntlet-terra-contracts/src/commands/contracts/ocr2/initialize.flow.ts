import { FlowCommand } from '@chainlink/gauntlet-core'
import { CONTRACT_LIST } from '../../../lib/contracts'
import { waitExecute, TransactionResponse } from '@chainlink/gauntlet-terra'

import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { makeAbstractCommand } from '../../abstract'
import { makeOCR2DeployCommand } from './deploy'
import { makeOCR2SetConfigCommand } from './setConfig'
import { makeOCR2SetBillingCommand } from './setBilling'
import { makeOCR2SetPayeesCommand } from './setPayees'

export default class OCR2InitializeFlow extends FlowCommand<TransactionResponse> {
  static id = 'ocr2:initialize:flow'
  static category = CONTRACT_LIST.OCR_2
  static examples = ['yarn gauntlet ocr2:initialize:flow --network=local --id=[ID] --rdd=[PATH_TO_RDD]']

  constructor(flags, args) {
    super(flags, args, waitExecute, makeAbstractCommand)

    this.stepIds = {
      OCR_2: 1,
    }

    this.flow = [
      {
        name: 'Deploy OCR 2',
        command: makeOCR2DeployCommand,
        id: this.stepIds.OCR_2,
        type: 'maker',
      },
      {
        name: 'Change RDD',
        exec: this.showRddInstructions,
      },
      {
        name: 'Set Config',
        command: makeOCR2SetConfigCommand,
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
        name: 'Set Billing',
        command: makeOCR2SetBillingCommand,
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
        type: 'maker',
      },
      // Inspection here
      // {
      //   name: 'Inspection',
      //   command: OCR2Inspect,
      //   flags: {
      //     state: FlowCommand.ID.contract(this.stepIds.OCR_2),
      //     billingAccessController: process.env.BILLING_ACCESS_CONTROLLER || this.flags.billingAccessController,
      //     requesterAccessController: process.env.REQUESTER_ACCESS_CONTROLLER || this.flags.requesterAccessController,
      //     link: process.env.LINK || this.flags.link,
      //   },
      // },
    ]
  }

  showRddInstructions = async () => {
    logger.info(
      `
        Change the RDD ID with the new contract address: 
          - Contract Address: ${this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))}
      `,
    )

    await prompt('Ready? Continue')
  }
}
