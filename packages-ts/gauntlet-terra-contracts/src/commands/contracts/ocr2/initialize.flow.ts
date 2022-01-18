import { FlowCommand } from '@chainlink/gauntlet-core'
import { CONTRACT_LIST } from '../../../lib/contracts'
import { waitExecute, TransactionResponse } from '@chainlink/gauntlet-terra'

import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { makeAbstractCommand } from '../../abstract'
import DeployOCR2 from './deploy'
import SetBilling from './setBilling'
import SetConfig from './setConfig'
import SetPayees from './setPayees'

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
        command: DeployOCR2,
        id: this.stepIds.OCR_2,
      },
      {
        name: 'Change RDD',
        exec: this.showRddInstructions,
      },
      {
        name: 'Set Billing',
        command: SetBilling,
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
      },
      {
        name: 'Set Config',
        command: SetConfig,
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
      },
      {
        name: 'Set Payees',
        command: SetPayees,
        args: [this.getReportStepDataById(FlowCommand.ID.contract(this.stepIds.OCR_2))],
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
