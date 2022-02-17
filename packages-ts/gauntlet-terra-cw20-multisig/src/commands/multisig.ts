import { Result } from '@chainlink/gauntlet-core'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { AccAddress, MsgExecuteContract } from '@terra-money/terra.js'
import { isDeepEqual } from '../lib/utils'

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
  wasm: {
    execute: {
      contract_addr: string
      funds: {
        denom: string
        amount: string
      }[]
      msg: string
    }
  }
}

enum Action {
  CREATE = 'create',
  APPROVE = 'approve',
  EXECUTE = 'execute',
  NONE = 'none',
}

type State = {
  threshold: number
  nextAction: Action
  owners: AccAddress[]
  approvers: string[]
  // https://github.com/CosmWasm/cw-plus/blob/82138f9484e538913f7faf78bc292fb14407aae8/packages/cw3/src/query.rs#L75
  currentStatus?: 'pending' | 'open' | 'rejected' | 'passed' | 'executed'
  data?: WasmMsg[]
}

export const wrapCommand = (command) => {
  return class Multisig extends TerraCommand {
    command: TerraCommand
    multisig: AccAddress
    multisigGroup: AccAddress

    static id = `${command.id}:multisig`

    constructor(flags, args) {
      super(flags, args)

      this.command = new command(flags, args)

      if (!AccAddress.validate(process.env.MULTISIG_ADDRESS)) throw new Error(`Invalid Multisig wallet address`)
      if (!AccAddress.validate(process.env.MULTISIG_GROUP)) throw new Error(`Invalid Multisig group address`)
      this.multisig = process.env.MULTISIG_ADDRESS as AccAddress
      this.multisigGroup = process.env.MULTISIG_GROUP as AccAddress
    }

    makeRawTransaction = async (signer: AccAddress, state?: State) => {
      const message = await this.command.makeRawTransaction(this.multisig)

      const operations = {
        [Action.CREATE]: this.executePropose,
        [Action.APPROVE]: this.executeApproval,
        [Action.EXECUTE]: this.executeExecution,
        [Action.NONE]: () => {
          throw new Error('No action needed')
        },
      }

      if (state.nextAction !== Action.CREATE) {
        this.require(
          await this.isSameProposal(state.data, [this.toWasmMsg(message)]),
          'The transaction generated is different from the proposal provided',
        )
      }

      return operations[state.nextAction](signer, Number(this.flags.proposal), message)
    }

    isSameProposal = (proposalMsgs: WasmMsg[], generatedMsgs: WasmMsg[]) => {
      return isDeepEqual(proposalMsgs, generatedMsgs)
    }

    toWasmMsg = (message: MsgExecuteContract): WasmMsg => {
      return {
        wasm: {
          execute: {
            contract_addr: message.contract,
            funds: message.coins.toArray().map((c) => c.toData()),
            msg: Buffer.from(JSON.stringify(message.execute_msg)).toString('base64'),
          },
        },
      }
    }

    executePropose: ProposalAction = async (signer, _, message) => {
      logger.info('Generating data for creating new proposal')
      const proposeInput = {
        propose: {
          description: command.id,
          msgs: [this.toWasmMsg(message)],
          title: command.id,
          // TODO: Set expiration time
          // latest: { never: {} }
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
      const groupState = await this.query(this.multisigGroup, {
        list_members: {},
      })
      const owners = groupState.members.map((m) => m.addr)
      const thresholdState = await this.query(this.multisig, {
        threshold: {},
      })
      const threshold = thresholdState.absolute_count.total_weight
      if (!proposalId) {
        return {
          threshold,
          nextAction: Action.CREATE,
          owners,
          approvers: [],
        }
      }
      const proposalState = await this.query(this.multisig, {
        proposal: {
          proposal_id: proposalId,
        },
      })
      const votes = await this.query(this.multisig, {
        list_votes: {
          proposal_id: proposalId,
        },
      })
      const status = proposalState.status
      const toNextAction = {
        passed: Action.EXECUTE,
        open: Action.APPROVE,
        pending: Action.APPROVE,
        rejected: Action.NONE,
        executed: Action.NONE,
      }
      return {
        threshold,
        nextAction: toNextAction[status],
        owners,
        currentStatus: status,
        data: proposalState.msgs,
        approvers: votes.votes.filter((v) => v.vote === Vote.YES).map((v) => v.voter),
      }
    }

    printPostInstructions = async (proposalId: number) => {
      const state = await this.fetchState(proposalId)
      const approvalsLeft = state.threshold - state.approvers.length
      const messages = {
        [Action.APPROVE]: `The proposal needs ${approvalsLeft} more approvals. Run the same command with the flag --proposal=${proposalId}`,
        [Action.EXECUTE]: `The proposal reached the threshold and can be executed. Run the same command with the flag --proposal=${proposalId}`,
        [Action.NONE]: `The proposal has been executed. No more actions needed`,
      }
      logger.line()
      logger.info(`${messages[state.nextAction]}`)
      logger.line()
    }

    execute = async () => {
      let proposalId = !!this.flags.proposal && Number(this.flags.proposal)
      const state = await this.fetchState(proposalId)

      if (state.nextAction === Action.NONE) {
        await this.printPostInstructions(proposalId)
        return
      }
      const rawTx = await this.makeRawTransaction(this.wallet.key.accAddress, state)

      logger.info(`Proposal State:
        - Total Owners: ${state.owners.length}
        - Owners List: ${state.owners}

        - Threshold: ${state.threshold}
        - Total Approvers: ${state.approvers.length}
        - Approvers List: ${state.approvers}

        - Next Action: ${state.nextAction.toUpperCase()}
      `)

      const actionMessage = {
        [Action.CREATE]: 'CREATING',
        [Action.APPROVE]: 'APPROVING',
        [Action.EXECUTE]: 'EXECUTING',
      }

      if (this.flags.execute) {
        await prompt(`Continue ${actionMessage[state.nextAction]} proposal?`)
        const tx = await this.signAndSend([rawTx])

        if (state.nextAction === Action.CREATE) {
          const proposalFromEvent = tx.events[0].wasm.proposal_id[0]
          logger.success(`New proposal created with ID: ${proposalFromEvent}`)
          proposalId = Number(proposalFromEvent)
        }

        await this.printPostInstructions(proposalId)

        return {
          responses: [
            {
              tx,
              contract: this.multisig,
            },
          ],
          data: {
            proposalId,
          },
        } as Result<TransactionResponse>
      }

      // TODO: Test raw message
      const msgData = Buffer.from(JSON.stringify(rawTx.execute_msg)).toString('base64')
      logger.line()
      logger.success(`Message generated succesfully for ${actionMessage[state.nextAction]} proposal`)
      logger.log()
      logger.log(msgData)
      logger.log()
      logger.line()

      return {
        responses: [
          {
            tx: {},
            contract: this.multisig,
          },
        ],
        data: {
          proposalId,
          message: msgData,
        },
      } as Result<TransactionResponse>
    }
  }
}
