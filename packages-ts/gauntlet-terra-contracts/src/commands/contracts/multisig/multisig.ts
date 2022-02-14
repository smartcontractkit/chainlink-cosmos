// multisig.ts
//
// For now, propose, vote, and execute functionality are all combined into one CONTRACT:COMMAND::multisig meta-command
//  This is parallel to how things are implemented in Solana.  The execute happens automatically as soon as the last
//  vote required to exceeed the threshold is cast.  And the difference between propose and vote is distinguished by
//  whether the --proposal=PROPOSAL_HASH flag is passed.  Later, we may want to split this into CONTRACT::COMMAND::propose,
//  CONTRACT::COMMAND::vote, and CONTRACT::COMMAND::execute.  We may also want to add CONTRACT::COMMAND::close, to
//  abort a proposal early (before it expires), disallowing any further voting on it.

import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { Result } from '@chainlink/gauntlet-core'
import { TerraCommand, RawTransaction, TransactionResponse } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../../lib/constants'
import { CONTRACT_LIST, Contract, getContract, TERRA_OPERATIONS } from '../../../lib/contracts'
import AbstractCommand from '../../abstract'

type ProposalContext = { 
    rawTx: RawTransaction,
    multisigSigner: string,
    proposalState: any,
}

type StringGetter = () => string

abstract class MultisigTerraCommand  extends TerraCommand {
    static category = CATEGORIES.MULTISIG

    commandType:any
    multisigOp:StringGetter

    command:AbstractCommand
    multisigAddress:string
    multisigContract: Promise<Contract>

    constructor(flags, args) {
        super(flags, args)
        
        logger.info(`Running ${this.commandType()} in multisig mode`)
        this.command = new this.commandType()(flags, args)
        this.command.invokeMiddlewares(this.command, this.command.middlewares)
        this.require(!!process.env.MULTISIG_ADDRESS, 'Please set MULTISIG_ADDRESS env var')
        this.multisigContract = getContract(CONTRACT_LIST.MULTISIG, flags.version)
        this.multisigAddress = process.env.MULTISIG_ADDRESS!
    }

    execute = async (): Promise<Result<TransactionResponse>> => {
        const tx = this.command.makeRawTransaction()
        console.debug(tx)

        return {
            responses: [],  
          } as Result<TransactionResponse>
    }
}

export const wrapCommand = (command) => {
    return class CustomCommand extends MultisigTerraCommand {
      static id = `${command.id}:multisig`
      static category = CATEGORIES.MULTISIG
  }
}

export { MultisigTerraCommand }
