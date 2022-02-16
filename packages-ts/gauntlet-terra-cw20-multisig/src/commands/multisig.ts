import { Result } from '@chainlink/gauntlet-core'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { AccAddress, MsgExecuteContract } from '@terra-money/terra.js'

type ProposalAction = (
  signer: AccAddress,
  proposalId: number,
  message: MsgExecuteContract,
) => Promise<MsgExecuteContract>

enum Vote {
  YES = 'yes',
  NO = 'no',
  ABS = 'abstain',
  VETO = 'veto',
}

type WasmMsg = {
  execute: {
    contract_addr: string
    funds: {
      denom: string
      amount: string
    }[]
    msg: string
  }
}

enum Action {
  CREATE = 'create',
  APPROVE = 'approve',
  EXECUTE = 'execute',
}

type State = {
  threshold: number
  proposalStatus: Action
}

export const wrapCommand = (command) => {
  return class Multisig extends TerraCommand {
    command: TerraCommand
    multisig: AccAddress

    static id = `${command.id}:multisig`

    constructor(flags, args) {
      super(flags, args)

      this.command = new command(flags, args)

      if (!AccAddress.validate(process.env.MULTISIG_ADDRESS)) throw new Error(`Invalid Multisig wallet address`)
      this.multisig = process.env.MULTISIG_ADDRESS as AccAddress
    }

    makeRawTransaction = async (signer: AccAddress, status?: Action) => {
      const message = await this.command.makeRawTransaction(this.multisig)

      const operations = {
        [Action.CREATE]: this.executePropose,
        [Action.APPROVE]: this.executeApproval,
        [Action.EXECUTE]: this.executeExecution,
      }

      return operations[status](signer, Number(this.flags.proposal), message)
    }

    toWasmMsg = (message: MsgExecuteContract): WasmMsg => {
      return {
        execute: {
          contract_addr: message.contract,
          funds: message.coins.toArray().map((c) => c.toData()),
          msg: Buffer.from(message.toJSON()).toString('base64'),
        },
      }
    }

    executePropose: ProposalAction = async (signer, proposalId, message) => {
      logger.info('Generating data for creating new proposal')
      const proposeInput = {
        propose: {
          description: command.id,
          msgs: [
            {
              wasm: this.toWasmMsg(message),
            },
          ],
          title: command.id,
          // latest: {
          //   never: {},
          // },
        },
      }
      return new MsgExecuteContract(signer, this.multisig, proposeInput)
    }

    executeApproval: ProposalAction = async (signer, proposalId) => {
      logger.info(`Generating data for approving proposal ${proposalId}`)
      const approvalInput = {
        vote: {
          vote: Vote.YES,
          proposal_id: proposalId,
        },
      }
      return new MsgExecuteContract(signer, this.multisig, approvalInput)
    }

    executeExecution: ProposalAction = async (signer, proposalId) => {
      logger.info(`Generating data for executing proposal ${proposalId}`)
      const executeInput = {
        execute: {
          proposal_id: proposalId,
        },
      }
      return new MsgExecuteContract(signer, this.multisig, executeInput)
    }

    fetchState = async (proposalId?: number): Promise<State> => {
      const threshold = await this.query(this.multisig, {
        threshold: {},
      })
      if (!proposalId)
        return {
          threshold,
          proposalStatus: Action.CREATE,
        }
      const proposalState = await this.query(this.multisig, {
        proposal: {
          proposal_id: proposalId,
        },
      })
      console.log(proposalState)

      return {
        threshold,
        proposalStatus: Action.APPROVE,
      }
    }

    execute = async () => {
      let proposalId = !!this.flags.proposal && Number(this.flags.proposal)
      const state = await this.fetchState(proposalId)
      const rawTx = await this.makeRawTransaction(this.wallet.key.accAddress, state.proposalStatus)

      console.info(`
        Proposal State:
          - Threshold: ${state.threshold}
          - Status: ${state.proposalStatus}
      `)

      const actionMessage = {
        [Action.CREATE]: 'CREATING',
        [Action.APPROVE]: 'APPROVING',
        [Action.EXECUTE]: 'EXECUTING',
      }
      await prompt(`Continue ${actionMessage[state.proposalStatus]} proposal?`)
      const tx = await this.signAndSend([rawTx])

      if (state.proposalStatus === Action.CREATE) {
        // get proposal ID from logs
      }

      // If ID Proposal is provided, check the proposal status, and either approve or execute.
      // If ID Proposal is not provided, create a new proposal

      return {} as Result<TransactionResponse>
    }
  }
}
