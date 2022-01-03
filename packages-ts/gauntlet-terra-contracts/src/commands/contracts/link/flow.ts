import { FlowCommand } from '@chainlink/gauntlet-core'
import { waitExecute, TransactionResponse } from '@chainlink/gauntlet-terra'

import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { CATEGORIES } from '../../../lib/constants'
import DeployLink from './deployLink'

export default class LinkFlow extends FlowCommand<TransactionResponse> {
  static id = 'deploy:link:flow'
  static category = CATEGORIES.LINK

  constructor(flags, args) {
    super(flags, args, waitExecute)

    this.stepIds = {
      LINK: 1,
      LINK_2: 2,
    }

    this.flow = [
      {
        name: 'Deploy LINK',
        command: DeployLink,
        id: this.stepIds.LINK,
      },
      {
        name: 'Deploy another LINK',
        command: DeployLink,
        id: this.stepIds.LINK_2,
      },
      {
        name: 'Do something!',
        exec: this.doSomething,
      },
    ]
  }

  doSomething = () => {
    // Access to any Step
    const linkAddress = this.getReportStepDataById(ID.contract(this.stepIds.LINK))
    const linkAddress2 = this.getReportStepDataById(ID.contract(this.stepIds.LINK_2))
    logger.log(`
    Previous deployments:
      - LINK 1: ${linkAddress}
      - LINK 2: ${linkAddress2}
    `)
  }
}

const ID = {
  contract: (id: number, index = 0): string => `ID.${id}.txs.${index}.contract`,
  tx: (id: number, index = 0): string => `ID.${id}.txs.${index}.tx`,
}
