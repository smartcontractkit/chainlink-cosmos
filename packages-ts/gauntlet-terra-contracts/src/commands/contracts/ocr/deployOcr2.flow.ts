import { FlowCommand } from '@chainlink/gauntlet-core'
import { CATEGORIES } from '../../../lib/constants'
import { waitExecute, TransactionResponse } from '@chainlink/gauntlet-terra'
import { makeAbstractCommand } from '../../abstract'
export default class DeployOCR2Flow extends FlowCommand<TransactionResponse> {
  static id = 'ocr2:deploy:flow'
  static category = CATEGORIES.OCR

  // yarn gauntlet ocr2:deploy:flow --network=bombay-testnet --decimals=8 --description="OCR2" --max_answer=10000000 --min_answer=0

  constructor(flags, args) {
    super(flags, args, waitExecute, makeAbstractCommand)

    this.stepIds = {
      BILLING_ACCESS_CONTROLLER: 1,
      REQUEST_ACCESS_CONTROLLER: 2,
      LINK: 3,
    }

    this.flow = [
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
        name: 'Deploy OCR 2',
        command: 'ocr2:deploy',
        flags: {
          billing_access_controller: ID.contract(this.stepIds.BILLING_ACCESS_CONTROLLER),
          requester_access_controller: ID.contract(this.stepIds.REQUEST_ACCESS_CONTROLLER),
          link_token: process.env.LINK_ADDRESS,
          decimals: Number(this.flags.decimals),
        },
      },
    ]
  }
}

const ID = {
  contract: (id: number, index = 0): string => `ID.${id}.txs.${index}.contract`,
  tx: (id: number, index = 0): string => `ID.${id}.txs.${index}.tx`,
}
