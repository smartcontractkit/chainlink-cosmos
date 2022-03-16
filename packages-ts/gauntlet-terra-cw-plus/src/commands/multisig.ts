import { Result } from '@chainlink/gauntlet-core'
import { time, logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { AccAddress, MsgExecuteContract, MsgSend } from '@terra-money/terra.js'
import { isDeepEqual } from '../lib/utils'
import { fetchProposalState, makeInspectionMessage } from './inspect'
import { Vote, Cw3WasmMsg, Action, State, Cw3BankMsg, Expiration } from '../lib/types'

export const DEFAULT_VOTING_PERIOD_IN_SECS = 24 * 60 * 60

type ProposalAction = (
  signer: AccAddress,
  proposalId: number,
  message: MsgExecuteContract | MsgSend,
) => Promise<MsgExecuteContract>

export const wrapCommand = (command) => {
  return class Multisig extends TerraCommand {
    command: TerraCommand
    multisig: AccAddress

    static id = `${command.id}:multisig`

    constructor(flags, args) {
      super(flags, args)
    }

    buildCommand = async (flags, args): Promise<TerraCommand> => {
      if (!AccAddress.validate(process.env.CW3_FLEX_MULTISIG)) throw new Error(`Invalid Multisig wallet address`)
      if (!AccAddress.validate(process.env.CW4_GROUP)) throw new Error(`Invalid Multisig group address`)
      this.multisig = process.env.CW3_FLEX_MULTISIG as AccAddress

      const c = new command(flags, args) as TerraCommand
      await c.invokeMiddlewares(c, c.middlewares)
      this.command = c.buildCommand ? await c.buildCommand(flags, args) : c
      return this.command
    }

    makeRawTransaction = async (signer: AccAddress, state?: State) => {
      const message = await this.command.makeRawTransaction(this.multisig)
      await this.command.simulate(this.multisig, [message])
      logger.info(`Command simulation successful.`)

      const operations = {
        [Action.CREATE]: this.makeProposeTransaction,
        [Action.APPROVE]: this.makeAcceptTransaction,
        [Action.EXECUTE]: this.makeExecuteTransaction,
        [Action.NONE]: () => {
          throw new Error('No action needed')
        },
      }

      if (state.proposal.nextAction !== Action.CREATE) {
        this.require(
          await this.isSameProposal(state.proposal.data, [this.toMsg(message)]),
          'The transaction generated is different from the proposal provided',
        )
      }

      const proposal_id = Number(this.flags.proposal || this.flags.multisigProposal) // alias requested by eng ops
      return operations[state.proposal.nextAction](signer, Number(proposal_id), message)
    }

    isSameProposal = (proposalMsgs: (Cw3WasmMsg | Cw3BankMsg)[], generatedMsgs: (Cw3WasmMsg | Cw3BankMsg)[]) => {
      return isDeepEqual(proposalMsgs, generatedMsgs)
    }

    toMsg = (message: MsgSend | MsgExecuteContract): Cw3BankMsg | Cw3WasmMsg => {
      if (message instanceof MsgSend) return this.toBankMsg(message as MsgSend)
      if (message instanceof MsgExecuteContract) return this.toWasmMsg(message as MsgExecuteContract)
    }

    toBankMsg = (message: MsgSend): Cw3BankMsg => {
      return {
        bank: {
          send: {
            amount: message.amount.toArray().map((c) => c.toData()),
            to_address: message.to_address,
          },
        },
      }
    }

    toWasmMsg = (message: MsgExecuteContract): Cw3WasmMsg => {
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

    async parseVotingPeriod(votingPeriod: string): Promise<number> {
      const state = await this.fetchState()
      const n = Number.parseInt(votingPeriod)
      if (isNaN(n) || n < 0 || n > state.multisig.maxVotingPeriod) {
        throw new Error(
          `votingPeriod=${votingPeriod}: must be a valid duration in seconds, ` +
            `(range: [0-${state.multisig.maxVotingPeriod}], default: ${DEFAULT_VOTING_PERIOD_IN_SECS})`,
        )
      }
      return n
    }

    makeProposeTransaction: ProposalAction = async (signer, _, message) => {
      // default voting period of 24 hours, if unspecified
      const votingPeriod: number = this.flags.votingPeriod
        ? await this.parseVotingPeriod(this.flags.votingPeriod)
        : DEFAULT_VOTING_PERIOD_IN_SECS

      const msExpiration: number = Date.now() + votingPeriod * 1000 // milliseconds since 1970
      logger.info(`Generating data for creating new multisig proposal (expires at ${new Date(msExpiration)})`)
      // Expiration.at_time is a string representation of nanoseconds since 1970
      const expiration: Expiration = { at_time: (msExpiration * time.Millisecond).toString() }

      const proposeInput = {
        propose: {
          description: command.id,
          msgs: [this.toMsg(message)],
          title: command.id,
          latest: expiration,
        },
      }
      return new MsgExecuteContract(signer, this.multisig, proposeInput)
    }

    makeAcceptTransaction: ProposalAction = async (signer, proposalId) => {
      logger.info(`Generating data for approving proposal ${proposalId}`)
      const approvalInput = {
        vote: {
          vote: Vote.YES,
          proposal_id: proposalId,
        },
      }
      return new MsgExecuteContract(signer, this.multisig, approvalInput)
    }

    makeExecuteTransaction: ProposalAction = async (signer, proposalId) => {
      logger.info(`Generating data for executing multisig proposal ${proposalId}`)
      const executeInput = {
        execute: {
          proposal_id: proposalId,
        },
      }
      return new MsgExecuteContract(signer, this.multisig, executeInput)
    }

    fetchState = async (proposalId?: number): Promise<State> => {
      return fetchProposalState(this.provider)(this.multisig, proposalId)
    }

    printPostInstructions = async (proposalId: number) => {
      const state = await this.fetchState(proposalId)
      if (!state.proposal.id) {
        logger.error(`Multisig proposal ${proposalId} not found`)
        return
      }
      const approvalsLeft = state.multisig.threshold - state.proposal.approvers.length
      const messages = {
        passed: `The multisig proposal reached the threshold and can be executed. Run the same command with the flag --multisigProposal=${proposalId}`,
        open: `The multisig proposal needs ${approvalsLeft} more approvals. Run the same command with the flag --multisigProposal=${proposalId}`,
        pending: `The multisig proposal needs ${approvalsLeft} more approvals. Run the same command with the flag --multisigProposal=${proposalId}`,
        rejected: `The multisig proposal has been rejected. No actions available`,
        executed: `The multisig proposal has been executed. No more actions needed`,
      }
      logger.line()
      logger.info(`${messages[state.proposal.currentStatus]}`)
      logger.line()
    }

    execute = async () => {
      // TODO: Gauntlet core should initialize commands using `buildCommand` instead of new Command
      await this.buildCommand(this.flags, this.args)

      let proposalId = Number(this.flags.proposal || this.flags.multisigProposal) // alias requested by eng ops
      const state = await this.fetchState(proposalId)
      logger.info(makeInspectionMessage(state))

      if (state.proposal.nextAction === Action.NONE) {
        await this.printPostInstructions(proposalId)
        return
      }
      const rawTx = await this.makeRawTransaction(this.wallet.key.accAddress, state)

      const actionMessage = {
        [Action.CREATE]: 'CREATING',
        [Action.APPROVE]: 'APPROVING',
        [Action.EXECUTE]: 'EXECUTING',
      }

      if (this.flags.execute) {
        await this.command.beforeExecute(this.multisig)

        await prompt(`Continue ${actionMessage[state.proposal.nextAction]} multisig proposal?`)
        const tx = await this.signAndSend([rawTx])
        let response: Result<TransactionResponse> = {
          responses: [
            {
              tx,
              contract: this.multisig,
            },
          ],
          data: {
            proposalId,
          },
        }

        if (state.proposal.nextAction === Action.CREATE) {
          const proposalFromEvent = tx.events[0].wasm.proposal_id[0]
          logger.success(`New proposal created with multisig proposal ID: ${proposalFromEvent}`)
          proposalId = Number(proposalFromEvent)
        }

        if (state.proposal.nextAction === Action.EXECUTE && this.command.afterExecute) {
          const data = this.command.afterExecute(response)
          response = { ...response, data: { ...data } }
        }

        logger.success(`TX finished at ${tx.hash}`)
        await this.printPostInstructions(proposalId)

        return response
      }

      // TODO: Test raw message
      const msgData = Buffer.from(JSON.stringify(rawTx.execute_msg)).toString('base64')
      logger.line()
      logger.success(`Message generated succesfully for ${actionMessage[state.proposal.nextAction]} multisig proposal`)
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
